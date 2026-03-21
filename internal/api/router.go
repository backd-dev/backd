package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new chi.Router with the full middleware chain
func NewRouter(deps *Deps) chi.Router {
	r := chi.NewRouter()

	// Apply middleware in the correct order
	r.Use(
		middleware.RequestID, // chi built-in: X-Request-ID header
		RequestIDMiddleware,  // store in RequestContext
		LoggingMiddleware,    // slog on completion
		RecoveryMiddleware,   // panic → 500 + stack trace
		// metrics.RequestMiddleware, // Prometheus counters (will be added in milestone 11)
		AuthMiddleware(deps.Auth), // session/key → RequestContext
	)

	return r
}

// NewInternalRouter creates a router for internal API communication
// Binds to 127.0.0.1:9191 exclusively, no auth middleware
func NewInternalRouter(deps *Deps) chi.Router {
	r := chi.NewRouter()

	// Internal router has minimal middleware - no auth
	r.Use(
		middleware.RequestID, // chi built-in: X-Request-ID header
		RequestIDMiddleware,  // store in RequestContext
		LoggingMiddleware,    // slog on completion
		RecoveryMiddleware,   // panic → 500 + stack trace
	)

	return r
}

// RegisterRoutes registers all public API routes
func RegisterRoutes(r chi.Router, deps *Deps) {
	// Register CRUD routes for all collections
	RegisterCRUDRoutes(r, deps)

	// Register authentication routes
	RegisterAuthRoutes(r, deps)

	// Register storage routes
	RegisterStorageRoutes(r, deps)

	// Register function call routes
	RegisterFunctionRoutes(r, deps)
}
