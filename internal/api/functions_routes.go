package api

import (
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
		
		// Check response status
		if response.Status >= 400 {
			// For non-2xx responses, pass through the error
			if response.Status == 404 {
				return nil, ErrFunctionNotFound("Function not found")
			} else if response.Status == 500 {
				return nil, ErrInternal("Function execution failed")
			} else {
				return nil, ErrInternal("Function call failed")
			}
		}
		
		// Return the function response
		// The response body is already JSON, so we need to parse it or return it as-is
		// For now, return it as raw bytes - in a complete implementation,
		// we might want to parse and validate the JSON
		return response.Body, nil
	}
}
