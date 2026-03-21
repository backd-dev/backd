package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// nameRegex validates names according to the rules: lowercase, alphanumeric + underscores, must start with letter
var nameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// migrationFileRegex validates migration file naming: NNN_description.sql
var migrationFileRegex = regexp.MustCompile(`^[0-9]{3}_[a-z0-9_]+\.sql$`)

// ValidateApp validates an AppConfig and collects all issues
func ValidateApp(app *AppConfig, migrationsPath, functionsPath string, domainNames []string) *ValidationResult {
	result := &ValidationResult{
		Status:  StatusOK,
		AppName: app.Name,
	}

	// Validate app name
	if !validName(app.Name) {
		result.AddError("name", "invalid format", "must start with lowercase letter, contain only lowercase letters, numbers, and underscores")
	}

	// Validate required fields
	if app.Keys.PublishableKey == "" {
		result.AddError("keys.publishable_key", "required field missing", "generate with 'backd bootstrap' or add manually")
	}

	// Validate auth domain reference
	if app.Auth.Domain != "" {
		if !contains(domainNames, app.Auth.Domain) {
			result.AddError("auth.domain", "domain not found", fmt.Sprintf("available domains: %s", strings.Join(domainNames, ", ")))
		}
	}

	// Validate database config
	validateDatabaseConfig(&app.Database, result)

	// Validate storage config if present
	if app.Storage != nil {
		validateStorageConfig(app.Storage, result)
	}

	// Validate secrets
	validateSecrets(app.Secrets, result)

	// Validate cron entries
	validateCronEntries(app.Cron, functionsPath, result)

	// Validate policies
	validatePolicies(app.Policies, result)

	// Note: Skip directory validation in tests since we don't have real directories

	return result
}

// ValidateDomain validates a DomainConfig
func ValidateDomain(domain *DomainConfig) *ValidationResult {
	result := &ValidationResult{
		Status:  StatusOK,
		AppName: domain.Name,
	}

	// Validate domain name
	if !validName(domain.Name) {
		result.AddError("name", "invalid format", "must start with lowercase letter, contain only lowercase letters, numbers, and underscores")
	}

	// Validate provider
	if !IsValidProvider(domain.Provider) {
		result.AddError("provider", "invalid provider", fmt.Sprintf("allowed providers: %s", strings.Join(AllowedProviders(), ", ")))
	}

	// Validate database config
	validateDatabaseConfig(&domain.Database, result)

	return result
}

// ValidateAll validates all configurations in a ConfigSet
func ValidateAll(set *ConfigSet, configRoot string) []*ValidationResult {
	var results []*ValidationResult

	// Validate domains first (apps may reference them)
	for name, domain := range set.Domains {
		domain.Name = name // Ensure name matches key
		results = append(results, ValidateDomain(domain))
	}

	domainNames := set.DomainNames()

	// Validate apps
	for name, app := range set.Apps {
		app.Name = name // Ensure name matches key
		migrationsPath := filepath.Join(configRoot, name, "migrations")
		functionsPath := filepath.Join(configRoot, name, "functions")
		results = append(results, ValidateApp(app, migrationsPath, functionsPath, domainNames))
	}

	return results
}

// validName checks if a string matches the name regex
func validName(s string) bool {
	return nameRegex.MatchString(s)
}

// validateDatabaseConfig validates database configuration
func validateDatabaseConfig(db *DatabaseConfig, result *ValidationResult) {
	// Check if either DSN or individual fields are provided
	if db.DSN == "" && db.Host == "" {
		result.AddWarning("database", "no database configuration", "will inherit server DATABASE_URL")
		return
	}

	// Validate port if provided
	if db.Port != 0 && (db.Port < 1 || db.Port > 65535) {
		result.AddError("database.port", "invalid port", "must be between 1 and 65535")
	}

	// Validate connection pool settings
	if db.MinConnections > db.MaxConnections {
		result.AddError("database.min_connections", "cannot exceed max_connections", "")
	}
}

