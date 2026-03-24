package db

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Provision creates a new database for the given name and type.
// Retries connecting to PostgreSQL up to 30 times with 2-second intervals
// to handle container orchestration startup ordering.
func (db *dbImpl) Provision(ctx context.Context, name string, dbType DBType) error {
	var dbName string
	var serverDSN string

	switch dbType {
	case DBTypeApp:
		appDSN, err := db.resolveAppDSN(name)
		if err != nil {
			return fmt.Errorf("failed to resolve app DSN: %w", err)
		}
		dbName = fmt.Sprintf("backd_%s", name)
		serverDSN = replaceDatabaseName(appDSN, "postgres")

	case DBTypeDomain:
		domainDSN, err := db.resolveDomainDSN(name)
		if err != nil {
			return fmt.Errorf("failed to resolve domain DSN: %w", err)
		}
		dbName = fmt.Sprintf("backd_domain_%s", name)
		serverDSN = replaceDatabaseName(domainDSN, "postgres")

	default:
		return fmt.Errorf("unknown database type: %d", dbType)
	}

	// Connect to the default postgres database with retries
	var pool *pgxpool.Pool
	var err error
	for attempt := range 30 {
		pool, err = pgxpool.New(ctx, serverDSN)
		if err == nil {
			// Verify the connection is actually usable
			if pingErr := pool.Ping(ctx); pingErr == nil {
				break
			} else {
				pool.Close()
				err = pingErr
			}
		}
		if attempt < 29 {
			slog.Info("waiting for postgres", "attempt", attempt+1, "error", err)
			time.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to connect to postgres server after retries: %w", err)
	}
	defer pool.Close()

	// Create the database
	createSQL := fmt.Sprintf("CREATE DATABASE %s", dbName)
	if _, err := pool.Exec(ctx, createSQL); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			slog.Info("database already exists", "name", dbName)
			return nil
		}
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	slog.Info("created database", "name", dbName, "type", dbType)
	return nil
}

// replaceDatabaseName replaces the database name in a DSN using proper URL parsing
func replaceDatabaseName(dsn, newDBName string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		// Fallback: try basic string replacement
		parts := strings.Split(dsn, "/")
		if len(parts) >= 4 {
			parts[len(parts)-1] = newDBName
			return strings.Join(parts, "/")
		}
		return dsn
	}
	u.Path = "/" + newDBName
	return u.String()
}
