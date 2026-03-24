package db

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationFile represents a migration file
type MigrationFile struct {
	Name string
	SQL  string
}

// Migrate applies pending migrations for an app
func (db *dbImpl) Migrate(ctx context.Context, appName, migrationsPath string) error {
	// Get connection pool for the app
	pool, err := db.Pool(appName)
	if err != nil {
		return fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	// Read migration files from directory
	files, err := readMigrationFiles(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// Load already applied migrations
	applied, err := loadAppliedMigrations(ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to load applied migrations: %w", err)
	}

	// Find pending migrations
	pending := findPendingMigrations(files, applied)

	// Apply pending migrations
	for _, file := range pending {
		if err := applyMigration(ctx, pool, file); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file.Name, err)
		}
	}

	slog.Info("migration completed", "app", appName, "applied_count", len(pending))
	return nil
}

// readMigrationFiles reads all SQL migration files from a directory
func readMigrationFiles(migrationsPath string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []MigrationFile{}, nil // No migrations directory is fine
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []MigrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Check if filename has numeric prefix
		if !isValidMigrationFilename(name) {
			continue
		}

		path := filepath.Join(migrationsPath, name)
		sql, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		files = append(files, MigrationFile{
			Name: name,
			SQL:  string(sql),
		})
	}

	// Sort by numeric prefix
	slices.SortFunc(files, func(a, b MigrationFile) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	return files, nil
}

// isValidMigrationFilename checks if filename follows NNN_description.sql format
func isValidMigrationFilename(name string) bool {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 {
		return false
	}

	// Check if first part is numeric
	numStr := parts[0]
	if len(numStr) != 3 {
		return false
	}

	for _, ch := range numStr {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return strings.HasSuffix(name, ".sql")
}

// loadAppliedMigrations loads the list of already applied migrations from the database
func loadAppliedMigrations(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, "SELECT filename FROM _migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, fmt.Errorf("failed to scan migration filename: %w", err)
		}
		applied[filename] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return applied, nil
}

// findPendingMigrations returns migrations that haven't been applied yet
func findPendingMigrations(files []MigrationFile, applied map[string]bool) []MigrationFile {
	var pending []MigrationFile
	for _, file := range files {
		if !applied[file.Name] {
			pending = append(pending, file)
		}
	}
	return pending
}

// applyMigration applies a single migration in a transaction
func applyMigration(ctx context.Context, pool *pgxpool.Pool, file MigrationFile) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute the migration SQL
	if _, err := tx.Exec(ctx, file.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record the migration as applied
	if _, err := tx.Exec(ctx, "INSERT INTO _migrations (filename) VALUES ($1)", file.Name); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	slog.Info("applied migration", "file", file.Name)
	return nil
}

// UpsertPublishableKey stores or updates the publishable key for an app
func (db *dbImpl) UpsertPublishableKey(ctx context.Context, appName, key string) error {
	pool, err := db.Pool(appName)
	if err != nil {
		return fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	// Check if key already exists
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM _api_keys WHERE app_name = $1 AND type = 'publishable'", appName).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query publishable key: %w", err)
	}

	if count == 0 {
		// Insert new key
		keyID := NewXID()
		_, err = pool.Exec(ctx,
			"INSERT INTO _api_keys (id, app_name, key_hash, type, created_at, updated_at) VALUES ($1, $2, $3, 'publishable', NOW(), NOW())",
			keyID, appName, key)
		if err != nil {
			return fmt.Errorf("failed to insert publishable key: %w", err)
		}
		slog.Info("publishable key stored", "app", appName)
	}

	return nil
}

// VerifyPublishableKey checks if the publishable key in the database matches the provided key
func (db *dbImpl) VerifyPublishableKey(ctx context.Context, appName, key string) error {
	pool, err := db.Pool(appName)
	if err != nil {
		return fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM _api_keys WHERE app_name = $1 AND type = 'publishable'", appName).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query publishable key: %w", err)
	}

	// If no key exists yet, this is the first run — not an error
	if count == 0 {
		return nil
	}

	// Key exists — verify it matches by checking key_hash
	var storedHash string
	err = pool.QueryRow(ctx, "SELECT key_hash FROM _api_keys WHERE app_name = $1 AND type = 'publishable'", appName).Scan(&storedHash)
	if err != nil {
		return fmt.Errorf("failed to query publishable key hash: %w", err)
	}

	// Simple comparison — in production this would use a constant-time compare on hashed keys
	if storedHash != key {
		return fmt.Errorf("publishable key mismatch for app %s", appName)
	}

	return nil
}

// EnsureSecretKey ensures a secret key exists for the app
func (db *dbImpl) EnsureSecretKey(ctx context.Context, appName string, secrets Secrets) error {
	pool, err := db.Pool(appName)
	if err != nil {
		return fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	// Check if secret key already exists
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM _secrets WHERE name = 'secret_key'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing secret key: %w", err)
	}

	if count > 0 {
		return nil // Secret key already exists
	}

	// Generate new secret key
	key, err := secrets.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate secret key: %w", err)
	}

	// Store the secret key encoded as base64 to avoid UTF8 encoding issues
	// TODO: In a real implementation, this would use the secrets package to encrypt the key
	keyBase64 := base64.StdEncoding.EncodeToString(key)
	_, err = pool.Exec(ctx, "INSERT INTO _secrets (name, value_encrypted) VALUES ($1, $2)", "secret_key", keyBase64)
	if err != nil {
		return fmt.Errorf("failed to store secret key: %w", err)
	}

	slog.Info("generated and stored secret key", "app", appName)
	return nil
}
