package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RegisterFunctionRoutes registers function call routes
func RegisterFunctionRoutes(r chi.Router, deps *Deps) {
	// Function routes: /api/v1/{app}/functions/{functionName}
	r.Route("/functions", func(r chi.Router) {
		// Call function - POST /functions/{functionName}
		r.Post("/{functionName}", Handler(handleFunctionCall(deps)).Handle(deps))
	})
}

// Function call handler
func handleFunctionCall(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// Get function name from URL
		functionName := chi.URLParam(r, "functionName")
		if functionName == "" {
			return nil, ErrBadRequest("MISSING_FUNCTION_NAME", "Function name is required")
		}

		// Reject underscore-prefixed function names at the HTTP layer
		// Private functions are never reachable via HTTP from any client
		if strings.HasPrefix(functionName, "_") {
			return nil, ErrFunctionNotFound("Function not found")
		}

		// Check if functions client is available
		if deps.FunctionsClient == nil {
			return nil, ErrServiceUnavailable("Functions service not available")
		}

		// Forward request to functions service
		response, err := deps.FunctionsClient.Call(r.Context(), rc.AppName, functionName, r)
		if err != nil {
			return nil, ErrInternal("Failed to call function")
		}

		// Pass through the response verbatim — non-2xx responses are not wrapped
		// Return raw body for the Handler wrapper to serialize
		var result any
		if len(response.Body) > 0 {
			// Try to parse as JSON for proper envelope; fall back to raw bytes
			if err := json.Unmarshal(response.Body, &result); err != nil {
				result = string(response.Body)
			}
		}

		// For error status codes, return as a BackdError with the original body
		if response.Status >= 400 {
			return nil, &BackdError{
				Code:       "FUNCTION_ERROR",
				Detail:     string(response.Body),
				StatusCode: response.Status,
			}
		}

		return result, nil
	}
}
