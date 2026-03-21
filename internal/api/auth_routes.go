package api

import (
	"encoding/json"
	"net/http"

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
}

type SignInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
		
		// Register user using auth package
		user, err := deps.Auth.Register(r.Context(), rc.AppName, req.Username, req.Password)
		if err != nil {
			return nil, ErrBadRequest("REGISTRATION_FAILED", err.Error())
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
		
		// Sign in using auth package (assuming local domain for now)
		session, err := deps.Auth.SignIn(r.Context(), rc.AppName, "local", req.Username, req.Password)
		if err != nil {
			return nil, ErrUnauthorized("Invalid credentials")
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
