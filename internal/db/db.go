package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/backd-dev/backd/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBType represents the type of database
type DBType int

const (
	DBTypeApp DBType = iota
	DBTypeDomain
)

// TableInfo represents table information
type TableInfo struct {
	Name    string
	Columns []ColumnInfo
}

// ColumnInfo represents column information
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	IsFile   bool // true if column name ends with "__file"
}

// DB interface defines all database operations
type DB interface {
	Provision(ctx context.Context, name string, dbType DBType) error
	Bootstrap(ctx context.Context, name string, dbType DBType) error
	Migrate(ctx context.Context, appName, migrationsPath string) error
	Pool(name string) (*pgxpool.Pool, error)
	Exec(ctx context.Context, app, query string, args ...any) error
	Query(ctx context.Context, app, query string, args ...any) ([]map[string]any, error)
	QueryOne(ctx context.Context, app, query string, args ...any) (map[string]any, error)
	Tables(ctx context.Context, appName string) ([]TableInfo, error)
	Columns(ctx context.Context, appName, table string) ([]ColumnInfo, error)
	VerifyPublishableKey(ctx context.Context, appName, key string) error
	EnsureSecretKey(ctx context.Context, appName string, s Secrets) error
}

// Secrets interface for key management (will be implemented by secrets package)
type Secrets interface {
	GenerateKey() ([]byte, error)
}

// dbImpl implements the DB interface
type dbImpl struct {
	configSet *config.ConfigSet
	serverCfg *config.ServerConfig
	pools     map[string]*pgxpool.Pool
	poolsMu   sync.RWMutex
}

// NewDB creates a new database instance
func NewDB(configSet *config.ConfigSet, serverCfg *config.ServerConfig) DB {
	return &dbImpl{
		configSet: configSet,
		serverCfg: serverCfg,
		pools:     make(map[string]*pgxpool.Pool),
	}
}

// resolveAppDSN resolves the database connection string for an app
func (db *dbImpl) resolveAppDSN(appName string) (string, error) {
	if db.configSet == nil {
		return "", fmt.Errorf("config set is nil")
	}

	appCfg, exists := db.configSet.GetApp(appName)
	if !exists {
		return "", fmt.Errorf("app %q not found in config", appName)
	}

	cfg := appCfg.Database
	switch {
	case cfg.DSN != "":
		return cfg.DSN, nil
	case cfg.Host != "":
		return buildAppDSN(cfg, appName), nil
	case db.serverCfg != nil && db.serverCfg.DatabaseURL != "":
		return db.serverCfg.DatabaseURL, nil
	default:
		return "", fmt.Errorf("no database config for app %s", appName)
	}
}

// resolveDomainDSN resolves the database connection string for a domain
func (db *dbImpl) resolveDomainDSN(domainName string) (string, error) {
	if db.configSet == nil {
		return "", fmt.Errorf("config set is nil")
	}

	domainCfg, exists := db.configSet.GetDomain(domainName)
	if !exists {
		return "", fmt.Errorf("domain %q not found in config", domainName)
	}

	cfg := domainCfg.Database
	switch {
	case cfg.DSN != "":
		return cfg.DSN, nil
	case cfg.Host != "":
		return buildDomainDSN(cfg, domainName), nil
	case db.serverCfg != nil && db.serverCfg.DatabaseURL != "":
		return db.serverCfg.DatabaseURL, nil
	default:
		return "", fmt.Errorf("no database config for domain %s", domainName)
	}
}

// buildAppDSN builds a DSN from host configuration for an app
func buildAppDSN(cfg config.DatabaseConfig, appName string) string {
	dbName := fmt.Sprintf("backd_%s", appName)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		dbName,
	)

	// Add SSL mode if specified
	if cfg.SSLMode != "" {
		dsn += fmt.Sprintf("?sslmode=%s", cfg.SSLMode)
	}

	return dsn
}

// buildDomainDSN builds a DSN from host configuration for a domain
func buildDomainDSN(cfg config.DatabaseConfig, domainName string) string {
	dbName := fmt.Sprintf("backd_domain_%s", domainName)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		dbName,
	)

	// Add SSL mode if specified
	if cfg.SSLMode != "" {
		dsn += fmt.Sprintf("?sslmode=%s", cfg.SSLMode)
	}

	return dsn
}
