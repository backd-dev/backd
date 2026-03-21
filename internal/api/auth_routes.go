package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(r chi.Router, deps *Deps) {
	// Auth routes: /api/v1/{app}/auth
	r.Route("/auth", func(r chi.Router) {
		// Local authentication endpoints
		r.Route("/local", func(r chi.Router) {
			// Register new user - POST /auth/local/register
			r.Post("/register", Handler(handleRegister(deps)).Handle(deps))

			// Sign in - POST /auth/local/login
			r.Post("/login", Handler(handleSignIn(deps)).Handle(deps))
		})

		// Refresh session - POST /auth/refresh
		r.Post("/refresh", Handler(handleRefresh(deps)).Handle(deps))

		// Sign out - POST /auth/logout
		r.Post("/logout", Handler(handleSignOut(deps)).Handle(deps))
	})
}

// Auth request/response structures
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain,omitempty"`
}

type SignInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain,omitempty"`
}

type RefreshRequest struct {
	Token string `json:"token"`
}

// Auth handlers

func handleRegister(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, ErrBadRequest("INVALID_JSON", "Invalid JSON body")
		}

		if req.Username == "" || req.Password == "" {
			return nil, ErrBadRequest("MISSING_FIELDS", "Username and password are required")
		}

		// Use provided domain or default to "local"
		domain := req.Domain
		if domain == "" {
			domain = "local"
		}

		// Register user using auth package
		user, err := deps.Auth.Register(r.Context(), rc.AppName, req.Username, req.Password)
		if err != nil {
			// Map different auth errors to appropriate API errors
			if strings.Contains(err.Error(), "user already exists") || strings.Contains(err.Error(), "duplicate") {
				return nil, ErrBadRequest("USER_EXISTS", "Username already exists")
			} else if strings.Contains(err.Error(), "invalid username") {
				return nil, ErrBadRequest("INVALID_USERNAME", "Username is invalid")
			} else if strings.Contains(err.Error(), "weak password") {
				return nil, ErrBadRequest("WEAK_PASSWORD", "Password is too weak")
			} else {
				return nil, ErrInternal("Registration failed")
			}
		}

		return user, nil
	}
}

func handleSignIn(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		var req SignInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, ErrBadRequest("INVALID_JSON", "Invalid JSON body")
		}

		if req.Username == "" || req.Password == "" {
			return nil, ErrBadRequest("MISSING_FIELDS", "Username and password are required")
		}

		// Use provided domain or default to "local"
		domain := req.Domain
		if domain == "" {
			domain = "local"
		}

		// Sign in using auth package
		session, err := deps.Auth.SignIn(r.Context(), rc.AppName, domain, req.Username, req.Password)
		if err != nil {
			// Map different auth errors to appropriate API errors
			if strings.Contains(err.Error(), "invalid credentials") || strings.Contains(err.Error(), "user not found") {
				return nil, ErrUnauthorized("Invalid credentials")
			} else if strings.Contains(err.Error(), "user disabled") {
				return nil, ErrForbidden("Account disabled")
			} else if strings.Contains(err.Error(), "too many attempts") {
				return nil, ErrTooManyRequests("Too many login attempts")
			} else {
				return nil, ErrInternal("Authentication failed")
			}
		}

		return session, nil
	}
}

func handleRefresh(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		var req RefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, ErrBadRequest("INVALID_JSON", "Invalid JSON body")
		}

		if req.Token == "" {
			return nil, ErrBadRequest("MISSING_TOKEN", "Refresh token is required")
		}

		// Validate session using auth package
		authCtx, err := deps.Auth.ValidateSession(r.Context(), req.Token)
		if err != nil {
			return nil, ErrSessionExpired("Session expired")
		}

		return authCtx, nil
	}
}

func handleSignOut(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// Get token from Authorization header or request body
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			// Try to get token from request body
			var req struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Token != "" {
				token = req.Token
			}
		}

		if token == "" {
			return nil, ErrBadRequest("MISSING_TOKEN", "Session token is required")
		}

		// Sign out using auth package
		err := deps.Auth.SignOut(r.Context(), token)
		if err != nil {
			return nil, ErrInternal("Failed to sign out")
		}

		return nil, nil // No content on successful sign out
	}
}
