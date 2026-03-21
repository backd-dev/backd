package commands

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/backd-dev/backd/internal/api"
	"github.com/backd-dev/backd/internal/auth"
	"github.com/backd-dev/backd/internal/celql"
	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/functions"
	"github.com/backd-dev/backd/internal/metrics"
	"github.com/backd-dev/backd/internal/secrets"
	"github.com/backd-dev/backd/internal/storage"
	"github.com/fernandezvara/commandkit"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// dbSecretsAdapter adapts secrets.Secrets to db.Secrets interface
type dbSecretsAdapter struct {
	secrets secrets.Secrets
}

func (a *dbSecretsAdapter) GenerateKey() ([]byte, error) {
	// Generate a 32-byte random key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return key, nil
}

// authDBAdapter adapts db.DB to auth.DB interface
type authDBAdapter struct {
	db db.DB
}

func (a *authDBAdapter) Exec(ctx context.Context, app, query string, args ...any) error {
	return a.db.Exec(ctx, app, query, args...)
}

func (a *authDBAdapter) Query(ctx context.Context, app, query string, args ...any) ([]map[string]any, error) {
	return a.db.Query(ctx, app, query, args...)
}

func (a *authDBAdapter) QueryOne(ctx context.Context, app, query string, args ...any) (map[string]any, error) {
	return a.db.QueryOne(ctx, app, query, args...)
}

func (a *authDBAdapter) Pool(name string) (interface{}, error) {
	return a.db.Pool(name)
}

// StartFunc is the command function for the start command
var StartFunc func(ctx *commandkit.CommandContext) error = func(ctx *commandkit.CommandContext) error {
	// Get required config-dir flag
	configDir, err := commandkit.Get[string](ctx, "config-dir")
	if err != nil {
		return fmt.Errorf("failed to get config-dir: %w", err)
	}
	if configDir == "" {
		return fmt.Errorf("config-dir is required")
	}

	// Get mode flag
	mode, _ := commandkit.Get[string](ctx, "mode")
	if mode == "" {
		mode = "both"
	}

	// Create context for operations
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

	// Phase 3: Derive per-app/domain encryption keys via HKDF (held in memory)
	masterKey := []byte(serverCfg.EncryptionKey)
	domainKeys := make(map[string][]byte)
	appKeys := make(map[string][]byte)

	// Derive domain keys
	for domainName := range configSet.Domains {
		domainKeys[domainName] = secrets.DeriveDomainKey(masterKey, domainName)
		slog.Debug("derived domain key", "domain", domainName)
	}

	// Derive app keys
	for appName := range configSet.Apps {
		appKeys[appName] = secrets.DeriveAppKey(masterKey, appName)
		slog.Debug("derived app key", "app", appName)
	}

	// Initialize core instances
	dbInstance := db.NewDB(configSet, serverCfg)
	secretsInstance := secrets.NewSecrets(dbInstance, masterKey)

	// Create adapter for db.Secrets interface
	secretsAdapter := &dbSecretsAdapter{secrets: secretsInstance}

	// Create CELQL instance for policy compilation
	celqlInstance, err := celql.New()
	if err != nil {
		slog.Error("failed to create CELQL instance", "error", err)
		os.Exit(1)
	}

	// Phase 4: Sequential domain provisioning
	slog.Info("provisioning domains")
	for domainName := range configSet.Domains {
		if err := provisionDomain(dbCtx, dbInstance, secretsInstance, secretsAdapter, domainName); err != nil {
			slog.Error("failed to provision domain", "domain", domainName, "error", err)
			os.Exit(1)
		}
		slog.Info("domain provisioned", "domain", domainName)
	}

	// Phase 5: Sequential app provisioning
	slog.Info("provisioning apps")
	appStatus := make(map[string]string)
	failedApps := 0

	for appName, appConfig := range configSet.Apps {
		status, err := provisionApp(dbCtx, dbInstance, secretsInstance, secretsAdapter, celqlInstance, appName, appConfig, configDir)
		if err != nil {
			slog.Error("app provisioning failed", "app", appName, "error", err)
			appStatus[appName] = fmt.Sprintf("FAILED: %s", err)
			failedApps++
		} else {
			appStatus[appName] = status
			slog.Info("app provisioned", "app", appName, "status", status)
		}
	}

	// Phase 6: Post-provisioning summary
	totalApps := len(configSet.Apps)
	readyApps := totalApps - failedApps

	slog.Info("provisioning summary", "total_apps", totalApps, "ready_apps", readyApps, "failed_apps", failedApps)

	if totalApps > 0 && readyApps == 0 {
		slog.Error("all apps failed to provision")
		os.Exit(1)
	}

	// Phase 7: Start services
	return startServices(serverCfg, configSet, appStatus, mode)
}

