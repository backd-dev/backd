package api

import (
	"encoding/json"
	"net/http"

	"github.com/backd-dev/backd/internal/db"
	"github.com/go-chi/chi/v5"
)

// RegisterInternalRoutes registers all internal API routes for Deno communication
// This router binds to 127.0.0.1:9191 exclusively and has no auth middleware
func RegisterInternalRoutes(r chi.Router, deps *Deps) {
	// Health check for internal services
	r.Get("/health", Handler(handleInternalHealth(deps)).Handle(deps))

	// Deno function execution endpoint
	r.Post("/deno/execute", Handler(handleDenoExecute(deps)).Handle(deps))

	// Job queue operations — used by SDK's Jobs.Enqueue
	r.Route("/internal", func(r chi.Router) {
		r.Post("/jobs", Handler(handleJobEnqueue(deps)).Handle(deps))
	})

	// Job lifecycle — used by workers
	r.Route("/jobs", func(r chi.Router) {
		r.Post("/claim", Handler(handleJobClaim(deps)).Handle(deps))
		r.Post("/{jobId}/complete", Handler(handleJobComplete(deps)).Handle(deps))
		r.Post("/{jobId}/fail", Handler(handleJobFail(deps)).Handle(deps))
	})
}

func handleInternalHealth(_ *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		return map[string]any{
			"status":  "ok",
			"service": "backd-internal-api",
		}, nil
	}
}

func handleDenoExecute(_ *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		return map[string]any{
			"message": "Deno execution endpoint - not yet implemented",
		}, nil
	}
}

type jobEnqueueRequest struct {
	App         string `json:"app"`
	Function    string `json:"function"`
	Input       string `json:"input"`
	Trigger     string `json:"trigger"`
	RunAt       string `json:"run_at,omitzero"`
	MaxAttempts int    `json:"max_attempts,omitzero"`
}

func handleJobEnqueue(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		var req jobEnqueueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, ErrBadRequest("INVALID_JSON", "Invalid JSON body")
		}

		if req.App == "" || req.Function == "" {
			return nil, ErrBadRequest("MISSING_FIELDS", "app and function are required")
		}

		jobID := db.NewXID()
		trigger := req.Trigger
		if trigger == "" {
			trigger = "sdk"
		}
		maxAttempts := req.MaxAttempts
		if maxAttempts == 0 {
			maxAttempts = 3
		}

		query := `INSERT INTO _jobs (id, app_name, function, payload, trigger, status, max_attempts, run_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 'pending', $6, COALESCE($7::timestamptz, NOW()), NOW(), NOW())`

		var runAt any
		if req.RunAt != "" {
			runAt = req.RunAt
		}

		err := deps.DB.Exec(r.Context(), req.App, query,
			jobID, req.App, req.Function, req.Input, trigger, maxAttempts, runAt)
		if err != nil {
			return nil, ErrInternal("Failed to enqueue job")
		}

		return map[string]string{"job_id": jobID}, nil
	}
}

func handleJobClaim(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		return map[string]any{
			"message": "Job claim endpoint - not yet implemented",
		}, nil
	}
}

func handleJobComplete(_ *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		jobId := chi.URLParam(r, "jobId")
		if jobId == "" {
			return nil, ErrBadRequest("MISSING_JOB_ID", "Job ID is required")
		}

		return map[string]any{
			"job_id": jobId,
			"status": "completed",
		}, nil
	}
}

func handleJobFail(_ *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		jobId := chi.URLParam(r, "jobId")
		if jobId == "" {
			return nil, ErrBadRequest("MISSING_JOB_ID", "Job ID is required")
		}

		return map[string]any{
			"job_id": jobId,
			"status": "failed",
		}, nil
	}
}
