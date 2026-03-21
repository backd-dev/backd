package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/backd-dev/backd/internal/celql"
	"github.com/backd-dev/backd/internal/config"
	"github.com/fernandezvara/commandkit"
)

// ValidateFunc is the command function for the validate command
var ValidateFunc func(ctx *commandkit.CommandContext) error = func(ctx *commandkit.CommandContext) error {
	configDir, err := commandkit.Get[string](ctx, "config-dir")
	if err != nil {
		return fmt.Errorf("failed to get config-dir: %w", err)
	}
	if configDir == "" {
		configDir = "."
	}

	asJSON, err := commandkit.Get[bool](ctx, "json")
	if err != nil {
		return fmt.Errorf("failed to get json: %w", err)
	}

	checkEnv, err := commandkit.Get[bool](ctx, "check-env")
	if err != nil {
		return fmt.Errorf("failed to get check-env: %w", err)
	}

	return validateConfigs(configDir, asJSON, checkEnv)
}

func Validate() *commandkit.CommandBuilder {
	return commandkit.New().Command("validate").
		ShortHelp("Validate configuration files").
		LongHelp("Static analysis of configuration files without runtime requirements. Validates all apps and domains in the config root.").
		Config(func(cmdConfig *commandkit.CommandConfig) {
			cmdConfig.Define("config-dir").String().Default(".").Description("Configuration root directory")
			cmdConfig.Define("json").Bool().Default(false).Description("Output results as JSON")
			cmdConfig.Define("check-env").Bool().Default(false).Description("Check if environment variables exist")
		}).
		Func(func(ctx *commandkit.CommandContext) error {
			configDir := ctx.Flags["config-dir"]
			if configDir == "" {
				configDir = "."
			}
			asJSON := ctx.Flags["json"] == "true"
			checkEnv := ctx.Flags["check-env"] == "true"

			return validateConfigs(configDir, asJSON, checkEnv)
		})
}

type ValidationResult struct {
	Valid    bool                     `json:"valid"`
	Apps     []AppValidationResult    `json:"apps"`
	Domains  []DomainValidationResult `json:"domains"`
	Errors   []string                 `json:"errors,omitempty"`
	Warnings []string                 `json:"warnings,omitempty"`
}

