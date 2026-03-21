package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/backd-dev/backd/internal/db"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// AppStatus represents the status of an individual app
type AppStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ReadyResponse represents the response from the /ready endpoint
type ReadyResponse struct {
	Status  string            `json:"status"`
	Apps    map[string]AppStatus `json:"apps"`
	Domains map[string]AppStatus `json:"domains"`
}

// HealthResponse represents the response from the /health endpoint
type HealthResponse struct {
	Status string `json:"status"`
}

// StartMetricsServer starts the Prometheus metrics server on the given port
// This server runs on a separate port from the main application router
func StartMetricsServer(ctx context.Context, port int, db db.DB) error {
	mux := http.NewServeMux()
	
	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())
	
	// Health endpoint - always returns ok
	mux.HandleFunc("/health", handleHealth)
	
	// Ready endpoint - checks app and domain health
	mux.HandleFunc("/ready", handleReady(db))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't return since this is called in a goroutine
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return nil
}

// handleHealth returns a simple health check that always succeeds
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := HealthResponse{
		Status: "ok",
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleReady checks the health of all apps and domains
func handleReady(database db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		response := ReadyResponse{
			Status:  "ok",
			Apps:    make(map[string]AppStatus),
			Domains: make(map[string]AppStatus),
		}
		
		// For now, return all apps as "ready"
		// In a real implementation, this would check database connectivity,
		// migration status, and other health indicators per app
		// TODO: Implement actual health checks when app management is available
		
		// Set status based on app availability
		// If no apps are configured, status is "ok"
		// If some apps are failing, status is "degraded"
		// If all apps are failing, status is "unavailable"
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
