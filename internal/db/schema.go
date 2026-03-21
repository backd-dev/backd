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
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		email TEXT,
		type TEXT DEFAULT 'user',
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createSessionsTable = `CREATE TABLE IF NOT EXISTS _sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES _users(id) ON DELETE CASCADE,
		app_name TEXT NOT NULL,
		token TEXT UNIQUE NOT NULL,
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
		id TEXT PRIMARY KEY,
		app_name TEXT NOT NULL,
		key_hash TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'publishable',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createPoliciesTable = `CREATE TABLE IF NOT EXISTS _policies (
		table_name TEXT NOT NULL,
		operation TEXT NOT NULL,
		expression TEXT NOT NULL,
		check_expr TEXT,
		columns TEXT[],
		defaults JSONB DEFAULT '{}',
		soft_delete TEXT,
		PRIMARY KEY (table_name, operation)
	)`

	createFilesTable = `CREATE TABLE IF NOT EXISTS _files (
		id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		content_type TEXT,
		size_bytes BIGINT,
		storage_key TEXT NOT NULL,
		bucket TEXT,
		secure BOOLEAN DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createJobsTable = `CREATE TABLE IF NOT EXISTS _jobs (
		id TEXT PRIMARY KEY,
		app_name TEXT NOT NULL,
		function TEXT NOT NULL,
		payload JSONB DEFAULT '{}',
		trigger TEXT DEFAULT 'manual',
		status TEXT DEFAULT 'pending',
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		run_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		started_at TIMESTAMP WITH TIME ZONE,
		completed_at TIMESTAMP WITH TIME ZONE,
		error TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	createSecretAuditTable = `CREATE TABLE IF NOT EXISTS _secret_audit (
		id TEXT PRIMARY KEY,
		secret_name TEXT NOT NULL,
		action TEXT NOT NULL,
		app_name TEXT,
		accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		context JSONB DEFAULT '{}'
	)`

	// Domain tables — sessions have app_name for cross-app session scoping
	createDomainSessionsTable = `CREATE TABLE IF NOT EXISTS _sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES _users(id) ON DELETE CASCADE,
		app_name TEXT NOT NULL,
		token TEXT UNIQUE NOT NULL,
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
