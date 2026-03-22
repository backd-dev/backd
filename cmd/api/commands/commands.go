package commands

import (
	"os"
	"strings"

	"github.com/backd-dev/backd/internal/config"
	"github.com/fernandezvara/commandkit"
)

// getServerConfigFromContext extracts environment variables from commandkit context
// and creates ServerConfig, ensuring all env vars go through ServerConfig
func getServerConfigFromContext(ctx *commandkit.CommandContext) *config.ServerConfig {
	// Build environment map from all environment variables for env substitution
	env := make(map[string]string)

	// Copy all environment variables to ensure ${VAR} substitution works for any variable
	for _, envVar := range os.Environ() {
		if key, value, found := strings.Cut(envVar, "="); found {
			env[key] = value
		}
	}

	return config.LoadServerConfig(env)
}

// Command functions for commandkit - all declared in their respective files
