package commands

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fernandezvara/commandkit"
)

//go:embed templates/app.yaml
var appYAMLTemplate []byte

//go:embed templates/domain.yaml
var domainYAMLTemplate []byte

//go:embed templates/docker-compose.yaml
var composeTemplate []byte

//go:embed templates/.gitignore
var gitignoreTemplate []byte

//go:embed templates/.env.example
var envExampleTemplate []byte

//go:embed templates/migration.sql
var migrationTemplate []byte

//go:embed templates/function.ts
var functionTemplate []byte

//go:embed templates/.env.secrets.example
var envSecretsTemplate []byte

// nameRegex validates names according to the rules: lowercase, alphanumeric + underscores, must start with letter
var nameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// validName checks if a string matches the name regex
func validName(s string) bool {
	return nameRegex.MatchString(s)
}

// BootstrapFunc is the command function for the bootstrap command
var BootstrapFunc func(ctx *commandkit.CommandContext) error = func(ctx *commandkit.CommandContext) error {
	name, err := commandkit.Get[string](ctx, "name")
	if err != nil {
		return fmt.Errorf("failed to get name: %w", err)
	}

	domain, err := commandkit.Get[bool](ctx, "domain")
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	force, err := commandkit.Get[bool](ctx, "force")
	if err != nil {
		return fmt.Errorf("failed to get force: %w", err)
	}

	if domain {
		return bootstrapDomain(name, force)
	}
	return bootstrapApp(name, force)
}

func Bootstrap() *commandkit.CommandBuilder {
	return commandkit.New().Command("bootstrap").
		ShortHelp("Scaffold a new backd application or domain").
		LongHelp("Create a new backd application directory structure with configuration files, or create a domain configuration.").
		Config(func(cmdConfig *commandkit.CommandConfig) {
			cmdConfig.Define("domain").Bool().Default(false).Description("Create a domain instead of an app")
			cmdConfig.Define("force").Bool().Default(false).Description("Overwrite existing files")
		}).
		Func(func(ctx *commandkit.CommandContext) error {
			if len(ctx.Args) < 1 {
				return fmt.Errorf("missing required argument: app-name or domain name")
			}

			name := ctx.Args[0]
			isDomain := ctx.Flags["domain"] == "true"
			force := ctx.Flags["force"] == "true"

			if isDomain {
				return bootstrapDomain(name, force)
			}
			return bootstrapApp(name, force)
		})
}

func bootstrapApp(appName string, force bool) error {
	// Validate app name using local validation
	if !validName(appName) {
		return fmt.Errorf("invalid app name: must start with lowercase letter, contain only lowercase letters, numbers, and underscores")
	}

	// Create apps directory if it doesn't exist
	appsDir := "apps"
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return fmt.Errorf("failed to create apps directory: %w", err)
	}

	// Create app directory
	appDir := filepath.Join(appsDir, appName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	// Generate publishable key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate publishable key: %w", err)
	}
	publishableKey := base64.RawURLEncoding.EncodeToString(key)

	// Write app.yaml
	appContent := strings.ReplaceAll(string(appYAMLTemplate), "__APP_NAME__", appName)
	appContent = strings.ReplaceAll(appContent, "__PUBLISHABLE_KEY__", publishableKey)

	appPath := filepath.Join(appDir, "app.yaml")
	if err := writeFile(appPath, appContent, force); err != nil {
		return err
	}

	// Create functions directory
	funcsDir := filepath.Join(appDir, "functions")
	if err := os.MkdirAll(funcsDir, 0755); err != nil {
		return fmt.Errorf("failed to create functions directory: %w", err)
	}

	// Create migrations directory and initial migration
	migrationsDir := filepath.Join(appDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	initMigrationPath := filepath.Join(migrationsDir, "001_init.sql")
	if err := writeFile(initMigrationPath, string(migrationTemplate), force); err != nil {
		return err
	}

	// Write project root files (only on first run unless force)
	if err := writeProjectRootFiles(force); err != nil {
		return err
	}

	slog.Info("App scaffolded successfully", "app", appName, "path", appDir)
	return nil
}

func bootstrapDomain(domainName string, force bool) error {
	// Validate domain name using local validation
	if !validName(domainName) {
		return fmt.Errorf("invalid domain name: must start with lowercase letter, contain only lowercase letters, numbers, and underscores")
	}

	// Create domains directory if it doesn't exist
	domainsDir := "_domains"
	if err := os.MkdirAll(domainsDir, 0755); err != nil {
		return fmt.Errorf("failed to create domains directory: %w", err)
	}

	// Create domain directory
	domainDir := filepath.Join(domainsDir, domainName)
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return fmt.Errorf("failed to create domain directory: %w", err)
	}

	// Write domain.yaml
	domainContent := strings.ReplaceAll(string(domainYAMLTemplate), "__DOMAIN_NAME__", domainName)

	domainPath := filepath.Join(domainDir, "domain.yaml")
	if err := writeFile(domainPath, domainContent, force); err != nil {
		return err
	}

	slog.Info("Domain scaffolded successfully", "domain", domainName, "path", domainDir)
	return nil
}

func writeProjectRootFiles(force bool) error {
	// docker-compose.yaml
	composePath := "docker-compose.yaml"
	if err := writeFile(composePath, string(composeTemplate), force); err != nil {
		return err
	}

	// .gitignore
	gitignorePath := ".gitignore"
	if err := writeFile(gitignorePath, string(gitignoreTemplate), force); err != nil {
		return err
	}

	// .env.example
	envExamplePath := ".env.example"
	if err := writeFile(envExamplePath, string(envExampleTemplate), force); err != nil {
		return err
	}

	// .env.secrets.example
	envSecretsPath := ".env.secrets.example"
	if err := writeFile(envSecretsPath, string(envSecretsTemplate), force); err != nil {
		return err
	}

	return nil
}

func writeFile(path, content string, force bool) error {
	// Check if file exists
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("file %s already exists (use --force to overwrite)", path)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}
