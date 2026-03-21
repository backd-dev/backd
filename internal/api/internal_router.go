package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterInternalRoutes registers all internal API routes for Deno communication
// This router binds to 127.0.0.1:9191 exclusively and has no auth middleware
func RegisterInternalRoutes(r chi.Router, deps *Deps) {
	// Health check for internal services
	r.Get("/health", Handler(handleInternalHealth(deps)).Handle(deps))
	
	// Deno function execution endpoint
	r.Post("/deno/execute", Handler(handleDenoExecute(deps)).Handle(deps))
	
	// Job queue operations
	r.Route("/jobs", func(r chi.Router) {
		// Claim next job - POST /jobs/claim
		r.Post("/claim", Handler(handleJobClaim(deps)).Handle(deps))
		
		// Complete job - POST /jobs/{jobId}/complete
		r.Post("/{jobId}/complete", Handler(handleJobComplete(deps)).Handle(deps))
		
		// Fail job - POST /jobs/{jobId}/fail
		r.Post("/{jobId}/fail", Handler(handleJobFail(deps)).Handle(deps))
	})
}

// Internal health check handler
func handleInternalHealth(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		return map[string]any{
			"status": "ok",
			"service": "backd-internal-api",
		}, nil
	}
}

// Deno execution handler - for internal function execution
func handleDenoExecute(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// This would handle Deno process execution requests
		// For now, placeholder implementation
		
		return map[string]any{
			"message": "Deno execution endpoint - not yet implemented",
		}, nil
	}
}

// Job claim handler - for workers to claim next available job
func handleJobClaim(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// This would handle job claiming from the queue
		// For now, placeholder implementation
		
		return map[string]any{
			"message": "Job claim endpoint - not yet implemented",
		}, nil
	}
}

// Job completion handler - for workers to mark jobs as completed
func handleJobComplete(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		jobId := chi.URLParam(r, "jobId")
		if jobId == "" {
			return nil, ErrBadRequest("MISSING_JOB_ID", "Job ID is required")
		}
		
		// This would handle job completion
		// For now, placeholder implementation
		
		return map[string]any{
			"job_id":  jobId,
			"status":  "completed",
			"message": "Job completion endpoint - not yet implemented",
		}, nil
	}
}

// Job failure handler - for workers to mark jobs as failed
func handleJobFail(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		jobId := chi.URLParam(r, "jobId")
		if jobId == "" {
			return nil, ErrBadRequest("MISSING_JOB_ID", "Job ID is required")
		}
		
		// This would handle job failure
		// For now, placeholder implementation
		
		return map[string]any{
			"job_id":  jobId,
			"status":  "failed",
			"message": "Job failure endpoint - not yet implemented",
		}, nil
	}
}
