package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool returns a connection pool for the given app or domain name
// Creates the pool if it doesn't exist
func (db *dbImpl) Pool(name string) (*pgxpool.Pool, error) {
	db.poolsMu.RLock()
	pool, exists := db.pools[name]
	db.poolsMu.RUnlock()

	if exists {
		return pool, nil
	}

	// Need to create the pool
	db.poolsMu.Lock()
	defer db.poolsMu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := db.pools[name]; exists {
		return pool, nil
	}

	// Try to resolve as app first, then as domain
	dsn, err := db.resolveAppDSN(name)
	if err != nil {
		// Try as domain
		dsn, err = db.resolveDomainDSN(name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve DSN for %q: %w", name, err)
		}
	}

	// Create the pool
	pool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool for %q: %w", name, err)
	}

	// Store the pool
	db.pools[name] = pool

	slog.Info("created connection pool", "name", name)
	return pool, nil
}

// CloseAll closes all connection pools
func (db *dbImpl) CloseAll() {
	db.poolsMu.Lock()
	defer db.poolsMu.Unlock()

	for name, pool := range db.pools {
		pool.Close()
		slog.Info("closed connection pool", "name", name)
	}

	db.pools = make(map[string]*pgxpool.Pool)
}
