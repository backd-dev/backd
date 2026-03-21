package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Reserved table DDL constants
// This is the only file that should contain CREATE TABLE statements for reserved tables

const (
	// App tables
	createMigrationsTable = `CREATE TABLE IF NOT EXISTS _migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createUsersTable = `CREATE TABLE IF NOT EXISTS _users (
		_id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		email TEXT,
		type TEXT DEFAULT 'user',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createSessionsTable = `CREATE TABLE IF NOT EXISTS _sessions (
		_id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES _users(_id) ON DELETE CASCADE,
		token_hash TEXT UNIQUE NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		meta JSONB DEFAULT '{}',
		meta_app JSONB DEFAULT '{}'
	)`

	createSecretsTable = `CREATE TABLE IF NOT EXISTS _secrets (
		name TEXT PRIMARY KEY,
		value_encrypted TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createApiKeysTable = `CREATE TABLE IF NOT EXISTS _api_keys (
		_id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		publishable_key TEXT UNIQUE NOT NULL,
		secret_key_encrypted TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createPoliciesTable = `CREATE TABLE IF NOT EXISTS _policies (
		table_name TEXT NOT NULL,
		policy_name TEXT NOT NULL,
		expression TEXT NOT NULL,
		check_expression TEXT,
		allowed_columns TEXT[],
		defaults JSONB DEFAULT '{}',
		soft_delete_column TEXT,
		PRIMARY KEY (table_name, policy_name)
	)`

	createFilesTable = `CREATE TABLE IF NOT EXISTS _files (
		_id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		content_type TEXT,
		size_bytes BIGINT,
		storage_key TEXT NOT NULL,
		secure BOOLEAN DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createJobsTable = `CREATE TABLE IF NOT EXISTS _jobs (
		_id TEXT PRIMARY KEY,
		function TEXT NOT NULL,
		payload JSONB DEFAULT '{}',
		status TEXT DEFAULT 'pending',
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		scheduled_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		started_at TIMESTAMP WITH TIME ZONE,
		completed_at TIMESTAMP WITH TIME ZONE,
		error_message TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createSecretAuditTable = `CREATE TABLE IF NOT EXISTS _secret_audit (
		_id TEXT PRIMARY KEY,
		secret_name TEXT NOT NULL,
		action TEXT NOT NULL,
		accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		context JSONB DEFAULT '{}'
	)`

	// Domain tables (same as app but with extra app_name column in sessions)
	createDomainSessionsTable = `CREATE TABLE IF NOT EXISTS _sessions (
		_id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES _users(_id) ON DELETE CASCADE,
		app_name TEXT NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		meta JSONB DEFAULT '{}',
		meta_app JSONB DEFAULT '{}'
	)`
)

// BootstrapApp creates all reserved tables for an app database
func (db *dbImpl) BootstrapApp(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		createMigrationsTable,
		createUsersTable,
		createSessionsTable,
		createSecretsTable,
		createApiKeysTable,
		createPoliciesTable,
		createFilesTable,
		createJobsTable,
		createSecretAuditTable,
	}

	for _, ddl := range tables {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("failed to create app table: %w", err)
		}
	}

	slog.Info("bootstrapped app database with reserved tables")
	return nil
}

// BootstrapDomain creates all reserved tables for a domain database
func (db *dbImpl) BootstrapDomain(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		createMigrationsTable,
		createUsersTable,
		createDomainSessionsTable,
		createSecretsTable,
	}

	for _, ddl := range tables {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("failed to create domain table: %w", err)
		}
	}

	slog.Info("bootstrapped domain database with reserved tables")
	return nil
}

// Bootstrap creates the database and bootstraps it with reserved tables
func (db *dbImpl) Bootstrap(ctx context.Context, name string, dbType DBType) error {
	// Get or create connection pool
	pool, err := db.Pool(name)
	if err != nil {
		return fmt.Errorf("failed to get pool for %s: %w", name, err)
	}

	switch dbType {
	case DBTypeApp:
		return db.BootstrapApp(ctx, pool)
	case DBTypeDomain:
		return db.BootstrapDomain(ctx, pool)
	default:
		return fmt.Errorf("unknown database type: %d", dbType)
	}
}
