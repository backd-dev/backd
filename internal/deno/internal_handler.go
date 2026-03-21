package deno

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/backd-dev/backd/internal/auth"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/secrets"
	"github.com/go-chi/chi/v5"
)

// InternalHandler handles internal HTTP requests from Deno workers
type InternalHandler struct {
	deno    Deno
	db      db.DB
	auth    auth.Auth
	secrets secrets.Secrets
}

// NewInternalHandler creates a new internal handler
func NewInternalHandler(deno Deno, db db.DB, auth auth.Auth, secrets secrets.Secrets) *InternalHandler {
	return &InternalHandler{
		deno:    deno,
		db:      db,
		auth:    auth,
		secrets: secrets,
	}
}

// startInternalServer starts the internal HTTP server on 127.0.0.1:9191
func (d *denoImpl) startInternalServer() {
	handler := NewInternalHandler(d, d.db, d.auth, d.secrets)

	router := chi.NewRouter()
	handler.Routes(router)

	server := &http.Server{
		Addr:         "127.0.0.1:9191",
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash - for now just ignore
		}
	}()
}

// Routes registers all internal routes
func (h *InternalHandler) Routes(r chi.Router) {
	r.Post("/internal/query", h.handleQuery)
	r.Post("/internal/secret", h.handleSecret)
	r.Post("/internal/auth", h.handleAuth)
	r.Post("/internal/jobs", h.handleEnqueueJob)
}

// handleQuery handles database queries
func (h *InternalHandler) handleQuery(w http.ResponseWriter, r *http.Request) {
	var request struct {
		App    string `json:"app"`
		Query  string `json:"query"`
		Params []any  `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Execute query
	results, err := h.db.Query(r.Context(), request.App, request.Query, request.Params...)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleSecret handles secret retrieval
func (h *InternalHandler) handleSecret(w http.ResponseWriter, r *http.Request) {
	var request struct {
		App  string `json:"app"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Get secret
	secret, err := h.secrets.Get(r.Context(), request.App, request.Name)
	if err != nil {
		http.Error(w, "secret not found", http.StatusNotFound)
		return
	}

	// Log access - placeholder since LogAccess might not be implemented yet
	// h.secrets.LogAccess(r.Context(), request.App, request.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"secret": secret})
}

// handleAuth handles authentication validation
func (h *InternalHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	var request struct {
		App    string `json:"app"`
		Token  string `json:"token"`
		APIKey string `json:"api_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var result any
	var err error

	if request.Token != "" {
		// Validate session
		result, err = h.auth.ValidateSession(r.Context(), request.Token)
	} else if request.APIKey != "" {
		// Validate API key
		result, err = h.auth.ValidateKey(r.Context(), request.App, request.APIKey)
	} else {
		http.Error(w, "token or api_key required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleEnqueueJob handles job enqueueing
func (h *InternalHandler) handleEnqueueJob(w http.ResponseWriter, r *http.Request) {
	var request struct {
		App      string `json:"app"`
		Function string `json:"function"`
		Input    string `json:"input"`
		Trigger  string `json:"trigger"`
		RunAt    string `json:"run_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Parse run_at if provided
	var runAt time.Time
	if request.RunAt != "" {
		parsed, err := time.Parse(time.RFC3339, request.RunAt)
		if err != nil {
			http.Error(w, "invalid run_at format", http.StatusBadRequest)
			return
		}
		runAt = parsed
	} else {
		runAt = time.Now()
	}

	// Insert job
	jobID := db.NewXID()
	err := h.db.Exec(r.Context(), request.App,
		`INSERT INTO _jobs (id, function, payload, trigger, status, run_at, created_at)
		 VALUES ($1, $2, $3, $4, 'pending', $5, NOW())`,
		jobID, request.Function, request.Input, request.Trigger, runAt)

	if err != nil {
		http.Error(w, "failed to enqueue job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}
