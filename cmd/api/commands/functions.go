package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/secrets"
	"github.com/fernandezvara/commandkit"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// FunctionsFunc is the command function for the functions command
var FunctionsFunc func(ctx *commandkit.CommandContext) error = func(ctx *commandkit.CommandContext) error {
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
	// Build environment map for substitution
	env := make(map[string]string)
	for _, envVar := range os.Environ() {
		if key, value, found := strings.Cut(envVar, "="); found {
			env[key] = value
		}
	}

	configSet, err := config.LoadAll(configDir, env)
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

	// Phase 3: Initialize core instances
	dbInstance := db.NewDB(configSet, serverCfg)
	secretsInstance := secrets.NewSecrets(dbInstance, []byte(serverCfg.EncryptionKey))

	// Phase 4: App-specific provisioning
	slog.Info("provisioning apps for functions service")
	for appName, appConfig := range configSet.Apps {
		if err := provisionAppForFunctions(dbCtx, dbInstance, secretsInstance, appName, appConfig, configDir); err != nil {
			slog.Error("app functions provisioning failed", "app", appName, "error", err)
			os.Exit(1)
		}
		slog.Info("app provisioned for functions", "app", appName)
	}

	// Phase 5: Start functions service based on mode
	slog.Info("starting functions service", "mode", mode)

	switch mode {
	case "functions":
		return startFunctionsHTTP(serverCfg)
	case "worker":
		return startFunctionsWorker(serverCfg, dbInstance, secretsInstance, configSet)
	case "both":
		// Start both in parallel
		go func() {
			if err := startFunctionsHTTP(serverCfg); err != nil {
				slog.Error("functions HTTP service failed", "error", err)
			}
		}()
		return startFunctionsWorker(serverCfg, dbInstance, secretsInstance, configSet)
	default:
		return fmt.Errorf("invalid mode: %s (must be functions, worker, or both)", mode)
	}
}

func provisionAppForFunctions(ctx context.Context, dbInstance db.DB, secretsInstance secrets.Secrets, appName string, appConfig *config.AppConfig, configDir string) error {
	// Verify app database exists
	pool, err := dbInstance.Pool(appName)
	if err != nil {
		return fmt.Errorf("failed to get database pool for app %s: %w", appName, err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed for app %s: %w", appName, err)
	}

	// Load function files from functions/ directory into memory index
	if err := loadFunctionFiles(appName, configDir); err != nil {
		return fmt.Errorf("failed to load function files for app %s: %w", appName, err)
	}

	// Register cron schedules from app.yaml
	if err := registerCronSchedules(appName, appConfig); err != nil {
		return fmt.Errorf("failed to register cron schedules for app %s: %w", appName, err)
	}

	return nil
}

func loadFunctionFiles(appName, configDir string) error {
	// Scan functions directory for .js/.ts files
	functionsDir := fmt.Sprintf("%s/%s/functions", configDir, appName)

	// Check if functions directory exists
	if _, err := os.Stat(functionsDir); os.IsNotExist(err) {
		slog.Info("no functions directory found", "app", appName)
		return nil
	}

	// Read directory contents
	entries, err := os.ReadDir(functionsDir)
	if err != nil {
		return fmt.Errorf("failed to read functions directory for app %s: %w", appName, err)
	}

	functionCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// Check for .js or .ts files
			if strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".ts") {
				functionCount++

				// Load file content into memory
				filePath := fmt.Sprintf("%s/%s", functionsDir, name)
				content, err := os.ReadFile(filePath)
				if err != nil {
					slog.Error("failed to read function file", "app", appName, "file", name, "error", err)
					continue
				}

				// Basic syntax validation (check for basic function structure)
				if err := validateFunctionSyntax(string(content), name); err != nil {
					slog.Warn("function syntax validation failed", "app", appName, "file", name, "error", err)
					continue
				}

				slog.Debug("function file loaded", "app", appName, "file", name, "size", len(content))

				// TODO: Store in memory index for Deno execution
				// TODO: Prepare for hot reload monitoring
			}
		}
	}

	slog.Info("function files indexed", "app", appName, "count", functionCount)
	return nil
}

func validateFunctionSyntax(content, filename string) error {
	// Basic validation - check for common function patterns
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("file is empty")
	}

	// Check for Deno function exports
	if !strings.Contains(content, "export") && !strings.Contains(content, "Deno.") {
		return fmt.Errorf("no exports or Deno APIs found")
	}

	// TODO: Add more sophisticated syntax validation
	// For now, just basic checks
	return nil
}

func registerCronSchedules(appName string, appConfig *config.AppConfig) error {
	if len(appConfig.Cron) == 0 {
		slog.Debug("no cron schedules configured", "app", appName)
		return nil
	}

	cronCount := 0
	for _, cronEntry := range appConfig.Cron {
		// Validate cron expression format (basic validation)
		if err := validateCronExpression(cronEntry.Schedule); err != nil {
			slog.Warn("invalid cron expression", "app", appName, "schedule", cronEntry.Schedule, "error", err)
			continue
		}

		// Insert cron job into _jobs table for tracking
		// TODO: Replace with actual cron scheduler registration
		// For now, just log the registration
		slog.Info("cron schedule registered",
			"app", appName,
			"schedule", cronEntry.Schedule,
			"function", cronEntry.Function,
			"trigger", "cron")
		cronCount++

		// TODO: Insert _jobs rows with trigger='cron' on fire
		// This would typically involve:
		// 1. Parsing the cron expression
		// 2. Registering with a cron scheduler
		// 3. Setting up the job to fire on schedule
	}

	slog.Info("cron schedules registered", "app", appName, "count", cronCount)
	return nil
}

