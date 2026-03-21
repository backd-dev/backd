package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RequestMiddleware creates a middleware that records HTTP request metrics
// This middleware should be used in the public API router
func RequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Process the request
		next.ServeHTTP(ww, r)

		// Extract app name from chi URL parameters
		var app string
		if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
			app = routeCtx.URLParam("app")
		}
		if app == "" {
			app = "unknown"
		}

		// Record metrics
		RequestTotal.WithLabelValues(app, r.Method, strconv.Itoa(ww.Status())).Inc()
		RequestDuration.WithLabelValues(app, r.Method).Observe(time.Since(start).Seconds())
	})
}

// InternalRequestMiddleware creates a middleware for internal router requests
// This is similar to RequestMiddleware but optimized for internal endpoints
func InternalRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Process the request
		next.ServeHTTP(ww, r)

		// For internal requests, we use "internal" as the app name
		app := "internal"

		// Record metrics
		RequestTotal.WithLabelValues(app, r.Method, strconv.Itoa(ww.Status())).Inc()
		RequestDuration.WithLabelValues(app, r.Method).Observe(time.Since(start).Seconds())
	})
}
