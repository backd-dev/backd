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
	// Find user by username
	query := `
		SELECT id, username, password_hash, metadata, created_at, updated_at
		FROM _users
		WHERE username = $1 AND type = 'local'`

	row, err := a.db.QueryOne(ctx, appName, query, username)
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

	// Generate session token using db.NewXID
	sessionToken := db.NewXID()
	userID := row["id"].(string)

	// Calculate expiry time
	expiresAt := time.Now().Add(24 * time.Hour) // TODO: get from config

	// Create session
	insertQuery := `
		INSERT INTO _sessions (id, user_id, app_name, token, created_at, expires_at)
		VALUES ($1, $2, $3, $4, NOW(), $5)`

	err = a.db.Exec(ctx, appName, insertQuery, sessionToken, userID, appName, sessionToken, expiresAt)
	if err != nil {
		slog.Error("failed to create session", "app", appName, "user_id", userID, "error", err)
		return nil, fmt.Errorf("auth.SignIn: %w", err)
	}

	session := &Session{
		ID:        sessionToken,
		UserID:    userID,
		AppName:   appName,
		Token:     sessionToken,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	slog.Info("session created", "app", appName, "user_id", userID, "session_id", sessionToken)
	return session, nil
}

// SignOut invalidates a user session
func (a *authImpl) SignOut(ctx context.Context, token string) error {
	query := `DELETE FROM _sessions WHERE token = $1`

	err := a.db.Exec(ctx, "", query, token) // Use empty app name since sessions are global
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

	row, err := a.db.QueryOne(ctx, "", query, token) // Use empty app name since sessions are global
	if err != nil {
		slog.Error("failed to query session", "token", token, "error", err)
		return nil, fmt.Errorf("auth.ValidateSession: %w", err)
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