func provisionDomain(ctx context.Context, dbInstance db.DB, secretsInstance secrets.Secrets, dbSecrets db.Secrets, domainName string) error {
	// Create database if not exists
	if err := dbInstance.Provision(ctx, domainName, db.DBTypeDomain); err != nil {
		return fmt.Errorf("provision failed: %w", err)
	}

	// Bootstrap reserved tables
	if err := dbInstance.Bootstrap(ctx, domainName, db.DBTypeDomain); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Generate domain secret key if not exists
	if err := dbInstance.EnsureSecretKey(ctx, domainName, dbSecrets); err != nil {
		return fmt.Errorf("failed to ensure domain secret key: %w", err)
	}

	return nil
}

func provisionApp(ctx context.Context, dbInstance db.DB, secretsInstance secrets.Secrets, dbSecrets db.Secrets, celqlInstance celql.CELQL, appName string, appConfig *config.AppConfig, configDir string) (string, error) {
	// Create database if not exists
	if err := dbInstance.Provision(ctx, appName, db.DBTypeApp); err != nil {
		return "", fmt.Errorf("provision failed: %w", err)
	}

	// Bootstrap reserved tables
	if err := dbInstance.Bootstrap(ctx, appName, db.DBTypeApp); err != nil {
		return "", fmt.Errorf("bootstrap failed: %w", err)
	}

	// Run migrations
	migrationsPath := fmt.Sprintf("%s/%s/migrations", configDir, appName)
	if err := dbInstance.Migrate(ctx, appName, migrationsPath); err != nil {
		return "", fmt.Errorf("migration failed: %w", err)
	}

	// Verify publishable key
	if err := dbInstance.VerifyPublishableKey(ctx, appName, appConfig.Keys.PublishableKey); err != nil {
		return "", fmt.Errorf("publishable key verification failed: %w", err)
	}

	// Generate secret key if not exists in _secrets
	if err := dbInstance.EnsureSecretKey(ctx, appName, dbSecrets); err != nil {
		return "", fmt.Errorf("failed to ensure app secret key: %w", err)
	}

	// Load RLS policies: sync app.yaml → _policies table (full replace)
	if err := loadRLSPolicies(ctx, dbInstance, appName, appConfig); err != nil {
		return "", fmt.Errorf("RLS policy loading failed: %w", err)
	}

	// Pre-compile all CEL expressions via celql
	if err := compileRLSPolicies(appName, appConfig, celqlInstance); err != nil {
		return "", fmt.Errorf("policy compilation failed: %w", err)
	}

	return "ready", nil
}

