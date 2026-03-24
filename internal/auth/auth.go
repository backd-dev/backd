package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/backd-dev/backd/internal/celql"
	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/google/cel-go/cel"
)

// KeyType represents the type of API key
type KeyType string

const (
	KeyTypePublishable KeyType = "publishable"
	KeyTypeSecret      KeyType = "secret"
)

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	AppName   string    `json:"app_name"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// User represents a user account
type User struct {
	ID        string         `json:"id"`
	Username  string         `json:"username"`
	Type      string         `json:"type"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// RequestContext represents authentication context for request processing
type RequestContext struct {
	UID           string
	Meta          map[string]any
	MetaApp       map[string]any
	Authenticated bool
	KeyType       string
}

// PolicyKey uniquely identifies a cached policy
type policyKey struct {
	appName   string
	table     string
	operation string
}

// PolicyResult represents the result of policy evaluation
type PolicyResult struct {
	SQLClause string            `json:"sql_clause"`
	Params    []any             `json:"params"`
	Defaults  map[string]string `json:"defaults"`
	Columns   []string          `json:"columns"`  // allowlist of columns
	SoftCol   string            `json:"soft_col"` // empty if hard delete
}

// PolicyCache caches pre-compiled CEL programs and policy entries
type PolicyCache struct {
	programs map[policyKey]*cel.Ast
	policies map[policyKey]*config.PolicyEntry
}

// Auth interface defines all authentication and authorization operations
type Auth interface {
	// Session lifecycle
	SignIn(ctx context.Context, appName, domainName, username, password string) (*Session, error)
	SignOut(ctx context.Context, token string) error
	ValidateSession(ctx context.Context, token string) (*RequestContext, error)

	// Key validation
	ValidateKey(ctx context.Context, appName, key string) (KeyType, error)
	UpsertPublishableKey(ctx context.Context, appName, key string) error
	VerifyPublishableKey(ctx context.Context, appName, key string) error

	// User registration & profile
	Register(ctx context.Context, appName, username, password string) (*User, error)
	UpdateUsername(ctx context.Context, appName, userID, username string) error
	UpdatePassword(ctx context.Context, appName, userID, password string) error
	GetUser(ctx context.Context, appName, userID string) (*User, error)

	// Metadata mutations — surgical JSONB updates
	SetGlobalMeta(ctx context.Context, appName, userID, key string, value any) error
	SetAppMeta(ctx context.Context, appName, userID, key string, value any) error

	// RLS
	LoadPolicies(ctx context.Context, appName string, cfg *config.AppConfig) error
	EvaluatePolicy(ctx context.Context, appName, table, operation string, rc *RequestContext) (PolicyResult, error)

	// Defaults application for CRUD operations
	ApplyDefaults(defaults map[string]string, rc *RequestContext) map[string]any
}

// authImpl implements the Auth interface
type authImpl struct {
	db     db.DB
	celql  celql.CELQL
	cache  *PolicyCache
	config *config.ConfigSet
}

// NewAuth creates a new Auth instance
func NewAuth(database db.DB, celql celql.CELQL, cfg *config.ConfigSet) Auth {
	return &authImpl{
		db:     database,
		celql:  celql,
		config: cfg,
		cache: &PolicyCache{
			programs: make(map[policyKey]*cel.Ast),
			policies: make(map[policyKey]*config.PolicyEntry),
		},
	}
}

// resolveAuthDB returns the database name to use for auth operations.
// If the app has auth.domain set, returns the domain name; otherwise returns the app name.
func (a *authImpl) resolveAuthDB(appName string) string {
	if a.config != nil {
		if appCfg, ok := a.config.GetApp(appName); ok && appCfg.Auth.Domain != "" {
			return appCfg.Auth.Domain
		}
	}
	return appName
}

