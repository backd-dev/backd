package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"gopkg.in/yaml.v3"
)

// Load loads a single app.yaml file
func Load(path string, env map[string]string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Apply environment variable substitution to secrets and storage
	if err := substituteEnvVars(&config, env); err != nil {
		return nil, fmt.Errorf("environment substitution failed: %w", err)
	}

	// Apply default values
	config.ApplyDefaults()

	return &config, nil
}

// LoadDomain loads a single domain.yaml file
func LoadDomain(path string, env map[string]string) (*DomainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var config DomainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Apply environment variable substitution to database credentials
	if err := substituteDomainEnvVars(&config, env); err != nil {
		return nil, fmt.Errorf("environment substitution failed: %w", err)
	}

	// Apply default values
	config.ApplyDefaults()

	return &config, nil
}

// LoadAll scans a config root and loads all app and domain configurations
func LoadAll(root string, env map[string]string) (*ConfigSet, error) {
	configSet := NewConfigSet()

	// Load domain configurations first
	domainFiles, err := findDomainFiles(root)
	if err != nil {
		return nil, fmt.Errorf("failed to find domain files: %w", err)
	}

	for _, domainFile := range domainFiles {
		domain, err := LoadDomain(domainFile, env)
		if err != nil {
			return nil, fmt.Errorf("failed to load domain %s: %w", domainFile, err)
		}
		configSet.AddDomain(domain.Name, domain)
	}

	// Load app configurations
	appFiles, err := findAppFiles(root)
	if err != nil {
		return nil, fmt.Errorf("failed to find app files: %w", err)
	}

	for _, appFile := range appFiles {
		app, err := Load(appFile, env)
		if err != nil {
			return nil, fmt.Errorf("failed to load app %s: %w", appFile, err)
		}
		configSet.AddApp(app.Name, app)
	}

	return configSet, nil
}

// ScanRoot scans a config root and returns lists of app and domain files
func ScanRoot(root string) ([]string, []string, error) {
	appFiles, err := findAppFiles(root)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find app files: %w", err)
	}

	domainFiles, err := findDomainFiles(root)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find domain files: %w", err)
	}

	return appFiles, domainFiles, nil
}

// findAppFiles finds all app.yaml files in the config root
func findAppFiles(root string) ([]string, error) {
	var appFiles []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read config root %s: %w", root, err)
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), "_") {
			appYaml := filepath.Join(root, entry.Name(), "app.yaml")
			if _, err := os.Stat(appYaml); err == nil {
				appFiles = append(appFiles, appYaml)
			}
		}
	}

	return appFiles, nil
}

// findDomainFiles finds all domain.yaml files in the _domains directory
func findDomainFiles(root string) ([]string, error) {
	var domainFiles []string

	domainsDir := filepath.Join(root, "_domains")
	if _, err := os.Stat(domainsDir); os.IsNotExist(err) {
		return domainFiles, nil // No domains directory is fine
	}

	entries, err := os.ReadDir(domainsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read domains directory %s: %w", domainsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			domainYaml := filepath.Join(domainsDir, entry.Name(), "domain.yaml")
			if _, err := os.Stat(domainYaml); err == nil {
				domainFiles = append(domainFiles, domainYaml)
			}
		}
	}

	return domainFiles, nil
}

// substituteEnvVars performs environment variable substitution for app config
func substituteEnvVars(config *AppConfig, env map[string]string) error {
	// Substitute in secrets
	for name, value := range config.Secrets {
		substituted, err := EnvSubst(value, env)
		if err != nil {
			return fmt.Errorf("secrets.%s: %w", name, err)
		}
		config.Secrets[name] = substituted
	}

	// Substitute in storage config if present
	if config.Storage != nil {
		if config.Storage.AccessKeyID != "" {
			substituted, err := EnvSubst(config.Storage.AccessKeyID, env)
			if err != nil {
				return fmt.Errorf("storage.access_key_id: %w", err)
			}
			config.Storage.AccessKeyID = substituted
		}

		if config.Storage.SecretAccessKey != "" {
			substituted, err := EnvSubst(config.Storage.SecretAccessKey, env)
			if err != nil {
				return fmt.Errorf("storage.secret_access_key: %w", err)
			}
			config.Storage.SecretAccessKey = substituted
		}
	}

	// Substitute in database config
	if config.Database.Password != "" {
		substituted, err := EnvSubst(config.Database.Password, env)
		if err != nil {
			return fmt.Errorf("database.password: %w", err)
		}
		config.Database.Password = substituted
	}

	if config.Database.DSN != "" {
		substituted, err := EnvSubst(config.Database.DSN, env)
		if err != nil {
			return fmt.Errorf("database.dsn: %w", err)
		}
		config.Database.DSN = substituted
	}

	return nil
}

// substituteDomainEnvVars performs environment variable substitution for domain config
func substituteDomainEnvVars(config *DomainConfig, env map[string]string) error {
	// Substitute in database config
	if config.Database.Password != "" {
		substituted, err := EnvSubst(config.Database.Password, env)
		if err != nil {
			return fmt.Errorf("database.password: %w", err)
		}
		config.Database.Password = substituted
	}

	if config.Database.DSN != "" {
		substituted, err := EnvSubst(config.Database.DSN, env)
		if err != nil {
			return fmt.Errorf("database.dsn: %w", err)
		}
		config.Database.DSN = substituted
	}

	return nil
}
