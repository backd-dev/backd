package config

import (
	"fmt"
	"os"
	"strings"
)

// EnvSubst replaces ${VAR_NAME} patterns in value with values from env map
// Returns error listing ALL unresolved variables - never partial
func EnvSubst(value string, env map[string]string) (string, error) {
	var missing []string
	result := os.Expand(value, func(key string) string {
		if val, ok := env[key]; ok {
			return val
		}
		missing = append(missing, key)
		return ""
	})
	
	if len(missing) > 0 {
		return result, fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}
	
	return result, nil
}

// EnvSubstWithOSEnvironment performs substitution using os.Getenv
// This is a convenience function for testing and CLI usage
func EnvSubstWithOSEnvironment(value string) (string, error) {
	env := make(map[string]string)
	for _, envVar := range extractEnvVars(value) {
		env[envVar] = os.Getenv(envVar)
	}
	return EnvSubst(value, env)
}

// extractEnvVars extracts all ${VAR_NAME} patterns from a string
func extractEnvVars(value string) []string {
	var vars []string
	start := -1
	
	for i, char := range value {
		if char == '$' && i+1 < len(value) && value[i+1] == '{' {
			start = i + 2
		} else if char == '}' && start != -1 {
			varName := value[start:i]
			vars = append(vars, varName)
			start = -1
		}
	}
	
	return vars
}
