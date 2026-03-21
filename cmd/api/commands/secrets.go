package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/secrets"
	"github.com/fernandezvara/commandkit"
)

// SecretsFunc is the command function for the secrets command
var SecretsFunc func(ctx *commandkit.CommandContext) error = func(ctx *commandkit.CommandContext) error {
	// Get required config-dir flag
	configDir, err := commandkit.Get[string](ctx, "config-dir")
	if err != nil {
		return fmt.Errorf("failed to get config-dir: %w", err)
	}
	if configDir == "" {
		return fmt.Errorf("config-dir is required")
	}

	// Get optional flags
	targetApp, _ := commandkit.Get[string](ctx, "app")
	dryRun, _ := commandkit.Get[bool](ctx, "dry-run")

	// Create context for database operations
	dbCtx := context.Background()

	// Phase 1: Load ServerConfig from environment
	serverCfg := getServerConfigFromContext(ctx)

	if serverCfg.EncryptionKey == "" {
		slog.Error("BACKD_ENCRYPTION_KEY is required")
		os.Exit(1)
	}

	// Phase 2: Load and validate configurations
	configSet, err := config.LoadAll(configDir, nil)
	if err != nil {
		slog.Error("failed to load configurations", "error", err)
		os.Exit(1)
	}

	if validationResults := config.ValidateAll(configSet, configDir); len(validationResults) > 0 {
		for _, result := range validationResults {
			if result.Status == config.StatusError {
				var errorMessages []string
				for _, issue := range result.Issues {
					if issue.IsError {
						errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
					}
				}
				slog.Error("configuration validation failed", "app", result.AppName, "errors", errorMessages)
				os.Exit(1)
			}
		}
	}

	// Phase 3: Initialize database and secrets instances
	dbInstance := db.NewDB(configSet, serverCfg)
	secretsInstance := secrets.NewSecrets(dbInstance, []byte(serverCfg.EncryptionKey))

	// Phase 4: Collect all secrets to validate (Phase 1)
	var allSecrets []secretToApply
	var appNames []string
	var domainNames []string

	if targetApp != "" {
		// Process specific app
		appConfig, exists := configSet.Apps[targetApp]
		if !exists {
			return fmt.Errorf("app %s not found", targetApp)
		}
		appSecrets := collectSecrets(targetApp, appConfig.Secrets)
		allSecrets = append(allSecrets, appSecrets...)
		appNames = append(appNames, targetApp)
	} else {
		// Process all apps and domains
		for appName, appConfig := range configSet.Apps {
			appSecrets := collectSecrets(appName, appConfig.Secrets)
			allSecrets = append(allSecrets, appSecrets...)
			appNames = append(appNames, appName)
		}
		for domainName := range configSet.Domains {
			domainNames = append(domainNames, domainName)
		}
	}

	// Phase 5: Validate all environment variables (Phase 1)
	var validationErrors []string
	for _, secret := range allSecrets {
		if strings.HasPrefix(secret.value, "${") && strings.HasSuffix(secret.value, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(secret.value, "${"), "}")
			if os.Getenv(envVar) == "" {
				validationErrors = append(validationErrors, fmt.Sprintf("app %s: environment variable %s not set", secret.appName, envVar))
			}
		}
	}

	if len(validationErrors) > 0 {
		slog.Error("environment variable validation failed", "errors", validationErrors)
		os.Exit(1)
	}

	if dryRun {
		slog.Info("dry-run complete: all environment variables are valid")
		return nil
	}

	// Phase 6: Apply secrets (Phase 2)
	for _, appName := range appNames {
		if err := applySecretsForApp(dbCtx, secretsInstance, appName, configSet.Apps[appName]); err != nil {
			slog.Error("failed to apply secrets for app", "app", appName, "error", err)
			// Continue with other apps
		} else {
			slog.Info("secrets applied for app", "app", appName)
		}
	}

	slog.Info("secrets apply complete")
	return nil
}

type secretToApply struct {
	appName string
	name    string
	value   string
}

func collectSecrets(appName string, secretsMap map[string]string) []secretToApply {
	var result []secretToApply
	for name, value := range secretsMap {
		result = append(result, secretToApply{
			appName: appName,
			name:    name,
			value:   value,
		})
	}
	return result
}

func applySecretsForApp(ctx context.Context, secretsInstance secrets.Secrets, appName string, appConfig *config.AppConfig) error {
	for name, value := range appConfig.Secrets {
		// Resolve environment variable if needed
		var resolvedValue string
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
			resolvedValue = os.Getenv(envVar)
		} else {
			resolvedValue = value
		}

		// Store secret using the secrets interface
		if err := secretsInstance.Set(ctx, appName, name, resolvedValue); err != nil {
			return fmt.Errorf("failed to store secret %s: %w", name, err)
		}
	}

	return nil
}
