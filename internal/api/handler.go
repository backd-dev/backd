package api

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5"
)

// Handler defines the signature for all API handlers
// Every handler must return (any, error) - never take http.ResponseWriter as a parameter
type Handler func(r *http.Request, rc *RequestContext) (any, error)

// Handle wraps a Handler with HTTP response writing and panic recovery
// This is the only place where Handler functions are converted to http.Handler
func (h Handler) Handle(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Recover from panics and convert to 500 error
		defer func() {
			if err := recover(); err != nil {
				slog.Error("Handler panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
					"path", r.URL.Path,
					"method", r.Method,
				)

				writeHandlerError(w, ErrInternal("Internal server error"))
			}
		}()

		// Get request context
		rc := RequestContextFrom(r.Context())

		// Call handler
		result, err := h(r, rc)
		if err != nil {
			writeHandlerError(w, err)
			return
		}

		// Write success response
		if result != nil {
			writeSuccess(w, result)
		} else {
			writeNoContent(w)
		}
	}
}

// Router wraps chi.Router and provides handler registration
type Router struct {
	r    chi.Router
	deps *Deps
}

// NewAPIRouter creates a new API router with the given dependencies
func NewAPIRouter(deps *Deps) *Router {
	return &Router{
		r:    chi.NewRouter(),
		deps: deps,
	}
}

// Handle registers a handler for the given pattern
func (router *Router) Handle(pattern string, handler Handler) {
	router.r.Handle(pattern, http.HandlerFunc(handler.Handle(router.deps)))
}

// ServeHTTP implements http.Handler interface
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.r.ServeHTTP(w, r)
}
