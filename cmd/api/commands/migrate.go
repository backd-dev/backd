package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/fernandezvara/commandkit"
)

// MigrateFunc is the command function for the migrate command
var MigrateFunc func(ctx *commandkit.CommandContext) error = func(ctx *commandkit.CommandContext) error {
	// Get required config-dir flag
	configDir, err := commandkit.Get[string](ctx, "config-dir")
	if err != nil {
		return fmt.Errorf("failed to get config-dir: %w", err)
	}
	if configDir == "" {
		return fmt.Errorf("config-dir is required")
	}

	// Get optional app flag
	targetApp, _ := commandkit.Get[string](ctx, "app")

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

	// Phase 3: Initialize database instance
	dbInstance := db.NewDB(configSet, serverCfg)

	// Phase 4: Migrate domains first (for completeness)
	for domainName := range configSet.Domains {
		if err := dbInstance.Bootstrap(dbCtx, domainName, db.DBTypeDomain); err != nil {
			slog.Error("failed to bootstrap domain", "domain", domainName, "error", err)
			os.Exit(1)
		}
		slog.Info("domain bootstrapped", "domain", domainName)
	}

	// Phase 5: Migrate apps
	if targetApp != "" {
		// Migrate specific app
		appConfig, exists := configSet.Apps[targetApp]
		if !exists {
			return fmt.Errorf("app %s not found", targetApp)
		}

		if err := migrateApp(dbCtx, dbInstance, targetApp, appConfig, configDir); err != nil {
			return err
		}
	} else {
		// Migrate all apps
		for appName, appConfig := range configSet.Apps {
			if err := migrateApp(dbCtx, dbInstance, appName, appConfig, configDir); err != nil {
				slog.Error("app migration failed", "app", appName, "error", err)
				// Continue with other apps
			}
		}
	}

	slog.Info("migration complete")
	return nil
}

func migrateApp(ctx context.Context, dbInstance db.DB, appName string, appConfig *config.AppConfig, configDir string) error {
	// Bootstrap app (creates reserved tables)
	if err := dbInstance.Bootstrap(ctx, appName, db.DBTypeApp); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Run migrations
	migrationsPath := fmt.Sprintf("%s/%s/migrations", configDir, appName)
	if err := dbInstance.Migrate(ctx, appName, migrationsPath); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	slog.Info("app migrated", "app", appName)
	return nil
}
