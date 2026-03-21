package config

import "time"

// DomainConfig represents a domain configuration for shared authentication
type DomainConfig struct {
	Name             string        `yaml:"name"`
	Provider         string        `yaml:"provider"`
	SessionExpiry    time.Duration `yaml:"session_expiry"`
	AllowRegistration bool          `yaml:"allow_registration"`
	Database         DatabaseConfig `yaml:"database"`
}

// ApplyDefaults sets default values for DomainConfig
func (c *DomainConfig) ApplyDefaults() {
	if c.SessionExpiry == 0 {
		c.SessionExpiry = 24 * time.Hour
	}
	c.Database.ApplyDefaults()
}

// AllowedProviders returns the list of allowed authentication providers
func AllowedProviders() []string {
	return []string{"password"}
}

// IsValidProvider checks if the provider is allowed
func IsValidProvider(provider string) bool {
	for _, p := range AllowedProviders() {
		if p == provider {
			return true
		}
	}
	return false
}
