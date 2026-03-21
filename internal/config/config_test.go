package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestEnvSubst(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		env      map[string]string
		expected string
		hasError bool
	}{
		{
			name:     "simple substitution",
			value:    "hello ${WORLD}",
			env:      map[string]string{"WORLD": "world"},
			expected: "hello world",
			hasError: false,
		},
		{
			name:     "multiple substitutions",
			value:    "${USER}@${HOST}",
			env:      map[string]string{"USER": "alice", "HOST": "example.com"},
			expected: "alice@example.com",
			hasError: false,
		},
		{
			name:     "missing variable",
			value:    "hello ${MISSING}",
			env:      map[string]string{},
			expected: "hello ",
			hasError: true,
		},
		{
			name:     "no substitution",
			value:    "plain text",
			env:      map[string]string{},
			expected: "plain text",
			hasError: false,
		},
		{
			name:     "empty env map",
			value:    "${VAR}",
			env:      nil,
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EnvSubst(tt.value, tt.env)

			if tt.hasError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.hasError && result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", "app", true},
		{"valid with underscore", "my_app", true},
		{"valid with numbers", "app2", true},
		{"valid mixed", "my_app_123", true},
		{"invalid uppercase", "MyApp", false},
		{"invalid hyphen", "my-app", false},
		{"invalid start with number", "123app", false},
		{"invalid start with underscore", "_app", false},
		{"invalid empty", "", false},
		{"invalid special chars", "app@app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLoadServerConfig(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected *ServerConfig
	}{
		{
			name: "default values",
			env:  map[string]string{},
			expected: &ServerConfig{
				APIPort:         8080,
				FunctionsPort:   8081,
				MetricsPort:     9090,
				InternalPort:    9191,
				FunctionsURL:    "http://functions:8081",
				LogLevel:        "info",
				LogFormat:       "json",
				DenoMinWorkers:  2,
				DenoMaxWorkers:  10,
				DenoIdleTimeout: 5 * time.Minute,
				JobPollInterval: time.Second,
			},
		},
		{
			name: "custom values",
			env: map[string]string{
				"BACKD_ENCRYPTION_KEY":   "test-key",
				"BACKD_API_PORT":         "9000",
				"BACKD_LOG_LEVEL":        "debug",
				"BACKD_DENO_MIN_WORKERS": "5",
			},
			expected: &ServerConfig{
				EncryptionKey:   "test-key",
				APIPort:         9000,
				FunctionsPort:   8081,
				MetricsPort:     9090,
				InternalPort:    9191,
				FunctionsURL:    "http://functions:8081",
				LogLevel:        "debug",
				LogFormat:       "json",
				DenoMinWorkers:  5,
				DenoMaxWorkers:  10,
				DenoIdleTimeout: 5 * time.Minute,
				JobPollInterval: time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LoadServerConfig(tt.env)

			if result.EncryptionKey != tt.expected.EncryptionKey {
				t.Errorf("EncryptionKey: expected %q, got %q", tt.expected.EncryptionKey, result.EncryptionKey)
			}
			if result.APIPort != tt.expected.APIPort {
				t.Errorf("APIPort: expected %d, got %d", tt.expected.APIPort, result.APIPort)
			}
			if result.LogLevel != tt.expected.LogLevel {
				t.Errorf("LogLevel: expected %q, got %q", tt.expected.LogLevel, result.LogLevel)
			}
			if result.DenoMinWorkers != tt.expected.DenoMinWorkers {
				t.Errorf("DenoMinWorkers: expected %d, got %d", tt.expected.DenoMinWorkers, result.DenoMinWorkers)
			}
		})
	}
}

func TestAppConfigApplyDefaults(t *testing.T) {
	config := &AppConfig{
		Database: DatabaseConfig{},
		Auth:     AuthConfig{},
		Jobs:     JobsConfig{},
	}

	config.ApplyDefaults()

	if config.Database.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", config.Database.Port)
	}
	if config.Database.SSLMode != "disable" {
		t.Errorf("Expected ssl_mode 'disable', got %s", config.Database.SSLMode)
	}
	if config.Auth.SessionExpiry != 24*time.Hour {
		t.Errorf("Expected session expiry 24h, got %v", config.Auth.SessionExpiry)
	}
	if config.Jobs.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", config.Jobs.MaxAttempts)
	}
}

