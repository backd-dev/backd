package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/secrets"
	"github.com/fernandezvara/commandkit"
)

// FunctionsFunc is the command function for the functions command
var FunctionsFunc func(ctx *commandkit.CommandContext) error = func(ctx *commandkit.CommandContext) error {
	// Get required config-dir flag
	configDir, err := commandkit.Get[string](ctx, "config-dir")
	if err != nil {
		return fmt.Errorf("failed to get config-dir: %w", err)
	}
	if configDir == "" {
		return fmt.Errorf("config-dir is required")
	}

	// Get mode flag
	mode, _ := commandkit.Get[string](ctx, "mode")
	if mode == "" {
		mode = "both"
	}

	// Create context for operations
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

	// Phase 3: Initialize core instances
	dbInstance := db.NewDB(configSet, serverCfg)
	secretsInstance := secrets.NewSecrets(dbInstance, []byte(serverCfg.EncryptionKey))

	// Phase 4: App-specific provisioning
	slog.Info("provisioning apps for functions service")
	for appName, appConfig := range configSet.Apps {
		if err := provisionAppForFunctions(dbCtx, dbInstance, secretsInstance, appName, appConfig, configDir); err != nil {
			slog.Error("app functions provisioning failed", "app", appName, "error", err)
			os.Exit(1)
		}
		slog.Info("app provisioned for functions", "app", appName)
	}

	// Phase 5: Start functions service based on mode
	slog.Info("starting functions service", "mode", mode)

	switch mode {
	case "functions":
		return startFunctionsHTTP(serverCfg)
	case "worker":
		return startFunctionsWorker(serverCfg, dbInstance, secretsInstance, configSet)
	case "both":
		// Start both in parallel
		go func() {
			if err := startFunctionsHTTP(serverCfg); err != nil {
				slog.Error("functions HTTP service failed", "error", err)
			}
		}()
		return startFunctionsWorker(serverCfg, dbInstance, secretsInstance, configSet)
	default:
		return fmt.Errorf("invalid mode: %s (must be functions, worker, or both)", mode)
	}
}

func provisionAppForFunctions(ctx context.Context, dbInstance db.DB, secretsInstance secrets.Secrets, appName string, appConfig *config.AppConfig, configDir string) error {
	// Verify app database exists
	pool, err := dbInstance.Pool(appName)
	if err != nil {
		return fmt.Errorf("failed to get database pool for app %s: %w", appName, err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed for app %s: %w", appName, err)
	}

	// Load function files from functions/ directory into memory index
	if err := loadFunctionFiles(appName, configDir); err != nil {
		return fmt.Errorf("failed to load function files for app %s: %w", appName, err)
	}

	// Register cron schedules from app.yaml
	if err := registerCronSchedules(appName, appConfig); err != nil {
		return fmt.Errorf("failed to register cron schedules for app %s: %w", appName, err)
	}

	return nil
}

func loadFunctionFiles(appName, configDir string) error {
	// TODO: Implement function file loading from functions/ directory
	// This should index .js/.ts files and prepare them for Deno execution
	slog.Info("function files indexed", "app", appName, "count", 0) // Placeholder
	return nil
}

func registerCronSchedules(appName string, appConfig *config.AppConfig) error {
	// TODO: Implement cron schedule registration
	// This should register schedules from appConfig.Cron into the scheduler
	cronCount := len(appConfig.Cron)
	slog.Info("cron schedules registered", "app", appName, "count", cronCount)
	return nil
}

func startFunctionsHTTP(serverCfg *config.ServerConfig) error {
	// TODO: Implement functions HTTP server on BACKD_FUNCTIONS_PORT
	slog.Info("functions HTTP service not yet implemented", "port", serverCfg.FunctionsPort)
	return fmt.Errorf("functions HTTP service not yet implemented")
}

func startFunctionsWorker(serverCfg *config.ServerConfig, dbInstance db.DB, secretsInstance secrets.Secrets, configSet *config.ConfigSet) error {
	// TODO: Implement job worker
	slog.Info("functions worker service not yet implemented")
	return fmt.Errorf("functions worker service not yet implemented")
}
