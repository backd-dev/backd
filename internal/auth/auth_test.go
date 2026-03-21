package auth

import (
	"context"
	"testing"

	"github.com/backd-dev/backd/internal/celql"
	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/google/cel-go/cel"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mockDB implements the DB interface for testing
type mockDB struct {
	data map[string]map[string][]map[string]any
}

func newMockDB() *mockDB {
	return &mockDB{
		data: make(map[string]map[string][]map[string]any),
	}
}

func (m *mockDB) Exec(ctx context.Context, app, query string, args ...any) error {
	// Simple mock implementation
	return nil
}

func (m *mockDB) Query(ctx context.Context, app, query string, args ...any) ([]map[string]any, error) {
	// Simple mock implementation
	return []map[string]any{}, nil
}

func (m *mockDB) QueryOne(ctx context.Context, app, query string, args ...any) (map[string]any, error) {
	// Simple mock implementation
	return nil, nil
}

func (m *mockDB) Pool(name string) (*pgxpool.Pool, error) {
	return nil, nil
}

func (m *mockDB) Provision(ctx context.Context, name string, dbType db.DBType) error {
	return nil
}

func (m *mockDB) Bootstrap(ctx context.Context, name string, dbType db.DBType) error {
	return nil
}

func (m *mockDB) Migrate(ctx context.Context, appName, migrationsPath string) error {
	return nil
}

func (m *mockDB) Tables(ctx context.Context, appName string) ([]db.TableInfo, error) {
	return nil, nil
}

func (m *mockDB) Columns(ctx context.Context, appName, table string) ([]db.ColumnInfo, error) {
	return nil, nil
}

func (m *mockDB) VerifyPublishableKey(ctx context.Context, appName, key string) error {
	return nil
}

func (m *mockDB) EnsureSecretKey(ctx context.Context, appName string, s db.Secrets) error {
	return nil
}

// mockCELQL implements the CELQL interface for testing
type mockCELQL struct {
	ast *cel.Ast
}

func newMockCELQL() *mockCELQL {
	return &mockCELQL{}
}

func (m *mockCELQL) Parse(expression string) (*cel.Ast, error) {
	// Return a mock AST
	return m.ast, nil
}

func (m *mockCELQL) Validate(ast *cel.Ast) error {
	return nil
}

func (m *mockCELQL) Transpile(ast *cel.Ast, auth celql.AuthContext) (celql.TranspileResult, error) {
	return celql.TranspileResult{
		SQL:    "TRUE",
		Params: []any{},
	}, nil
}

func TestNewAuth(t *testing.T) {
	db := newMockDB()
	celql := newMockCELQL()

	auth := NewAuth(db, celql)
	if auth == nil {
		t.Fatal("NewAuth returned nil")
	}

	// Test that the returned value implements the Auth interface
	var _ Auth = auth
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "test-password-123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "long password",
			password: "this-is-a-very-long-password-with-special-characters-!@#$%^&*()",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash for valid password")
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	// Create a test hash
	password := "test-password-123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to create test hash: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrong-password",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty hash",
			password: password,
			hash:     "",
			want:     false,
		},
		{
			name:     "invalid hash format",
			password: password,
			hash:     "invalid-hash-format",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyPassword(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasswordHashingConsistency(t *testing.T) {
	password := "test-password-123"

	// Hash the same password multiple times
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash password: %v, %v", err1, err2)
	}

	// Hashes should be different (different salts)
	if hash1 == hash2 {
		t.Error("Hashes should be different due to random salt")
	}

	// But both should verify correctly
	if !VerifyPassword(password, hash1) {
		t.Error("First hash should verify correctly")
	}
	if !VerifyPassword(password, hash2) {
		t.Error("Second hash should verify correctly")
	}
}

func TestLoadPolicies(t *testing.T) {
	auth := NewAuth(newMockDB(), newMockCELQL())

	cfg := &config.AppConfig{
		Policies: map[string]config.TablePolicies{
			"users": {
				"select": config.PolicyEntry{
					Expression: "auth.uid == row.user_id",
					Check:      "true",
					Columns:    []string{"id", "username", "created_at"},
					Defaults:   map[string]string{"created_at": "now()"},
					Soft:       "deleted_at",
				},
			},
		},
	}

	err := auth.LoadPolicies(context.Background(), "test-app", cfg)
	if err != nil {
		t.Errorf("LoadPolicies() error = %v", err)
	}
}

func TestEvaluatePolicy(t *testing.T) {
	auth := NewAuth(newMockDB(), newMockCELQL())

	// Load policies first
	cfg := &config.AppConfig{
		Policies: map[string]config.TablePolicies{
			"users": {
				"select": config.PolicyEntry{
					Expression: "auth.uid == row.user_id",
					Check:      "true",
					Columns:    []string{"id", "username"},
					Defaults:   map[string]string{"created_at": "now()"},
					Soft:       "deleted_at",
				},
			},
		},
	}

	err := auth.LoadPolicies(context.Background(), "test-app", cfg)
	if err != nil {
		t.Fatalf("Failed to load policies: %v", err)
	}

	rc := &RequestContext{
		UID:           "user-123",
		Meta:          make(map[string]any),
		MetaApp:       make(map[string]any),
		Authenticated: true,
		KeyType:       "",
	}

	result, err := auth.EvaluatePolicy(context.Background(), "test-app", "users", "select", rc)
	if err != nil {
		t.Errorf("EvaluatePolicy() error = %v", err)
	}

	if result.SQLClause == "" {
		t.Error("EvaluatePolicy() returned empty SQL clause")
	}
}

func TestEvaluatePolicySecretKey(t *testing.T) {
	auth := NewAuth(newMockDB(), newMockCELQL())

	rc := &RequestContext{
		UID:           "user-123",
		Meta:          make(map[string]any),
		MetaApp:       make(map[string]any),
		Authenticated: true,
		KeyType:       string(KeyTypeSecret),
	}

	result, err := auth.EvaluatePolicy(context.Background(), "test-app", "users", "select", rc)
	if err != nil {
		t.Errorf("EvaluatePolicy() error = %v", err)
	}

	// Secret key should bypass RLS
	if result.SQLClause != "TRUE" {
		t.Errorf("EvaluatePolicy() secret key bypass failed, got = %v", result.SQLClause)
	}
}

func TestApplyDefaults(t *testing.T) {
	auth := NewAuth(newMockDB(), newMockCELQL())

	rc := &RequestContext{
		UID: "user-123",
		Meta: map[string]any{
			"department": "engineering",
			"role":       "admin",
		},
		MetaApp:       make(map[string]any),
		Authenticated: true,
		KeyType:       "",
	}

	defaults := map[string]string{
		"user_id":    "auth.uid",
		"department": "auth.meta.department",
		"created_at": "now()",
		"date":       "today()",
		"status":     "active",
	}

	result := auth.(*authImpl).ApplyDefaults(defaults, rc)

	// Check auth.uid resolution
	if result["user_id"] != "user-123" {
		t.Errorf("ApplyDefaults() auth.uid resolution failed, got = %v", result["user_id"])
	}

	// Check auth.meta resolution
	if result["department"] != "engineering" {
		t.Errorf("ApplyDefaults() auth.meta.resolution failed, got = %v", result["department"])
	}

	// Check function calls
	if result["created_at"] != "NOW()" {
		t.Errorf("ApplyDefaults() now() resolution failed, got = %v", result["created_at"])
	}

	if result["date"] != "CURRENT_DATE" {
		t.Errorf("ApplyDefaults() today() resolution failed, got = %v", result["date"])
	}

	// Check literal value
	if result["status"] != "active" {
		t.Errorf("ApplyDefaults() literal value failed, got = %v", result["status"])
	}
}

func TestKeyTypeConstants(t *testing.T) {
	if KeyTypePublishable != "publishable" {
		t.Errorf("KeyTypePublishable = %v, want %v", KeyTypePublishable, "publishable")
	}

	if KeyTypeSecret != "secret" {
		t.Errorf("KeyTypeSecret = %v, want %v", KeyTypeSecret, "secret")
	}
}

func TestPolicyCacheKey(t *testing.T) {
	key1 := policyKey{appName: "test-app", table: "users", operation: "select"}
	key2 := policyKey{appName: "test-app", table: "users", operation: "select"}
	key3 := policyKey{appName: "test-app", table: "users", operation: "insert"}

	// Test equality
	if key1 != key2 {
		t.Error("Identical policy keys should be equal")
	}

	if key1 == key3 {
		t.Error("Different operations should create different keys")
	}
}

// Benchmark tests
func BenchmarkHashPassword(b *testing.B) {
	password := "test-password-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "test-password-123"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyPassword(password, hash)
	}
}
