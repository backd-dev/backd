package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/backd-dev/backd/internal/auth"
	"github.com/go-chi/chi/v5"
)

// RequestIDMiddleware stores the X-Request-ID header in RequestContext
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = "req-" + generateRequestID()
		}

		// Get existing RequestContext and update it
		rc := RequestContextFrom(r.Context())
		rc.RequestID = requestID

		// Store updated context
		ctx := WithRequestContext(r.Context(), rc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggingMiddleware logs request completion with structured data
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		rc := RequestContextFrom(r.Context())

		slog.Info("Request completed",
			"request_id", rc.RequestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"duration_ms", duration.Milliseconds(),
			"app", rc.AppName,
			"user_id", rc.UserID,
			"authenticated", rc.Authenticated,
		)
	})
}

// RecoveryMiddleware recovers from panics and returns 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("Panic recovered in middleware",
					"error", err,
					"path", r.URL.Path,
					"method", r.Method,
					"request_id", RequestContextFrom(r.Context()).RequestID,
				)

				writeError(w, ErrInternal("Internal server error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates sessions and API keys, populating RequestContext
func AuthMiddleware(auth auth.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := RequestContextFrom(r.Context())

			// Extract app name from URL parameter
			rc.AppName = chi.URLParam(r, "app")
			if rc.AppName == "" {
				writeError(w, ErrBadRequest("MISSING_APP", "App name required in URL"))
				return
			}

			// Try session authentication first
			sessionToken := extractBearerToken(r.Header.Get("Authorization"))
			if sessionToken != "" {
				authCtx, err := auth.ValidateSession(r.Context(), sessionToken)
				if err == nil {
					rc.UserID = authCtx.UID
					rc.Authenticated = true
					rc.Meta = authCtx.Meta
					rc.MetaApp = authCtx.MetaApp
					rc.KeyType = ""
				}
			}

			// If no valid session, try API key authentication
			if !rc.Authenticated {
				apiKey := r.Header.Get("X-API-Key")
				if apiKey != "" {
					keyType, err := auth.ValidateKey(r.Context(), rc.AppName, apiKey)
					if err == nil {
						// API keys don't provide user ID directly, they're for service access
						rc.Authenticated = true
						rc.KeyType = string(keyType)
					}
				}
			}

			// Store updated RequestContext and continue
			ctx := WithRequestContext(r.Context(), rc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Helper functions

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func extractBearerToken(authHeader string) string {
	const bearerPrefix = "Bearer "
	if strings.HasPrefix(authHeader, bearerPrefix) {
		return strings.TrimPrefix(authHeader, bearerPrefix)
	}
	return ""
}

func generateRequestID() string {
	// Simple request ID generation - in production could use UUID
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
