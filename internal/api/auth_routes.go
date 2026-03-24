package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Helper function to determine the target app name for auth operations
// Returns the domain name if the app has auth.domain configured, otherwise returns the app name
func getAuthTargetAppName(rc *RequestContext, deps *Deps) string {
	// Get app configuration
	appCfg, exists := deps.Config.GetApp(rc.AppName)
	if !exists {
		return rc.AppName // fallback to app name
	}

	// If app has auth.domain configured, use the domain
	if appCfg.Auth.Domain != "" {
		return appCfg.Auth.Domain
	}

	// Otherwise use the app name
	return rc.AppName
}

// RegisterAuthRoutes registers authentication routes nested under /auth
// NOTE: This function is deprecated. Use RegisterDomainAuthRoutes with resource-based routing.
func RegisterAuthRoutes(r chi.Router, deps *Deps) {
	r.Route("/auth", func(r chi.Router) {
		registerAuthHandlers(r, deps)
	})
}

// RegisterDomainAuthRoutes registers authentication routes directly (no /auth prefix).
// Used for resource-based routing at /v1/auth/{app}/... and domain auth at /v1/_auth/{domain}/...
func RegisterDomainAuthRoutes(r chi.Router, deps *Deps) {
	registerAuthHandlers(r, deps)
}

// registerAuthHandlers wires the actual auth handler functions.
func registerAuthHandlers(r chi.Router, deps *Deps) {
	r.Route("/local", func(r chi.Router) {
		r.Post("/register", Handler(handleRegister(deps)).Handle(deps))
		r.Post("/login", Handler(handleSignIn(deps)).Handle(deps))
	})
	r.Post("/refresh", Handler(handleRefresh(deps)).Handle(deps))
	r.Post("/logout", Handler(handleSignOut(deps)).Handle(deps))
	r.Patch("/profile", Handler(handleUpdateProfile(deps)).Handle(deps))
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

		// Determine target app name (domain or app based on configuration)
		targetAppName := getAuthTargetAppName(rc, deps)

		// Register user using auth package
		user, err := deps.Auth.Register(r.Context(), targetAppName, req.Username, req.Password)
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

		// Determine target app name (domain or app based on configuration)
		targetAppName := getAuthTargetAppName(rc, deps)

		// Sign in using auth package
		session, err := deps.Auth.SignIn(r.Context(), targetAppName, domain, req.Username, req.Password)
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

		// Determine target app name (domain or app based on configuration)
		targetAppName := getAuthTargetAppName(rc, deps)

		// Get user profile for the validated session
		if authCtx.UID != "" && targetAppName != "" {
			user, err := deps.Auth.GetUser(r.Context(), targetAppName, authCtx.UID)
			if err == nil && user != nil {
				return user, nil
			}
		}

		// Fallback: return auth context as user-like object
		return map[string]any{
			"id":       authCtx.UID,
			"meta":     authCtx.Meta,
			"meta_app": authCtx.MetaApp,
		}, nil
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

func handleUpdateProfile(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		if !rc.Authenticated || rc.UserID == "" {
			return nil, ErrUnauthorized("Authentication required")
		}

		var params map[string]string
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			return nil, ErrBadRequest("INVALID_JSON", "Invalid JSON body")
		}

		// Determine target app name (domain or app based on configuration)
		targetAppName := getAuthTargetAppName(rc, deps)

		if newUsername, ok := params["username"]; ok && newUsername != "" {
			if err := deps.Auth.UpdateUsername(r.Context(), targetAppName, rc.UserID, newUsername); err != nil {
				if strings.Contains(err.Error(), "already taken") {
					return nil, ErrBadRequest("USERNAME_TAKEN", "Username already taken")
				}
				return nil, ErrInternal("Failed to update username")
			}
		}

		if newPassword, ok := params["password"]; ok && newPassword != "" {
			if err := deps.Auth.UpdatePassword(r.Context(), targetAppName, rc.UserID, newPassword); err != nil {
				return nil, ErrInternal("Failed to update password")
			}
		}

		user, err := deps.Auth.GetUser(r.Context(), targetAppName, rc.UserID)
		if err != nil {
			return nil, ErrInternal("Failed to get updated user")
		}

		return user, nil
	}
}