// generateSecureToken generates a cryptographically random hex token.
func generateSecureToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Register creates a new user account
func (a *authImpl) Register(ctx context.Context, appName, username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password cannot be empty")
	}

	authDB := a.resolveAuthDB(appName)

	// Check if user already exists
	checkQuery := `SELECT id FROM _users WHERE username = $1`
	existing, err := a.db.QueryOne(ctx, authDB, checkQuery, username)
	if err != nil {
		slog.Error("failed to check existing user", "app", appName, "username", username, "error", err)
		return nil, fmt.Errorf("auth.Register: %w", err)
	}

	if existing != nil {
		slog.Info("user already exists", "app", appName, "username", username)
		return nil, fmt.Errorf("auth.Register: user already exists")
	}

	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "app", appName, "username", username, "error", err)
		return nil, fmt.Errorf("auth.Register: %w", err)
	}

	// Create user
	userID := db.NewXID()
	insertQuery := `
		INSERT INTO _users (id, username, password_hash, type, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, 'user', '{}', NOW(), NOW())`

	err = a.db.Exec(ctx, authDB, insertQuery, userID, username, passwordHash)
	if err != nil {
		slog.Error("failed to create user", "app", appName, "username", username, "error", err)
		return nil, fmt.Errorf("auth.Register: %w", err)
	}

	user := &User{
		ID:        userID,
		Username:  username,
		Type:      "user",
		Metadata:  make(map[string]any),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	slog.Info("user registered", "app", appName, "username", username, "user_id", userID)
	return user, nil
}

// UpdateUsername updates a user's username
func (a *authImpl) UpdateUsername(ctx context.Context, appName, userID, username string) error {
	if userID == "" || username == "" {
		return fmt.Errorf("userID and username cannot be empty")
	}

	authDB := a.resolveAuthDB(appName)

	// Check if username is already taken by another user
	checkQuery := `SELECT id FROM _users WHERE username = $1 AND id <> $2`
	existing, err := a.db.QueryOne(ctx, authDB, checkQuery, username, userID)
	if err != nil {
		slog.Error("failed to check username availability", "app", appName, "username", username, "error", err)
		return fmt.Errorf("auth.UpdateUsername: %w", err)
	}

	if existing != nil {
		slog.Info("username already taken", "app", appName, "username", username)
		return fmt.Errorf("auth.UpdateUsername: username already taken")
	}

	// Update username
	updateQuery := `
		UPDATE _users 
		SET username = $1, updated_at = NOW()
		WHERE id = $2`

	err = a.db.Exec(ctx, authDB, updateQuery, username, userID)
	if err != nil {
		slog.Error("failed to update username", "app", appName, "user_id", userID, "username", username, "error", err)
		return fmt.Errorf("auth.UpdateUsername: %w", err)
	}

	slog.Info("username updated", "app", appName, "user_id", userID, "username", username)
	return nil
}

// UpdatePassword updates a user's password
func (a *authImpl) UpdatePassword(ctx context.Context, appName, userID, password string) error {
	if userID == "" || password == "" {
		return fmt.Errorf("userID and password cannot be empty")
	}

	authDB := a.resolveAuthDB(appName)

	// Hash new password
	passwordHash, err := HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "app", appName, "user_id", userID, "error", err)
		return fmt.Errorf("auth.UpdatePassword: %w", err)
	}

	// Update password
	updateQuery := `
		UPDATE _users 
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2`

	err = a.db.Exec(ctx, authDB, updateQuery, passwordHash, userID)
	if err != nil {
		slog.Error("failed to update password", "app", appName, "user_id", userID, "error", err)
		return fmt.Errorf("auth.UpdatePassword: %w", err)
	}

	slog.Info("password updated", "app", appName, "user_id", userID)
	return nil
}

// GetUser retrieves a user by ID
func (a *authImpl) GetUser(ctx context.Context, appName, userID string) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	query := `
		SELECT id, username, type, metadata, created_at, updated_at
		FROM _users
		WHERE id = $1`

	authDB := a.resolveAuthDB(appName)

	row, err := a.db.QueryOne(ctx, authDB, query, userID)
	if err != nil {
		slog.Error("failed to get user", "app", appName, "user_id", userID, "error", err)
		return nil, fmt.Errorf("auth.GetUser: %w", err)
	}

	if row == nil {
		slog.Info("user not found", "app", appName, "user_id", userID)
		return nil, fmt.Errorf("auth.GetUser: user not found")
	}

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

	user := &User{
		ID:        row["id"].(string),
		Username:  row["username"].(string),
		Type:      row["type"].(string),
		Metadata:  metadata,
		CreatedAt: row["created_at"].(time.Time),
		UpdatedAt: row["updated_at"].(time.Time),
	}

	slog.Debug("user retrieved", "app", appName, "user_id", userID)
	return user, nil
}