type AppValidationResult struct {
	Name     string   `json:"name"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type DomainValidationResult struct {
	Name     string   `json:"name"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func validateConfigs(configRoot string, asJSON, checkEnv bool) error {
	// Check if config root exists
	if _, err := os.Stat(configRoot); os.IsNotExist(err) {
		if asJSON {
			result := ValidationResult{
				Valid:  false,
				Errors: []string{fmt.Sprintf("config root not found: %s", configRoot)},
			}
			return outputJSON(result)
		}
		fmt.Printf("✗ Config root not found: %s\n", configRoot)
		return fmt.Errorf("config root not found: %s", configRoot)
	}

	// Scan for configurations
	appFiles, domainFiles, err := config.ScanRoot(configRoot)
	if err != nil {
		return fmt.Errorf("failed to scan config root: %w", err)
	}

	// Convert to our format
	result := ValidationResult{
		Valid:    true,
		Apps:     make([]AppValidationResult, 0),
		Domains:  make([]DomainValidationResult, 0),
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Process domain results first
	for _, domainFile := range domainFiles {
		domainName := filepath.Base(filepath.Dir(domainFile))
		domain, err := config.LoadDomain(domainFile, nil)
		if err != nil {
			result.Domains = append(result.Domains, DomainValidationResult{
				Name:     domainName,
				Valid:    false,
				Errors:   []string{fmt.Sprintf("failed to load: %v", err)},
				Warnings: []string{},
			})
			result.Valid = false
			continue
		}

		domainResult := config.ValidateDomain(domain)
		domainValidation := DomainValidationResult{
			Name:     domainName,
			Valid:    domainResult.Status == config.StatusOK,
			Errors:   make([]string, 0),
			Warnings: make([]string, 0),
		}

		for _, issue := range domainResult.Issues {
			if issue.IsError {
				domainValidation.Errors = append(domainValidation.Errors, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
				result.Valid = false
			} else {
				domainValidation.Warnings = append(domainValidation.Warnings, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
			}
		}

		result.Domains = append(result.Domains, domainValidation)
	}

	// Process app results
	for _, appFile := range appFiles {
		appName := filepath.Base(filepath.Dir(appFile))
		app, err := config.Load(appFile, nil)
		if err != nil {
			result.Apps = append(result.Apps, AppValidationResult{
				Name:     appName,
				Valid:    false,
				Errors:   []string{fmt.Sprintf("failed to load: %v", err)},
				Warnings: []string{},
			})
			result.Valid = false
			continue
		}

		migrationsPath := filepath.Join(configRoot, appName, "migrations")
		functionsPath := filepath.Join(configRoot, appName, "functions")
		appResult := config.ValidateApp(app, migrationsPath, functionsPath, []string{}) // TODO: Pass domain names

		appValidation := AppValidationResult{
			Name:     appName,
			Valid:    appResult.Status == config.StatusOK,
			Errors:   make([]string, 0),
			Warnings: make([]string, 0),
		}

		for _, issue := range appResult.Issues {
			if issue.IsError {
				appValidation.Errors = append(appValidation.Errors, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
				result.Valid = false
			} else {
				appValidation.Warnings = append(appValidation.Warnings, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
			}
		}

		// Validate CEL expressions in policies
		for tableName, policies := range app.Policies {
			for policyName, policy := range policies {
				if policy.Expression != "" {
					celql, err := celql.New()
					if err != nil {
						appValidation.Errors = append(appValidation.Errors, fmt.Sprintf("policies.%s.%s: failed to create CELQL validator: %s", tableName, policyName, err))
						result.Valid = false
						continue
					}
					ast, err := celql.Parse(policy.Expression)
					if err != nil {
						appValidation.Errors = append(appValidation.Errors, fmt.Sprintf("policies.%s.%s: %s", tableName, policyName, err))
						result.Valid = false
						continue
					}
					if err := celql.Validate(ast); err != nil {
						appValidation.Errors = append(appValidation.Errors, fmt.Sprintf("policies.%s.%s: %s", tableName, policyName, err))
						result.Valid = false
					}
				}
			}
		}

		result.Apps = append(result.Apps, appValidation)
	}

	// Check environment variables if requested
	if checkEnv {
		envErrors := checkEnvironmentVariables(appFiles)
		result.Errors = append(result.Errors, envErrors...)
		if len(envErrors) > 0 {
			result.Valid = false
		}
	}

	// Output results
	if asJSON {
		return outputJSON(result)
	}

	return outputText(result)
}

func checkEnvironmentVariables(appFiles []string) []string {
	var errors []string

	// Check app environment variables
	for _, appFile := range appFiles {
		appName := filepath.Base(filepath.Dir(appFile))
		app, err := config.Load(appFile, nil)
		if err != nil {
			continue // Skip if we can't load
		}

		for _, secretValue := range app.Secrets {
			if strings.HasPrefix(secretValue, "${") && strings.HasSuffix(secretValue, "}") {
				envVar := strings.TrimSuffix(strings.TrimPrefix(secretValue, "${"), "}")
				if os.Getenv(envVar) == "" {
					errors = append(errors, fmt.Sprintf("app %s: environment variable %s not set", appName, envVar))
				}
			}
		}
	}

	return errors
}

func outputText(result ValidationResult) error {
	fmt.Printf("Validation Results\n")
	fmt.Printf("=================\n\n")

	// Domains
	if len(result.Domains) > 0 {
		fmt.Printf("Domains (%d):\n", len(result.Domains))
		for _, domain := range result.Domains {
			if domain.Valid {
				fmt.Printf("  ✓ %s\n", domain.Name)
			} else {
				fmt.Printf("  ✗ %s\n", domain.Name)
				for _, err := range domain.Errors {
					fmt.Printf("    ✗ %s\n", err)
				}
			}
			for _, warn := range domain.Warnings {
				fmt.Printf("    ⚠ %s\n", warn)
			}
		}
		fmt.Println()
	}

	// Apps
	if len(result.Apps) > 0 {
		fmt.Printf("Apps (%d):\n", len(result.Apps))
		for _, app := range result.Apps {
			if app.Valid {
				fmt.Printf("  ✓ %s\n", app.Name)
			} else {
				fmt.Printf("  ✗ %s\n", app.Name)
				for _, err := range app.Errors {
					fmt.Printf("    ✗ %s\n", err)
				}
			}
			for _, warn := range app.Warnings {
				fmt.Printf("    ⚠ %s\n", warn)
			}
		}
		fmt.Println()
	}

	// Global errors and warnings
	for _, err := range result.Errors {
		fmt.Printf("✗ %s\n", err)
	}
	for _, warn := range result.Warnings {
		fmt.Printf("⚠ %s\n", warn)
	}

	// Summary
	if result.Valid {
		fmt.Printf("✓ All configurations are valid\n")
		return nil
	} else {
		fmt.Printf("✗ Configuration validation failed\n")
		return fmt.Errorf("validation failed")
	}
}

func outputJSON(result ValidationResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

	if !result.Valid {
		return fmt.Errorf("validation failed")
	}
	return nil
}
