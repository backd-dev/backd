package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/backd-dev/backd/internal/db"
)

// ValidateKey validates an API key and returns its type
func (a *authImpl) ValidateKey(ctx context.Context, appName, key string) (KeyType, error) {
	if key == "" {
		return "", fmt.Errorf("key cannot be empty")
	}

	// Query all keys for the app and compare
	query := `SELECT id, type, key_hash FROM _api_keys WHERE app_name = $1`
	rows, err := a.db.Query(ctx, appName, query, appName)
	if err != nil {
		slog.Error("failed to query api keys", "app", appName, "error", err)
		return "", fmt.Errorf("auth.ValidateKey: %w", err)
	}

	for _, row := range rows {
		storedHash, _ := row["key_hash"].(string)
		keyType, _ := row["type"].(string)
		if storedHash == key {
			slog.Debug("api key validated", "app", appName, "type", keyType)
			return KeyType(keyType), nil
		}
	}

	slog.Info("invalid API key", "app", appName)
	return "", fmt.Errorf("auth.ValidateKey: invalid key")
}

// UpsertPublishableKey stores or updates a publishable key
func (a *authImpl) UpsertPublishableKey(ctx context.Context, appName, key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if key already exists
	checkQuery := `SELECT id FROM _api_keys WHERE app_name = $1 AND type = 'publishable'`
	existing, err := a.db.QueryOne(ctx, appName, checkQuery, appName)
	if err != nil {
		slog.Error("failed to check existing publishable key", "app", appName, "error", err)
		return fmt.Errorf("auth.UpsertPublishableKey: %w", err)
	}

	if existing != nil {
		// Update existing key
		updateQuery := `
			UPDATE _api_keys 
			SET key_hash = $1, updated_at = NOW()
			WHERE app_name = $2 AND type = 'publishable'`

		err = a.db.Exec(ctx, appName, updateQuery, key, appName)
		if err != nil {
			slog.Error("failed to update publishable key", "app", appName, "error", err)
			return fmt.Errorf("auth.UpsertPublishableKey: %w", err)
		}

		slog.Info("publishable key updated", "app", appName)
	} else {
		// Insert new key
		insertQuery := `
			INSERT INTO _api_keys (id, app_name, key_hash, type, created_at, updated_at)
			VALUES ($1, $2, $3, 'publishable', NOW(), NOW())`

		keyID := db.NewXID()
		err = a.db.Exec(ctx, appName, insertQuery, keyID, appName, key)
		if err != nil {
			slog.Error("failed to insert publishable key", "app", appName, "error", err)
			return fmt.Errorf("auth.UpsertPublishableKey: %w", err)
		}

		slog.Info("publishable key created", "app", appName, "key_id", keyID)
	}

	return nil
}

// VerifyPublishableKey checks if a publishable key is valid
func (a *authImpl) VerifyPublishableKey(ctx context.Context, appName, key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	query := `
		SELECT id FROM _api_keys 
		WHERE app_name = $1 AND key_hash = $2 AND type = 'publishable'`

	row, err := a.db.QueryOne(ctx, appName, query, appName, key)
	if err != nil {
		slog.Error("failed to verify publishable key", "app", appName, "error", err)
		return fmt.Errorf("auth.VerifyPublishableKey: %w", err)
	}

	if row == nil {
		slog.Info("publishable key verification failed", "app", appName)
		return fmt.Errorf("auth.VerifyPublishableKey: invalid key")
	}

	slog.Debug("publishable key verified", "app", appName)
	return nil
}
