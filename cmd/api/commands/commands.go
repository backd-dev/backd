package commands

import (
	"os"

	"github.com/backd-dev/backd/internal/config"
	"github.com/fernandezvara/commandkit"
)

// getServerConfigFromContext extracts environment variables from commandkit context
// and creates ServerConfig, ensuring all env vars go through ServerConfig
func getServerConfigFromContext(ctx *commandkit.CommandContext) *config.ServerConfig {
	// Build environment map from commandkit context or fallback to os.Getenv
	env := make(map[string]string)

	// List all required environment variables
	envVars := []string{
		"BACKD_ENCRYPTION_KEY",
		"DATABASE_URL",
		"BACKD_API_PORT",
		"BACKD_FUNCTIONS_PORT",
		"BACKD_METRICS_PORT",
		"BACKD_INTERNAL_PORT",
		"BACKD_FUNCTIONS_URL",
		"BACKD_LOG_LEVEL",
		"BACKD_LOG_FORMAT",
		"BACKD_DENO_MIN_WORKERS",
		"BACKD_DENO_MAX_WORKERS",
		"BACKD_DENO_IDLE_TIMEOUT",
		"BACKD_JOB_POLL_INTERVAL",
	}

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			env[envVar] = value
		}
	}

	return config.LoadServerConfig(env)
}

// Command functions for commandkit - all declared in their respective files
