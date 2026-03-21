//go:build integration

package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/backd-dev/backd/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// getTestDSN returns a DSN for testing
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:5432/backd_test?sslmode=disable"
	}
	return dsn
}

// setupTestDB creates a test database
func setupTestDB(t *testing.T, name string) *pgxpool.Pool {
	ctx := context.Background()
	
	// Connect to postgres to create test database
	serverDSN := fmt.Sprintf("postgres://postgres:password@localhost:5432/postgres?sslmode=disable")
	pool, err := pgxpool.New(ctx, serverDSN)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	// Drop test database if it exists
	_, err = pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
	if err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}

	// Create test database
	_, err = pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", name))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to test database
	testDSN := fmt.Sprintf("postgres://postgres:password@localhost:5432/%s?sslmode=disable", name)
	testPool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return testPool
}

// cleanupTestDB drops the test database
func cleanupTestDB(t *testing.T, name string) {
	ctx := context.Background()
	
	serverDSN := fmt.Sprintf("postgres://postgres:password@localhost:5432/postgres?sslmode=disable")
	pool, err := pgxpool.New(ctx, serverDSN)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	// Kill connections to the test database
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, name))
	if err != nil {
		t.Logf("Warning: failed to kill connections: %v", err)
	}

	// Drop test database
	_, err = pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
	if err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}
}

func TestProvisionApp(t *testing.T) {
	dbName := "backd_test_app"
	defer cleanupTestDB(t, dbName)

	// Create test config
	configSet := config.NewConfigSet()
	serverCfg := &config.ServerConfig{
		DatabaseURL: getTestDSN(),
	}

	appConfig := &config.AppConfig{
		Name: "testapp",
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			SSLMode:  "disable",
		},
	}
	configSet.AddApp("testapp", appConfig)

	// Create DB instance
	db := NewDB(configSet, serverCfg)

	// Test provisioning
	ctx := context.Background()
	err := db.Provision(ctx, "testapp", DBTypeApp)
	if err != nil {
		t.Fatalf("Failed to provision app database: %v", err)
	}

	// Verify database exists by connecting to it
	testDSN := fmt.Sprintf("postgres://postgres:password@localhost:5432/%s?sslmode=disable", dbName)
	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to provisioned database: %v", err)
	}
	defer pool.Close()
}

func TestBootstrapApp(t *testing.T) {
	dbName := "backd_test_bootstrap"
	defer cleanupTestDB(t, dbName)

	// Create test database
	pool := setupTestDB(t, dbName)
	defer pool.Close()

	// Create test config
	configSet := config.NewConfigSet()
	serverCfg := &config.ServerConfig{
		DatabaseURL: getTestDSN(),
	}

	appConfig := &config.AppConfig{
		Name: "testapp",
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			SSLMode:  "disable",
		},
	}
	configSet.AddApp("testapp", appConfig)

	// Create DB instance
	db := NewDB(configSet, serverCfg)

	// Test bootstrapping
	ctx := context.Background()
	err := db.Bootstrap(ctx, "testapp", DBTypeApp)
	if err != nil {
		t.Fatalf("Failed to bootstrap app database: %v", err)
	}

	// Verify tables exist
	var tableCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name LIKE '\_%'
	`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}

	expectedTables := 9 // _migrations, _users, _sessions, _secrets, _api_keys, _policies, _files, _jobs, _secret_audit
	if tableCount != expectedTables {
		t.Errorf("Expected %d tables, got %d", expectedTables, tableCount)
	}
}

func TestMigrate(t *testing.T) {
	dbName := "backd_test_migrate"
	defer cleanupTestDB(t, dbName)

	// Create test database
	pool := setupTestDB(t, dbName)
	defer pool.Close()

	// Create test config
	configSet := config.NewConfigSet()
	serverCfg := &config.ServerConfig{
		DatabaseURL: getTestDSN(),
	}

	appConfig := &config.AppConfig{
		Name: "testapp",
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			SSLMode:  "disable",
		},
	}
	configSet.AddApp("testapp", appConfig)

	// Create DB instance
	db := NewDB(configSet, serverCfg)

	// Bootstrap first
	ctx := context.Background()
	err := db.Bootstrap(ctx, "testapp", DBTypeApp)
	if err != nil {
		t.Fatalf("Failed to bootstrap app database: %v", err)
	}

	// Create a temporary migration file
	tempDir := t.TempDir()
	migrationSQL := `
		CREATE TABLE test_table (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	migrationFile := fmt.Sprintf("%s/001_test.sql", tempDir)
	err = os.WriteFile(migrationFile, []byte(migrationSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write migration file: %v", err)
	}

	// Test migration
	err = db.Migrate(ctx, "testapp", tempDir)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Verify migration was applied
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM _migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 migration, got %d", count)
	}

	// Verify table was created
	var tableExists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'test_table')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	if !tableExists {
		t.Error("Migration table was not created")
	}
}

