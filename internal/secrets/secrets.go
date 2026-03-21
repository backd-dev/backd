package secrets

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/backd-dev/backd/internal/db"
)

// ErrSecretNotFound is returned when a secret doesn't exist
var ErrSecretNotFound = fmt.Errorf("secret not found")

// Secrets interface defines all secret management operations
type Secrets interface {
	Get(ctx context.Context, appName, name string) (string, error)
	Set(ctx context.Context, appName, name, value string) error
	Delete(ctx context.Context, appName, name string) error
}

// secretsImpl implements the Secrets interface
type secretsImpl struct {
	db        db.DB
	masterKey []byte
}

// NewSecrets creates a new Secrets instance
func NewSecrets(database db.DB, masterKey []byte) Secrets {
	return &secretsImpl{
		db:        database,
		masterKey: masterKey,
	}
}

// Get retrieves a secret, decrypts it, and logs the access
func (s *secretsImpl) Get(ctx context.Context, appName, name string) (string, error) {
	// Query encrypted secret from database
	query := `SELECT encrypted_value FROM _secrets WHERE name = $1`
	rows, err := s.db.Query(ctx, appName, query, name)
	if err != nil {
		return "", fmt.Errorf("secrets.Get: query failed: %w", err)
	}
	if len(rows) == 0 {
		return "", ErrSecretNotFound
	}

	encryptedValue, ok := rows[0]["encrypted_value"].(string)
	if !ok {
		return "", fmt.Errorf("secrets.Get: invalid encrypted value type")
	}

	// Derive app-specific key
	appKey := DeriveAppKey(s.masterKey, appName)

	// Decrypt the secret
	plaintext, err := Decrypt(appKey, encryptedValue)
	if err != nil {
		return "", fmt.Errorf("secrets.Get: decryption failed: %w", err)
	}

	// Log the access
	if err := LogAccess(ctx, s.db, appName, name, "get"); err != nil {
		// Log error but don't fail the operation
		slog.Error("failed to log secret access", "app", appName, "secret", name, "error", err)
	}

	return plaintext, nil
}

// Set encrypts and stores a secret
func (s *secretsImpl) Set(ctx context.Context, appName, name, value string) error {
	// Derive app-specific key
	appKey := DeriveAppKey(s.masterKey, appName)

	// Encrypt the secret
	encryptedValue, err := Encrypt(appKey, value)
	if err != nil {
		return fmt.Errorf("secrets.Set: encryption failed: %w", err)
	}

	// Store in database (upsert)
	query := `
		INSERT INTO _secrets (name, encrypted_value) 
		VALUES ($1, $2) 
		ON CONFLICT (name) DO UPDATE SET 
			encrypted_value = EXCLUDED.encrypted_value,
			updated_at = NOW()
	`
	if err := s.db.Exec(ctx, appName, query, name, encryptedValue); err != nil {
		return fmt.Errorf("secrets.Set: database insert failed: %w", err)
	}

	// Log the access
	if err := LogAccess(ctx, s.db, appName, name, "set"); err != nil {
		// Log error but don't fail the operation
		slog.Error("failed to log secret access", "app", appName, "secret", name, "error", err)
	}

	return nil
}

// Delete removes a secret from storage
func (s *secretsImpl) Delete(ctx context.Context, appName, name string) error {
	// Check if secret exists first
	_, err := s.Get(ctx, appName, name)
	if err != nil {
		if err == ErrSecretNotFound {
			return ErrSecretNotFound
		}
		return fmt.Errorf("secrets.Delete: failed to check secret existence: %w", err)
	}

	// Delete from database
	query := `DELETE FROM _secrets WHERE name = $1`
	if err := s.db.Exec(ctx, appName, query, name); err != nil {
		return fmt.Errorf("secrets.Delete: database delete failed: %w", err)
	}

	// Log the access
	if err := LogAccess(ctx, s.db, appName, name, "delete"); err != nil {
		// Log error but don't fail the operation
		slog.Error("failed to log secret access", "app", appName, "secret", name, "error", err)
	}

	return nil
}