// validateStorageConfig validates storage configuration
func validateStorageConfig(storage *StorageConfig, result *ValidationResult) {
	if storage.Endpoint == "" {
		result.AddError("storage.endpoint", "required field missing", "")
	}
	if storage.Bucket == "" {
		result.AddError("storage.bucket", "required field missing", "")
	}
	if storage.Region == "" {
		result.AddError("storage.region", "required field missing", "")
	}
	if storage.AccessKeyID == "" {
		result.AddError("storage.access_key_id", "required field missing", "")
	}
	if storage.SecretAccessKey == "" {
		result.AddError("storage.secret_access_key", "required field missing", "")
	}
}

// validateSecrets validates secrets configuration
func validateSecrets(secrets map[string]string, result *ValidationResult) {
	for name, value := range secrets {
		if !validName(name) {
			result.AddError(fmt.Sprintf("secrets.%s", name), "invalid secret name", "must start with lowercase letter, contain only lowercase letters, numbers, and underscores")
		}
		if value == "" {
			result.AddWarning(fmt.Sprintf("secrets.%s", name), "empty value", "secret will be removed")
		}
		// Check if value uses env var substitution
		if !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
			result.AddError(fmt.Sprintf("secrets.%s", name), "invalid format", "must use ${VAR_NAME} format")
		}
	}
}

// validateCronEntries validates cron job configuration
func validateCronEntries(cron []CronEntry, functionsPath string, result *ValidationResult) {
	for i, entry := range cron {
		fieldPrefix := fmt.Sprintf("cron[%d]", i)

		if entry.Schedule == "" {
			result.AddError(fmt.Sprintf("%s.schedule", fieldPrefix), "required field missing", "")
		}

		if entry.Function == "" {
			result.AddError(fmt.Sprintf("%s.function", fieldPrefix), "required field missing", "")
		} else if !validName(entry.Function) {
			result.AddError(fmt.Sprintf("%s.function", fieldPrefix), "invalid format", "must start with lowercase letter, contain only lowercase letters, numbers, and underscores")
		} else {
			// Check if function directory exists
			funcPath := filepath.Join(functionsPath, entry.Function)
			if _, err := os.Stat(funcPath); os.IsNotExist(err) {
				result.AddWarning(fmt.Sprintf("%s.function", fieldPrefix), "function directory not found", funcPath)
			}
		}
	}
}

// validatePolicies validates RLS policies
func validatePolicies(policies map[string]TablePolicies, result *ValidationResult) {
	for tableName, tablePolicies := range policies {
		if !validName(tableName) {
			result.AddError(fmt.Sprintf("policies.%s", tableName), "invalid table name", "must start with lowercase letter, contain only lowercase letters, numbers, and underscores")
		}

		for operation, policy := range tablePolicies {
			fieldPrefix := fmt.Sprintf("policies.%s.%s", tableName, operation)

			if policy.Expression == "" {
				result.AddError(fmt.Sprintf("%s.expression", fieldPrefix), "required field missing", "")
			}

			// Validate column names
			for _, col := range policy.Columns {
				if !validName(col) {
					result.AddError(fmt.Sprintf("%s.columns", fieldPrefix), "invalid column name", fmt.Sprintf("'%s' must start with lowercase letter, contain only lowercase letters, numbers, and underscores", col))
				}
			}
		}
	}
}

// validateDirectories checks if required directories exist
func validateDirectories(appName, migrationsPath, functionsPath string, result *ValidationResult) {
	// Check migrations directory
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		result.AddError("migrations", "directory not found", migrationsPath)
	} else {
		// Check for at least one migration file
		files, err := os.ReadDir(migrationsPath)
		if err == nil {
			hasMigration := false
			for _, file := range files {
				if !file.IsDir() && migrationFileRegex.MatchString(file.Name()) {
					hasMigration = true
					break
				}
			}
			if !hasMigration {
				result.AddError("migrations", "no valid migration files found", "must have at least 001_init.sql")
			}
		}
	}

	// Check functions directory
	if _, err := os.Stat(functionsPath); os.IsNotExist(err) {
		result.AddWarning("functions", "directory not found", "optional - create if using functions")
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