func TestQueryMethods(t *testing.T) {
	dbName := "backd_test_query"
	defer cleanupTestDB(t, dbName)

	// Create test database
	pool := setupTestDB(t, dbName)
	defer pool.Close()

	// Create test config
	configSet := config.NewConfigSet()
	serverCfg := &config.ServerConfig{
		DatabaseURL: getTestDSN(),
	}

	appConfig := &config.AppConfig{
		Name: "testapp",
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			SSLMode:  "disable",
		},
	}
	configSet.AddApp("testapp", appConfig)

	// Create DB instance
	db := NewDB(configSet, serverCfg)

	// Bootstrap first
	ctx := context.Background()
	err := db.Bootstrap(ctx, "testapp", DBTypeApp)
	if err != nil {
		t.Fatalf("Failed to bootstrap app database: %v", err)
	}

	// Create a test table
	_, err = pool.Exec(ctx, `
		CREATE TABLE test_data (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test Exec
	err = db.Exec(ctx, "testapp", "INSERT INTO test_data (id, name, value) VALUES ($1, $2, $3)", "1", "test", 42)
	if err != nil {
		t.Fatalf("Failed to exec insert: %v", err)
	}

	// Test Query
	rows, err := db.Query(ctx, "testapp", "SELECT id, name, value FROM test_data WHERE name = $1", "test")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row["name"] != "test" || row["value"] != int64(42) {
		t.Errorf("Unexpected row data: %+v", row)
	}

	// Test QueryOne
	row, err = db.QueryOne(ctx, "testapp", "SELECT id, name, value FROM test_data WHERE id = $1", "1")
	if err != nil {
		t.Fatalf("Failed to query one: %v", err)
	}

	if row["name"] != "test" || row["value"] != int64(42) {
		t.Errorf("Unexpected row data: %+v", row)
	}
}

func TestIntrospection(t *testing.T) {
	dbName := "backd_test_introspect"
	defer cleanupTestDB(t, dbName)

	// Create test database
	pool := setupTestDB(t, dbName)
	defer pool.Close()

	// Create test config
	configSet := config.NewConfigSet()
	serverCfg := &config.ServerConfig{
		DatabaseURL: getTestDSN(),
	}

	appConfig := &config.AppConfig{
		Name: "testapp",
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			SSLMode:  "disable",
		},
	}
	configSet.AddApp("testapp", appConfig)

	// Create DB instance
	db := NewDB(configSet, serverCfg)

	// Bootstrap first
	ctx := context.Background()
	err := db.Bootstrap(ctx, "testapp", DBTypeApp)
	if err != nil {
		t.Fatalf("Failed to bootstrap app database: %v", err)
	}

	// Create some test tables
	_, err = pool.Exec(ctx, `
		CREATE TABLE public_table (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			file_data__file TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE _private_table (
			id TEXT PRIMARY KEY,
			secret TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	// Test Tables - should exclude _private_table
	tables, err := db.Tables(ctx, "testapp")
	if err != nil {
		t.Fatalf("Failed to get tables: %v", err)
	}

	if len(tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if table.Name != "public_table" {
		t.Errorf("Expected table name 'public_table', got '%s'", table.Name)
	}

	// Test Columns
	columns, err := db.Columns(ctx, "testapp", "public_table")
	if err != nil {
		t.Fatalf("Failed to get columns: %v", err)
	}

	expectedColumns := 4 // id, name, file_data__file, created_at
	if len(columns) != expectedColumns {
		t.Errorf("Expected %d columns, got %d", expectedColumns, len(columns))
	}

	// Check file column detection
	var fileColumnFound bool
	for _, col := range columns {
		if col.Name == "file_data__file" && col.IsFile {
			fileColumnFound = true
		}
	}

	if !fileColumnFound {
		t.Error("File column not detected properly")
	}
}

func TestXIDGeneration(t *testing.T) {
	// Test that NewXID generates unique IDs
	id1 := NewXID()
	id2 := NewXID()

	if id1 == id2 {
		t.Error("NewXID should generate unique IDs")
	}

	if len(id1) != 20 { // xid.String() returns 20 characters
		t.Errorf("Expected xid length 20, got %d", len(id1))
	}
}