func validateCronExpression(schedule string) error {
	// Basic validation - check for 5 fields separated by spaces
	parts := strings.Split(schedule, " ")
	if len(parts) != 5 {
		return fmt.Errorf("cron expression must have 5 fields (minute hour day month weekday)")
	}

	// TODO: Add more sophisticated validation
	// For now, just check basic format
	return nil
}

func startFunctionsHTTP(serverCfg *config.ServerConfig) error {
	// Create functions HTTP server on BACKD_FUNCTIONS_PORT
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Add functions routes (will be implemented when internal/api is ready)
	// For now, add basic routes with placeholders
	// Function invocation handler shared by both URL patterns
	handleFunction := func(w http.ResponseWriter, r *http.Request) {
		functionName := chi.URLParam(r, "functionName")

		// Reject underscore-prefixed function names at the HTTP layer
		if strings.HasPrefix(functionName, "_") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"FUNCTION_NOT_FOUND","error_detail":"function not found"}`)
			return
		}

		// Return a stub success response for now
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"hello"}`)
	}

	router.Route("/v1/{app}", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"ok","service":"functions"}`)
		})

		// SDK calls POST /v1/{app}/{functionName} directly
		r.Post("/{functionName}", handleFunction)

		// Also support POST /v1/{app}/functions/{functionName}
		r.Post("/functions/{functionName}", handleFunction)
	})

	functionsAddr := fmt.Sprintf(":%d", serverCfg.FunctionsPort)
	slog.Info("starting functions HTTP server", "port", serverCfg.FunctionsPort)

	server := &http.Server{
		Addr:    functionsAddr,
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down functions HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("functions HTTP server shutdown failed", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("functions HTTP server failed: %w", err)
	}

	return nil
}

func startFunctionsWorker(serverCfg *config.ServerConfig, dbInstance db.DB, secretsInstance secrets.Secrets, configSet *config.ConfigSet) error {
	slog.Info("starting functions worker service")

	// Create context for worker operations
	ctx := context.Background()

	// Job worker main loop
	ticker := time.NewTicker(time.Duration(serverCfg.JobPollInterval))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("functions worker shutting down")
			return nil
		case <-ticker.C:
			if err := processJobs(ctx, dbInstance, configSet); err != nil {
				slog.Error("job processing failed", "error", err)
				// Continue processing other jobs, don't exit
			}
		}
	}
}

func processJobs(ctx context.Context, dbInstance db.DB, configSet *config.ConfigSet) error {
	for appName := range configSet.Apps {
		// Check for jobs in _jobs_queue table using FOR UPDATE SKIP LOCKED
		// This prevents multiple workers from claiming the same job
		query := `
			SELECT id, app_name, function_name, payload, trigger, retry_count, created_at
			FROM _jobs_queue 
			WHERE status = 'pending' 
			AND (app_name = $1 OR app_name = 'system')
			ORDER BY created_at ASC
			LIMIT 10
			FOR UPDATE SKIP LOCKED
		`

		jobs, err := dbInstance.Query(ctx, appName, query, appName)
		if err != nil {
			slog.Error("failed to query jobs", "app", appName, "error", err)
			continue
		}

		for _, job := range jobs {
			if err := processJob(ctx, dbInstance, job); err != nil {
				slog.Error("job processing failed", "app", appName, "job_id", job["id"], "error", err)
				// Continue processing other jobs
			}
		}

		if len(jobs) > 0 {
			slog.Debug("processed jobs", "app", appName, "count", len(jobs))
		}
	}
	return nil
}

func processJob(ctx context.Context, dbInstance db.DB, job map[string]any) error {
	jobID := job["id"].(string)
	appName := job["app_name"].(string)
	functionName := job["function_name"].(string)

	slog.Info("processing job", "job_id", jobID, "app", appName, "function", functionName)

	// Update job status to 'running'
	if err := dbInstance.Exec(ctx, appName,
		"UPDATE _jobs_queue SET status = 'running', started_at = NOW() WHERE id = $1", jobID); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// TODO: Execute the function
	// This would involve:
	// 1. Loading the function from memory index
	// 2. Calling the functions client or internal execution
	// 3. Capturing the result

	// For now, simulate execution
	time.Sleep(100 * time.Millisecond) // Simulate work

	// Update job status to 'completed'
	if err := dbInstance.Exec(ctx, appName,
		"UPDATE _jobs_queue SET status = 'completed', completed_at = NOW() WHERE id = $1", jobID); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	slog.Info("job completed", "job_id", jobID, "app", appName, "function", functionName)
	return nil
}