func TestDomainConfigApplyDefaults(t *testing.T) {
	config := &DomainConfig{
		Database: DatabaseConfig{},
	}

	config.ApplyDefaults()

	if config.SessionExpiry != 24*time.Hour {
		t.Errorf("Expected session expiry 24h, got %v", config.SessionExpiry)
	}
	if config.Database.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", config.Database.Port)
	}
}

func TestValidateApp(t *testing.T) {
	tests := []struct {
		name           string
		app            *AppConfig
		domainNames    []string
		expectedStatus Status
		expectedIssues int
	}{
		{
			name: "valid app",
			app: &AppConfig{
				Name: "my_app",
				Keys: KeysConfig{
					PublishableKey: "test-key",
				},
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
			domainNames:    []string{},
			expectedStatus: StatusOK,
			expectedIssues: 0,
		},
		{
			name: "invalid name",
			app: &AppConfig{
				Name: "MyApp",
				Keys: KeysConfig{
					PublishableKey: "test-key",
				},
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
			domainNames:    []string{},
			expectedStatus: StatusError,
			expectedIssues: 1,
		},
		{
			name: "missing publishable key",
			app: &AppConfig{
				Name: "my_app",
				Keys: KeysConfig{},
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
			domainNames:    []string{},
			expectedStatus: StatusError,
			expectedIssues: 1,
		},
		{
			name: "unknown domain",
			app: &AppConfig{
				Name: "my_app",
				Keys: KeysConfig{
					PublishableKey: "test-key",
				},
				Auth: AuthConfig{
					Domain: "unknown",
				},
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
			domainNames:    []string{"company"},
			expectedStatus: StatusError,
			expectedIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateApp(tt.app, "/tmp/migrations", "/tmp/functions", tt.domainNames)

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, result.Status)
			}
			if len(result.Issues) != tt.expectedIssues {
				t.Errorf("Expected %d issues, got %d", tt.expectedIssues, len(result.Issues))
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name           string
		domain         *DomainConfig
		expectedStatus Status
		expectedIssues int
	}{
		{
			name: "valid domain",
			domain: &DomainConfig{
				Name:     "company",
				Provider: "password",
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
			expectedStatus: StatusOK,
			expectedIssues: 0,
		},
		{
			name: "invalid name",
			domain: &DomainConfig{
				Name:     "Company",
				Provider: "password",
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
			expectedStatus: StatusError,
			expectedIssues: 1,
		},
		{
			name: "invalid provider",
			domain: &DomainConfig{
				Name:     "company",
				Provider: "invalid",
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
			expectedStatus: StatusError,
			expectedIssues: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDomain(tt.domain)

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, result.Status)
			}
			if len(result.Issues) != tt.expectedIssues {
				t.Errorf("Expected %d issues, got %d", tt.expectedIssues, len(result.Issues))
			}
		})
	}
}

func TestConfigSet(t *testing.T) {
	cs := NewConfigSet()

	// Test empty set
	if len(cs.Apps) != 0 || len(cs.Domains) != 0 {
		t.Errorf("Expected empty config set")
	}

	// Test adding apps and domains
	app := &AppConfig{Name: "test_app"}
	domain := &DomainConfig{Name: "test_domain"}

	cs.AddApp("test_app", app)
	cs.AddDomain("test_domain", domain)

	if len(cs.Apps) != 1 || len(cs.Domains) != 1 {
		t.Errorf("Expected 1 app and 1 domain")
	}

	// Test retrieval
	retrievedApp, ok := cs.GetApp("test_app")
	if !ok || retrievedApp.Name != "test_app" {
		t.Errorf("Failed to retrieve app")
	}

	retrievedDomain, ok := cs.GetDomain("test_domain")
	if !ok || retrievedDomain.Name != "test_domain" {
		t.Errorf("Failed to retrieve domain")
	}

	// Test name lists
	appNames := cs.AppNames()
	domainNames := cs.DomainNames()

	if len(appNames) != 1 || appNames[0] != "test_app" {
		t.Errorf("Unexpected app names: %v", appNames)
	}

	if len(domainNames) != 1 || domainNames[0] != "test_domain" {
		t.Errorf("Unexpected domain names: %v", domainNames)
	}
}

func TestLoadAppConfig(t *testing.T) {
	// Create temporary directory and config file
	tempDir := t.TempDir()
	appYaml := `name: test_app
description: "Test application"
keys:
  publishable_key: test-key-123
database:
  host: localhost
  port: 5432
  user: test
  password: ${DB_PASSWORD}
secrets:
  API_KEY: ${API_KEY}
`

	appYamlPath := filepath.Join(tempDir, "app.yaml")
	err := os.WriteFile(appYamlPath, []byte(appYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	env := map[string]string{
		"DB_PASSWORD": "secret123",
		"API_KEY":     "api-key-456",
	}

	config, err := Load(appYamlPath, env)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Name != "test_app" {
		t.Errorf("Expected name 'test_app', got %s", config.Name)
	}
	if config.Keys.PublishableKey != "test-key-123" {
		t.Errorf("Expected publishable key 'test-key-123', got %s", config.Keys.PublishableKey)
	}
	if config.Database.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", config.Database.Host)
	}
	if config.Database.Password != "secret123" {
		t.Errorf("Expected password 'secret123', got %s", config.Database.Password)
	}
	if config.Secrets["API_KEY"] != "api-key-456" {
		t.Errorf("Expected API_KEY 'api-key-456', got %s", config.Secrets["API_KEY"])
	}
}

func TestAllowedProviders(t *testing.T) {
	providers := AllowedProviders()
	if len(providers) != 1 || providers[0] != "password" {
		t.Errorf("Expected only 'password' provider, got %v", providers)
	}

	if !IsValidProvider("password") {
		t.Errorf("Expected 'password' to be valid")
	}

	if IsValidProvider("invalid") {
		t.Errorf("Expected 'invalid' to be invalid")
	}
}

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{Status: StatusOK}

	// Test adding errors and warnings
	result.AddError("field1", "error message", "hint")
	result.AddWarning("field2", "warning message", "hint")

	if result.Status != StatusError {
		t.Errorf("Expected status Error, got %v", result.Status)
	}

	if !result.HasErrors() {
		t.Errorf("Expected HasErrors to return true")
	}

	if result.ErrorCount() != 1 {
		t.Errorf("Expected 1 error, got %d", result.ErrorCount())
	}

	if result.WarningCount() != 1 {
		t.Errorf("Expected 1 warning, got %d", result.WarningCount())
	}

	// Test CLI formatting
	output := result.FormatForCLI()
	if output == "" {
		t.Errorf("Expected non-empty CLI output")
	}
}

func TestLoadDomain(t *testing.T) {
	// Create temporary directory and config file
	tempDir := t.TempDir()
	domainYaml := `name: test_domain
provider: password
database:
  host: localhost
  port: 5432
  user: domain_user
  password: ${DOMAIN_PASSWORD}
`

	domainYamlPath := filepath.Join(tempDir, "domain.yaml")
	err := os.WriteFile(domainYamlPath, []byte(domainYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to write test domain config: %v", err)
	}

	env := map[string]string{
		"DOMAIN_PASSWORD": "domain-secret",
	}

	config, err := LoadDomain(domainYamlPath, env)
	if err != nil {
		t.Fatalf("Failed to load domain config: %v", err)
	}

	if config.Name != "test_domain" {
		t.Errorf("Expected name 'test_domain', got %s", config.Name)
	}
	if config.Provider != "password" {
		t.Errorf("Expected provider 'password', got %s", config.Provider)
	}
	if config.Database.Password != "domain-secret" {
		t.Errorf("Expected password 'domain-secret', got %s", config.Database.Password)
	}
}

func TestValidateAll(t *testing.T) {
	cs := NewConfigSet()

	// Add valid app and domain
	app := &AppConfig{
		Name:     "test_app",
		Keys:     KeysConfig{PublishableKey: "test-key"},
		Database: DatabaseConfig{Host: "localhost"},
	}
	domain := &DomainConfig{
		Name:     "test_domain",
		Provider: "password",
		Database: DatabaseConfig{Host: "localhost"},
	}

	cs.AddApp("test_app", app)
	cs.AddDomain("test_domain", domain)

	results := ValidateAll(cs, "/tmp")

	// Should have 2 results (one for app, one for domain)
	if len(results) != 2 {
		t.Errorf("Expected 2 validation results, got %d", len(results))
	}

	// Both should be OK
	for _, result := range results {
		if result.Status != StatusOK {
			t.Errorf("Expected OK status, got %v", result.Status)
		}
	}
}

func TestEnvSubstWithOSEnvironment(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result, err := EnvSubstWithOSEnvironment("hello ${TEST_VAR}")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "hello test_value"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExtractEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{"single var", "${VAR}", []string{"VAR"}},
		{"multiple vars", "${USER}@${HOST}", []string{"USER", "HOST"}},
		{"no vars", "plain text", []string{}},
		{"mixed", "prefix_${VAR}_suffix", []string{"VAR"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEnvVars(tt.value)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d vars, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if i < len(result) && result[i] != expected {
					t.Errorf("Expected %q, got %q", expected, result[i])
				}
			}
		})
	}
}

func TestScanRoot(t *testing.T) {
	tempDir := t.TempDir()

	// Create app directory structure
	appDir := filepath.Join(tempDir, "test_app")
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	appYaml := filepath.Join(appDir, "app.yaml")
	err = os.WriteFile(appYaml, []byte("name: test_app"), 0644)
	if err != nil {
		t.Fatalf("Failed to write app.yaml: %v", err)
	}

	// Create domain directory structure
	domainsDir := filepath.Join(tempDir, "_domains", "test_domain")
	err = os.MkdirAll(domainsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create domain dir: %v", err)
	}

	domainYaml := filepath.Join(domainsDir, "domain.yaml")
	err = os.WriteFile(domainYaml, []byte("name: test_domain\nprovider: password"), 0644)
	if err != nil {
		t.Fatalf("Failed to write domain.yaml: %v", err)
	}

	appFiles, domainFiles, err := ScanRoot(tempDir)
	if err != nil {
		t.Fatalf("Failed to scan root: %v", err)
	}

	if len(appFiles) != 1 {
		t.Errorf("Expected 1 app file, got %d", len(appFiles))
	}

	if len(domainFiles) != 1 {
		t.Errorf("Expected 1 domain file, got %d", len(domainFiles))
	}
}

func TestLoadAll(t *testing.T) {
	tempDir := t.TempDir()

	// Create app directory structure
	appDir := filepath.Join(tempDir, "test_app")
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	appYaml := filepath.Join(appDir, "app.yaml")
	err = os.WriteFile(appYaml, []byte("name: test_app\nkeys:\n  publishable_key: test-key"), 0644)
	if err != nil {
		t.Fatalf("Failed to write app.yaml: %v", err)
	}

	// Create domain directory structure
	domainsDir := filepath.Join(tempDir, "_domains", "test_domain")
	err = os.MkdirAll(domainsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create domain dir: %v", err)
	}

	domainYaml := filepath.Join(domainsDir, "domain.yaml")
	err = os.WriteFile(domainYaml, []byte("name: test_domain\nprovider: password"), 0644)
	if err != nil {
		t.Fatalf("Failed to write domain.yaml: %v", err)
	}

	configSet, err := LoadAll(tempDir, map[string]string{})
	if err != nil {
		t.Fatalf("Failed to load all configs: %v", err)
	}

	if len(configSet.Apps) != 1 {
		t.Errorf("Expected 1 app, got %d", len(configSet.Apps))
	}

	if len(configSet.Domains) != 1 {
		t.Errorf("Expected 1 domain, got %d", len(configSet.Domains))
	}

	app, ok := configSet.GetApp("test_app")
	if !ok || app.Name != "test_app" {
		t.Errorf("Failed to get test_app")
	}

	domain, ok := configSet.GetDomain("test_domain")
	if !ok || domain.Name != "test_domain" {
		t.Errorf("Failed to get test_domain")
	}
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusOK, "OK"},
		{StatusWarn, "WARN"},
		{StatusError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValidateStorageConfig(t *testing.T) {
	result := &ValidationResult{Status: StatusOK}

	// Test valid storage config
	storage := &StorageConfig{
		Endpoint:        "https://s3.amazonaws.com",
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	}

	validateStorageConfig(storage, result)
	if result.Status != StatusOK {
		t.Errorf("Expected OK status for valid storage config")
	}

	// Test invalid storage config
	result = &ValidationResult{Status: StatusOK}
	storage = &StorageConfig{}

	validateStorageConfig(storage, result)
	if result.Status != StatusError {
		t.Errorf("Expected Error status for invalid storage config")
	}

	if result.ErrorCount() != 5 {
		t.Errorf("Expected 5 errors for invalid storage config, got %d", result.ErrorCount())
	}
}

func TestValidateSecrets(t *testing.T) {
	result := &ValidationResult{Status: StatusOK}

	// Test valid secrets
	secrets := map[string]string{
		"api_key": "${API_KEY}",
		"db_pass": "${DB_PASS}",
	}

	validateSecrets(secrets, result)
	if result.Status != StatusOK {
		t.Errorf("Expected OK status for valid secrets, got %v", result.Status)
		for _, issue := range result.Issues {
			t.Logf("Issue: %s - %s", issue.Field, issue.Message)
		}
	}

	// Test invalid secrets
	result = &ValidationResult{Status: StatusOK}
	secrets = map[string]string{
		"invalid-name": "${API_KEY}", // Invalid name (hyphen)
		"empty_value":  "",           // Empty value
	}

	validateSecrets(secrets, result)
	if result.ErrorCount() != 1 {
		t.Errorf("Expected 1 error for invalid secrets, got %d", result.ErrorCount())
		for _, issue := range result.Issues {
			t.Logf("Issue: %s - %s", issue.Field, issue.Message)
		}
	}

	if result.WarningCount() != 1 {
		t.Errorf("Expected 1 warning for empty secret, got %d", result.WarningCount())
	}
}

func TestValidateCronEntries(t *testing.T) {
	result := &ValidationResult{Status: StatusOK}

	// Test valid cron entries
	cron := []CronEntry{
		{
			Schedule: "0 3 * * *",
			Function: "cleanup",
		},
		{
			Schedule: "*/10 * * * *",
			Function: "process_pending",
			Payload:  map[string]any{"mode": "full"},
		},
	}

	validateCronEntries(cron, "/tmp/functions", result)
	if result.Status != StatusWarn {
		t.Errorf("Expected WARN status for valid cron entries with missing directories, got %v", result.Status)
	}

	if result.WarningCount() != 2 {
		t.Errorf("Expected 2 warnings for missing function directories, got %d", result.WarningCount())
	}

	if result.ErrorCount() != 0 {
		t.Errorf("Expected 0 errors, got %d", result.ErrorCount())
	}

	// Test invalid cron entries
	result = &ValidationResult{Status: StatusOK}
	cron = []CronEntry{
		{
			Schedule: "", // Missing schedule
			Function: "cleanup",
		},
		{
			Schedule: "0 3 * * *",
			Function: "", // Missing function
		},
		{
			Schedule: "0 3 * * *",
			Function: "invalid-function", // Invalid name
		},
	}

	validateCronEntries(cron, "/tmp/functions", result)
	if result.ErrorCount() != 3 {
		t.Errorf("Expected 3 errors for invalid cron entries, got %d", result.ErrorCount())
	}
}

func TestValidatePolicies(t *testing.T) {
	result := &ValidationResult{Status: StatusOK}

	// Test valid policies
	policies := map[string]TablePolicies{
		"posts": {
			"SELECT": {
				Expression: "row.deleted_at == null",
				Columns:    []string{"id", "title"},
			},
			"INSERT": {
				Expression: "auth.authenticated",
				Defaults:   map[string]string{"user_id": "auth.uid"},
			},
		},
	}

	validatePolicies(policies, result)
	if result.Status != StatusOK {
		t.Errorf("Expected OK status for valid policies")
	}

	// Test invalid policies
	result = &ValidationResult{Status: StatusOK}
	policies = map[string]TablePolicies{
		"invalid-table": { // Invalid table name
			"SELECT": {
				Expression: "row.deleted_at == null",
			},
		},
		"posts": {
			"SELECT": {
				Expression: "", // Missing expression
			},
			"INSERT": {
				Expression: "auth.authenticated",
				Columns:    []string{"invalid-column"}, // Invalid column name
			},
		},
	}

	validatePolicies(policies, result)
	if result.ErrorCount() != 3 {
		t.Errorf("Expected 3 errors for invalid policies, got %d", result.ErrorCount())
	}
}

func TestValidateDatabaseConfig(t *testing.T) {
	result := &ValidationResult{Status: StatusOK}

	// Test valid database config
	db := &DatabaseConfig{
		Host: "localhost",
		Port: 5432,
	}

	validateDatabaseConfig(db, result)
	if result.Status != StatusOK {
		t.Errorf("Expected OK status for valid database config")
	}

	// Test database config with invalid port
	result = &ValidationResult{Status: StatusOK}
	db = &DatabaseConfig{
		Host: "localhost",
		Port: 99999, // Invalid port
	}

	validateDatabaseConfig(db, result)
	if result.ErrorCount() != 1 {
		t.Errorf("Expected 1 error for invalid port, got %d", result.ErrorCount())
	}

	// Test database config with invalid connection pool
	result = &ValidationResult{Status: StatusOK}
	db = &DatabaseConfig{
		Host:           "localhost",
		Port:           5432,
		MinConnections: 10,
		MaxConnections: 5, // Min > Max
	}

	validateDatabaseConfig(db, result)
	if result.ErrorCount() != 1 {
		t.Errorf("Expected 1 error for invalid connection pool, got %d", result.ErrorCount())
	}
}

func TestStorageConfigApplyDefaults(t *testing.T) {
	config := &StorageConfig{}
	config.ApplyDefaults()

	if config.PresignExpiry != time.Hour {
		t.Errorf("Expected presign expiry 1h, got %v", config.PresignExpiry)
	}
}

func TestHasWarnings(t *testing.T) {
	result := &ValidationResult{Status: StatusOK}

	// Should not have warnings initially
	if result.HasWarnings() {
		t.Errorf("Expected HasWarnings to return false initially")
	}

	// Add a warning
	result.AddWarning("field", "warning message", "")

	if !result.HasWarnings() {
		t.Errorf("Expected HasWarnings to return true after adding warning")
	}
}

func TestGetPort(t *testing.T) {
	tests := []struct {
		value    string
		defaults int
		expected int
	}{
		{"", 8080, 8080},
		{"9000", 8080, 9000},
		{"invalid", 8080, 8080},
		{"99999", 8080, 8080}, // Invalid port
		{"-1", 8080, 8080},    // Invalid port
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := getPort(tt.value, tt.defaults)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	tests := []struct {
		value    string
		defaults time.Duration
		expected time.Duration
	}{
		{"", 5 * time.Minute, 5 * time.Minute},
		{"1h", 5 * time.Minute, time.Hour},
		{"invalid", 5 * time.Minute, 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := getDuration(map[string]string{"DURATION": tt.value}, "DURATION", tt.defaults)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSlicesContains(t *testing.T) {
	sl := []string{"apple", "banana", "cherry"}

	if !slices.Contains(sl, "apple") {
		t.Errorf("Expected slices.Contains to return true for 'apple'")
	}

	if slices.Contains(sl, "orange") {
		t.Errorf("Expected slices.Contains to return false for 'orange'")
	}

	if slices.Contains([]string{}, "anything") {
		t.Errorf("Expected slices.Contains to return false for empty slice")
	}
}

func TestStatusStringUnknown(t *testing.T) {
	var unknownStatus Status = 999
	result := unknownStatus.String()
	if result != "UNKNOWN" {
		t.Errorf("Expected 'UNKNOWN', got %s", result)
	}
}

func TestLoadWithError(t *testing.T) {
	// Test loading non-existent file
	_, err := Load("/non/existent/file.yaml", map[string]string{})
	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}

	// Test loading invalid YAML
	tempDir := t.TempDir()
	invalidYaml := filepath.Join(tempDir, "invalid.yaml")
	err = os.WriteFile(invalidYaml, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	_, err = Load(invalidYaml, map[string]string{})
	if err == nil {
		t.Errorf("Expected error for invalid YAML")
	}
}

func TestSubstituteEnvVars(t *testing.T) {
	config := &AppConfig{
		Secrets: map[string]string{
			"API_KEY": "${API_KEY}",
		},
		Database: DatabaseConfig{
			Password: "${DB_PASSWORD}",
		},
	}

	env := map[string]string{
		"API_KEY":     "secret123",
		"DB_PASSWORD": "dbpass",
	}

	err := substituteEnvVars(config, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config.Secrets["API_KEY"] != "secret123" {
		t.Errorf("Expected API_KEY to be substituted, got %s", config.Secrets["API_KEY"])
	}

	if config.Database.Password != "dbpass" {
		t.Errorf("Expected password to be substituted, got %s", config.Database.Password)
	}
}

func TestSubstituteEnvVarsError(t *testing.T) {
	config := &AppConfig{
		Secrets: map[string]string{
			"API_KEY": "${MISSING_VAR}",
		},
	}

	env := map[string]string{}

	err := substituteEnvVars(config, env)
	if err == nil {
		t.Errorf("Expected error for missing environment variable")
	}
}

func TestFindDomainFilesNoDomains(t *testing.T) {
	tempDir := t.TempDir()

	files, err := findDomainFiles(tempDir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected no domain files when _domains directory doesn't exist")
	}
}

func TestGetIntWithDefault(t *testing.T) {
	tests := []struct {
		value    string
		defaults int
		expected int
	}{
		{"", 10, 10},
		{"20", 10, 20},
		{"invalid", 10, 10},
		{"-5", 10, 10}, // Negative
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := getIntWithDefault(map[string]string{"NUM": tt.value}, "NUM", tt.defaults)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestLoadDomainWithError(t *testing.T) {
	// Test loading non-existent file
	_, err := LoadDomain("/non/existent/domain.yaml", map[string]string{})
	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}

	// Test loading invalid YAML
	tempDir := t.TempDir()
	invalidYaml := filepath.Join(tempDir, "invalid.yaml")
	err = os.WriteFile(invalidYaml, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	_, err = LoadDomain(invalidYaml, map[string]string{})
	if err == nil {
		t.Errorf("Expected error for invalid YAML")
	}
}

func TestSubstituteDomainEnvVars(t *testing.T) {
	config := &DomainConfig{
		Database: DatabaseConfig{
			Password: "${DOMAIN_PASSWORD}",
			DSN:      "${DOMAIN_DSN}",
		},
	}

	env := map[string]string{
		"DOMAIN_PASSWORD": "domain-secret",
		"DOMAIN_DSN":      "postgres://user:pass@host:5432/db",
	}

	err := substituteDomainEnvVars(config, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config.Database.Password != "domain-secret" {
		t.Errorf("Expected password to be substituted, got %s", config.Database.Password)
	}

	if config.Database.DSN != "postgres://user:pass@host:5432/db" {
		t.Errorf("Expected DSN to be substituted, got %s", config.Database.DSN)
	}
}

func TestFormatForCLI(t *testing.T) {
	result := &ValidationResult{
		Status: StatusError,
		Issues: []ValidationIssue{
			{Field: "name", Message: "invalid format", Hint: "use lowercase"},
			{Field: "keys", Message: "missing key", Hint: "add key"},
		},
		AppName: "test_app",
	}

	output := result.FormatForCLI()
	if output == "" {
		t.Errorf("Expected non-empty CLI output")
	}

	// Check that app name is included
	if !strings.Contains(output, "test_app:") {
		t.Errorf("Expected app name in CLI output")
	}

	// Check that issues are included
	if !strings.Contains(output, "invalid format") {
		t.Errorf("Expected issue message in CLI output")
	}
}

func TestLoadAllError(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid app.yaml
	appDir := filepath.Join(tempDir, "test_app")
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	appYaml := filepath.Join(appDir, "app.yaml")
	err = os.WriteFile(appYaml, []byte("invalid: yaml: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	_, err = LoadAll(tempDir, map[string]string{})
	if err == nil {
		t.Errorf("Expected error for invalid app.yaml")
	}
}

func TestStorageApplyDefaults(t *testing.T) {
	config := &StorageConfig{
		Endpoint: "https://s3.amazonaws.com",
		Bucket:   "test-bucket",
		Region:   "us-east-1",
		// PresignExpiry should be set by ApplyDefaults
	}

	config.ApplyDefaults()

	if config.PresignExpiry != time.Hour {
		t.Errorf("Expected presign expiry 1h, got %v", config.PresignExpiry)
	}
}

func TestScanRootError(t *testing.T) {
	// Test scanning non-existent directory
	_, _, err := ScanRoot("/non/existent/directory")
	if err == nil {
		t.Errorf("Expected error for non-existent directory")
	}
}

func TestSubstituteEnvVarsWithStorage(t *testing.T) {
	config := &AppConfig{
		Storage: &StorageConfig{
			AccessKeyID:     "${S3_ACCESS_KEY}",
			SecretAccessKey: "${S3_SECRET_KEY}",
		},
	}

	env := map[string]string{
		"S3_ACCESS_KEY": "access123",
		"S3_SECRET_KEY": "secret123",
	}

	err := substituteEnvVars(config, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config.Storage.AccessKeyID != "access123" {
		t.Errorf("Expected AccessKeyID to be substituted, got %s", config.Storage.AccessKeyID)
	}

	if config.Storage.SecretAccessKey != "secret123" {
		t.Errorf("Expected SecretAccessKey to be substituted, got %s", config.Storage.SecretAccessKey)
	}
}

func TestLoadAllWithStorage(t *testing.T) {
	tempDir := t.TempDir()

	// Create app with storage config
	appDir := filepath.Join(tempDir, "test_app")
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	appYaml := filepath.Join(appDir, "app.yaml")
	appContent := `name: test_app
keys:
  publishable_key: test-key
storage:
  endpoint: https://s3.amazonaws.com
  bucket: test-bucket
  region: us-east-1
  access_key_id: ${S3_ACCESS_KEY}
  secret_access_key: ${S3_SECRET_KEY}
`
	err = os.WriteFile(appYaml, []byte(appContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write app.yaml: %v", err)
	}

	env := map[string]string{
		"S3_ACCESS_KEY": "access123",
		"S3_SECRET_KEY": "secret123",
	}

	configSet, err := LoadAll(tempDir, env)
	if err != nil {
		t.Fatalf("Failed to load all configs: %v", err)
	}

	app, ok := configSet.GetApp("test_app")
	if !ok {
		t.Fatalf("Failed to get test_app")
	}

	if app.Storage.AccessKeyID != "access123" {
		t.Errorf("Expected AccessKeyID to be substituted, got %s", app.Storage.AccessKeyID)
	}
}

func TestJobsConfigApplyDefaults(t *testing.T) {
	config := &JobsConfig{}
	config.ApplyDefaults()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", config.MaxAttempts)
	}

	if config.Timeout != 15*time.Second {
		t.Errorf("Expected timeout 15s, got %v", config.Timeout)
	}
}

func TestFormatForCLIWithAppName(t *testing.T) {
	result := &ValidationResult{
		Status:  StatusOK,
		Issues:  []ValidationIssue{},
		AppName: "test_app",
	}

	output := result.FormatForCLI()

	// Should include app name even for OK status
	if !strings.Contains(output, "test_app:") {
		t.Errorf("Expected app name in CLI output for OK status")
	}

	// Should include status
	if !strings.Contains(output, "OK") {
		t.Errorf("Expected OK status in CLI output")
	}
}

func TestFormatForCLIWithoutAppName(t *testing.T) {
	result := &ValidationResult{
		Status: StatusWarn,
		Issues: []ValidationIssue{
			{Field: "field", Message: "warning message", Hint: "hint"},
		},
	}

	output := result.FormatForCLI()

	// Should include status
	if !strings.Contains(output, "WARN") {
		t.Errorf("Expected WARN status in CLI output")
	}

	// Should include warning symbol
	if !strings.Contains(output, "⚠") {
		t.Errorf("Expected warning symbol in CLI output")
	}
}
