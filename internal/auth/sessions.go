package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/backd-dev/backd/internal/db"
)

// Error types for session management
var (
	ErrSessionExpired  = fmt.Errorf("session expired")
	ErrSessionNotFound = fmt.Errorf("session not found")
)

// SignIn creates a new user session
func (a *authImpl) SignIn(ctx context.Context, appName, domainName, username, password string) (*Session, error) {
	authDB := a.resolveAuthDB(appName)

	// Find user by username
	query := `
		SELECT id, username, password_hash, metadata, created_at, updated_at
		FROM _users
		WHERE username = $1`

	row, err := a.db.QueryOne(ctx, authDB, query, username)
	if err != nil {
		slog.Error("failed to query user", "app", appName, "username", username, "error", err)
		return nil, fmt.Errorf("auth.SignIn: %w", err)
	}

	if row == nil {
		slog.Info("user not found", "app", appName, "username", username)
		return nil, fmt.Errorf("auth.SignIn: invalid credentials")
	}

	// Verify password
	passwordHash, ok := row["password_hash"].(string)
	if !ok || !VerifyPassword(password, passwordHash) {
		slog.Info("invalid password", "app", appName, "username", username)
		return nil, fmt.Errorf("auth.SignIn: invalid credentials")
	}

	userID := row["id"].(string)

	// Generate session ID and cryptographically random token
	sessionID := db.NewXID()
	sessionToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("auth.SignIn: %w", err)
	}

	// Calculate expiry time from config
	expiry := 24 * time.Hour
	if a.config != nil {
		if appCfg, ok := a.config.GetApp(appName); ok && appCfg.Auth.SessionExpiry > 0 {
			expiry = appCfg.Auth.SessionExpiry
		}
	}
	expiresAt := time.Now().Add(expiry)

	// Create session
	insertQuery := `
		INSERT INTO _sessions (id, user_id, app_name, token, created_at, expires_at)
		VALUES ($1, $2, $3, $4, NOW(), $5)`

	err = a.db.Exec(ctx, authDB, insertQuery, sessionID, userID, appName, sessionToken, expiresAt)
	if err != nil {
		slog.Error("failed to create session", "app", appName, "user_id", userID, "error", err)
		return nil, fmt.Errorf("auth.SignIn: %w", err)
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		AppName:   appName,
		Token:     sessionToken,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	slog.Info("session created", "app", appName, "user_id", userID, "session_id", sessionID)
	return session, nil
}

// SignOut invalidates a user session
func (a *authImpl) SignOut(ctx context.Context, token string) error {
	// We need to find which DB the session lives in. Try all domains first, then apps.
	query := `DELETE FROM _sessions WHERE token = $1`

	// Try each domain DB
	if a.config != nil {
		for domainName := range a.config.Domains {
			err := a.db.Exec(ctx, domainName, query, token)
			if err == nil {
				slog.Info("session deleted", "domain", domainName)
				return nil
			}
		}
		// Try each app DB
		for appName := range a.config.Apps {
			if a.config.Apps[appName].Auth.Domain != "" {
				continue // Skip apps that use domain auth
			}
			err := a.db.Exec(ctx, appName, query, token)
			if err == nil {
				slog.Info("session deleted", "app", appName)
				return nil
			}
		}
	}

	// Fallback: try with empty name (will likely fail but preserves old behavior)
	err := a.db.Exec(ctx, "", query, token)
	if err != nil {
		slog.Error("failed to delete session", "token", token, "error", err)
		return fmt.Errorf("auth.SignOut: %w", err)
	}

	slog.Info("session deleted", "token", token)
	return nil
}

// ValidateSession validates a session token and returns request context
func (a *authImpl) ValidateSession(ctx context.Context, token string) (*RequestContext, error) {
	query := `
		SELECT s.id, s.user_id, s.app_name, s.expires_at, u.username, u.metadata
		FROM _sessions s
		JOIN _users u ON s.user_id = u.id
		WHERE s.token = $1`

	// Search all domain and app DBs for the session
	var row map[string]any
	var err error

	if a.config != nil {
		// Try domain DBs first
		for domainName := range a.config.Domains {
			row, err = a.db.QueryOne(ctx, domainName, query, token)
			if err == nil && row != nil {
				break
			}
		}
		// Try app DBs if not found in domains
		if row == nil {
			for appName := range a.config.Apps {
				if a.config.Apps[appName].Auth.Domain != "" {
					continue // Skip apps that use domain auth
				}
				row, err = a.db.QueryOne(ctx, appName, query, token)
				if err == nil && row != nil {
					break
				}
			}
		}
	}

	if row == nil {
		slog.Info("session not found", "token", token)
		return nil, ErrSessionNotFound
	}

	// Check expiry
	expiresAt, ok := row["expires_at"].(time.Time)
	if !ok || time.Now().After(expiresAt) {
		slog.Info("session expired", "token", token, "expires_at", expiresAt)
		// Clean up expired session
		_ = a.SignOut(ctx, token)
		return nil, ErrSessionExpired
	}

	userID := row["user_id"].(string)
	appName := row["app_name"].(string)

	// Parse metadata
	var metadata map[string]any
	if metadataStr, ok := row["metadata"].(string); ok && metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			slog.Warn("failed to parse user metadata", "user_id", userID, "error", err)
			metadata = make(map[string]any)
		}
	} else {
		metadata = make(map[string]any)
	}

	// Extract global and app-specific metadata
	meta := make(map[string]any)
	metaApp := make(map[string]any)

	// Global metadata is stored under "_" key
	if globalMeta, ok := metadata["_"].(map[string]any); ok {
		meta = globalMeta
	}

	// App-specific metadata
	if appMeta, ok := metadata[appName].(map[string]any); ok {
		metaApp = appMeta
	}

	rc := &RequestContext{
		UID:           userID,
		Meta:          meta,
		MetaApp:       metaApp,
		Authenticated: true,
		KeyType:       "",
	}

	slog.Debug("session validated", "app", appName, "user_id", userID)
	return rc, nil
}