func startServices(serverCfg *config.ServerConfig, configSet *config.ConfigSet, appStatus map[string]string, mode string) error {
	// Create dependencies for API
	celqlInstance, err := celql.New()
	if err != nil {
		return fmt.Errorf("failed to create CELQL instance: %w", err)
	}

	// Re-create instances with proper dependencies for API
	dbInstance := db.NewDB(configSet, serverCfg)
	secretsInstance := secrets.NewSecrets(dbInstance, []byte(serverCfg.EncryptionKey))
	authDBAdapter := &authDBAdapter{db: dbInstance}
	authInstance := auth.NewAuth(authDBAdapter, celqlInstance)
	functionsClient := functions.NewHTTPClient(serverCfg.FunctionsURL)

	// Create metrics instance
	metricsInstance := metrics.NewMetrics()

	// Create storage instances
	storageInstances := make(map[string]storage.Storage)
	for appName := range configSet.Apps {
		storageInstances[appName] = storage.NewStorage(dbInstance, configSet)
	}

	// Create API router
	router := chi.NewRouter()

	// Add middleware chain
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	// TODO: Add metrics middleware when available
	// TODO: Add auth middleware when available

	// Create dependencies struct
	deps := &api.Deps{
		DB:              dbInstance,
		Auth:            authInstance,
		Secrets:         secretsInstance,
		Storage:         nil, // Will be set per app
		Metrics:         metricsInstance,
		Config:          configSet,
		FunctionsClient: functionsClient,
	}

	// Register routes for ready apps
	for appName, status := range appStatus {
		if status == "ready" {
			// Set storage for this app if available
			if storage, exists := storageInstances[appName]; exists {
				deps.Storage = storage
			} else {
				deps.Storage = nil
			}

			// TODO: Register CRUD routes when api package is ready
			// TODO: Register auth routes if not using domain auth
			// TODO: Register storage routes if configured
			// TODO: Register functions routes
			slog.Info("routes registered for app", "app", appName)
		} else {
			// Register 503 catch-all for failed apps
			registerFailedAppRoutes(router, appName, status)
			slog.Info("503 routes registered for failed app", "app", appName, "reason", status)
		}
	}

	// Add health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// Add readiness endpoint with detailed status
	router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		topLevelStatus := "ok"
		if len(appStatus) > 0 {
			hasFailed := false
			for _, status := range appStatus {
				if status != "ready" {
					hasFailed = true
					break
				}
			}
			if hasFailed {
				if len(appStatus) == len(configSet.Apps) {
					topLevelStatus = "unavailable"
				} else {
					topLevelStatus = "degraded"
				}
			}
		}

		// Simple JSON serialization
		jsonStr := fmt.Sprintf(`{"status":"%s","apps":{`, topLevelStatus)
		for appName, status := range appStatus {
			if status == "ready" {
				jsonStr += fmt.Sprintf(`"%s":{"status":"ready"},`, appName)
			} else {
				jsonStr += fmt.Sprintf(`"%s":{"status":"failed","reason":"%s"},`, appName, status)
			}
		}
		if len(appStatus) > 0 {
			jsonStr = jsonStr[:len(jsonStr)-1] // Remove trailing comma
		}
		jsonStr += `},"domains":{"company":{"status":"ready"}}}`

		fmt.Fprint(w, jsonStr)
	})

	// Start metrics server on BACKD_METRICS_PORT
	go func() {
		metricsAddr := fmt.Sprintf(":%d", serverCfg.MetricsPort)
		slog.Info("starting metrics server", "port", serverCfg.MetricsPort)
		// TODO: Replace with actual metrics server when ready
		metricsRouter := chi.NewRouter()
		metricsRouter.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"ok"}`)
		})
		if err := http.ListenAndServe(metricsAddr, metricsRouter); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	// Start API server
	apiAddr := fmt.Sprintf(":%d", serverCfg.APIPort)
	slog.Info("starting API server", "port", serverCfg.APIPort, "mode", mode)

	server := &http.Server{
		Addr:    apiAddr,
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down API server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("API server shutdown failed", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("API server failed: %w", err)
	}

	return nil
}

// loadRLSPolicies syncs app.yaml policies to _policies table (full replace)
func loadRLSPolicies(ctx context.Context, dbInstance db.DB, appName string, appConfig *config.AppConfig) error {
	// First, clear existing policies
	if err := dbInstance.Exec(ctx, appName, "DELETE FROM _policies"); err != nil {
		return fmt.Errorf("failed to clear existing policies: %w", err)
	}

	// Insert new policies from app.yaml
	for tableName, tablePolicies := range appConfig.Policies {
		for operation, policy := range tablePolicies {
			if policy.Expression != "" {
				query := `
					INSERT INTO _policies (table_name, operation, expression, check_expr, columns, defaults, soft_delete)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
				`
				err := dbInstance.Exec(ctx, appName, query,
					tableName, operation, policy.Expression, policy.Check,
					policy.Columns, policy.Defaults, policy.Soft,
				)
				if err != nil {
					return fmt.Errorf("failed to insert policy for %s.%s: %w", tableName, operation, err)
				}
			}
		}
	}

	return nil
}

// compileRLSPolicies pre-compiles all CEL expressions via celql
func compileRLSPolicies(appName string, appConfig *config.AppConfig, celqlInstance celql.CELQL) error {
	for tableName, tablePolicies := range appConfig.Policies {
		for operation, policy := range tablePolicies {
			if policy.Expression != "" {
				// Parse the CEL expression
				ast, err := celqlInstance.Parse(policy.Expression)
				if err != nil {
					return fmt.Errorf("policy parse failed for %s.%s: %w", tableName, operation, err)
				}

				// Validate the parsed AST
				if err := celqlInstance.Validate(ast); err != nil {
					return fmt.Errorf("policy validation failed for %s.%s: %w", tableName, operation, err)
				}
			}
		}
	}

	return nil
}

// registerFailedAppRoutes registers 503 catch-all routes for failed apps
func registerFailedAppRoutes(router chi.Router, appName, reason string) {
	// Register 503 handlers for all app routes
	router.Route(fmt.Sprintf("/v1/%s", appName), func(r chi.Router) {
		r.Use(failedAppMiddleware(reason))
		r.Handle("/*", failedAppHandler(reason))
	})

	// Register 503 handlers for auth routes
	router.Route(fmt.Sprintf("/v1/_auth/%s", appName), func(r chi.Router) {
		r.Use(failedAppMiddleware(reason))
		r.Handle("/*", failedAppHandler(reason))
	})
}

// failedAppMiddleware returns 503 for all requests to failed apps
func failedAppMiddleware(reason string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"error":"SERVICE_UNAVAILABLE","error_detail":"app failed to start: %s"}`, reason)
		})
	}
}

// failedAppHandler returns 503 for all requests
func failedAppHandler(reason string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"SERVICE_UNAVAILABLE","error_detail":"app failed to start: %s"}`, reason)
	})
}
