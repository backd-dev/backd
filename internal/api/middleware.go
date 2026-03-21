package api

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
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

// RecoveryMiddleware recovers from panics, logs stack trace, and returns 500
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("Panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
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
func AuthMiddleware(authService auth.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := RequestContextFrom(r.Context())

			// Extract app name from URL parameter
			rc.AppName = chi.URLParam(r, "app")
			if rc.AppName == "" {
				writeError(w, ErrBadRequest("MISSING_APP", "App name required in URL"))
				return
			}

			// Try session authentication first (X-Session header or backd_session cookie)
			sessionToken := r.Header.Get("X-Session")
			if sessionToken == "" {
				// Fall back to cookie
				cookie, err := r.Cookie("backd_session")
				if err == nil {
					sessionToken = cookie.Value
				}
			}

			if sessionToken != "" {
				authCtx, err := authService.ValidateSession(r.Context(), sessionToken)
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
				// Check for publishable key
				publishableKey := r.Header.Get("X-Publishable-Key")
				if publishableKey != "" {
					keyType, err := authService.ValidateKey(r.Context(), rc.AppName, publishableKey)
					if err == nil && keyType == auth.KeyTypePublishable {
						rc.Authenticated = true
						rc.KeyType = "publishable"
					}
				} else {
					// Check for secret key (internal Deno calls only)
					secretKey := r.Header.Get("X-Secret-Key")
					if secretKey != "" {
						keyType, err := authService.ValidateKey(r.Context(), rc.AppName, secretKey)
						if err == nil && keyType == auth.KeyTypeSecret {
							rc.Authenticated = true
							rc.KeyType = "secret"
						}
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
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
