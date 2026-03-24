package api

import (
	"encoding/json"
	"net/http"
)

// writeJSON writes any value as JSON with the given status code
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeSuccess writes a single data response
// Response format: { "data": {...} }
func writeSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]any{
		"data": data,
	}

	json.NewEncoder(w).Encode(response)
}

// writeList writes a list response with pagination metadata
// Response format: { "data": [...], "count": N, "limit": N, "offset": N }
func writeList(w http.ResponseWriter, data []map[string]any, count int, limit, offset int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]any{
		"data":   data,
		"count":  count,
		"limit":  limit,
		"offset": offset,
	}

	json.NewEncoder(w).Encode(response)
}

// writeNoContent writes a 204 No Content response
func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// writeError writes an error response
// Response format: { "error": "CODE", "error_detail": "human message" }
func writeError(w http.ResponseWriter, err *BackdError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)

	response := map[string]string{
		"error":        err.Code,
		"error_detail": err.Detail,
	}

	json.NewEncoder(w).Encode(response)
}

// writeHandlerError converts any error to a BackdError and writes it
// This is used by the Handler wrapper to convert handler errors to HTTP responses
func writeHandlerError(w http.ResponseWriter, err error) {
	if backdErr, ok := err.(*BackdError); ok {
		writeError(w, backdErr)
		return
	}

	// Convert unknown errors to internal errors
	writeError(w, ErrInternal("An unexpected error occurred"))
}
