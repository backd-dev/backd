package secrets

import (
	"context"
	"strings"
	"testing"

	"github.com/backd-dev/backd/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mockDB is a simple mock database implementation
type mockDB struct {
	secrets      map[string]map[string]string // app -> name -> encrypted_value
	auditRecords []map[string]any
}

func newMockDB() *mockDB {
	return &mockDB{
		secrets:      make(map[string]map[string]string),
		auditRecords: make([]map[string]any, 0),
	}
}

func (m *mockDB) Query(ctx context.Context, app, query string, args ...any) ([]map[string]any, error) {
	if appSecrets, exists := m.secrets[app]; exists {
		if secret, exists := appSecrets[args[0].(string)]; exists {
			return []map[string]any{
				{"encrypted_value": secret},
			}, nil
		}
	}
	return []map[string]any{}, nil
}

func (m *mockDB) Exec(ctx context.Context, app, query string, args ...any) error {
	// Handle upsert for secrets
	if query == `
		INSERT INTO _secrets (name, encrypted_value) 
		VALUES ($1, $2) 
		ON CONFLICT (name) DO UPDATE SET 
			encrypted_value = EXCLUDED.encrypted_value,
			updated_at = NOW()
	` {
		if m.secrets[app] == nil {
			m.secrets[app] = make(map[string]string)
		}
		m.secrets[app][args[0].(string)] = args[1].(string)
		return nil
	}

	// Handle delete for secrets
	if query == `DELETE FROM _secrets WHERE name = $1` {
		if appSecrets, exists := m.secrets[app]; exists {
			delete(appSecrets, args[0].(string))
		}
		return nil
	}

	// Handle audit logging
	if query == `
		INSERT INTO _secret_audit (id, secret_name, action, app_name) 
		VALUES ($1, $2, $3, $4)
	` {
		m.auditRecords = append(m.auditRecords, map[string]any{
			"id":          args[0],
			"secret_name": args[1],
			"action":      args[2],
			"app_name":    args[3],
		})
		return nil
	}

	return nil
}

func (m *mockDB) QueryOne(ctx context.Context, app, query string, args ...any) (map[string]any, error) {
	rows, err := m.Query(ctx, app, query, args...)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

func (m *mockDB) Tables(ctx context.Context, appName string) ([]db.TableInfo, error) {
	return nil, nil
}

func (m *mockDB) Columns(ctx context.Context, appName, table string) ([]db.ColumnInfo, error) {
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

func (m *mockDB) VerifyPublishableKey(ctx context.Context, appName, key string) error {
	return nil
}

func (m *mockDB) EnsureSecretKey(ctx context.Context, appName string, s db.Secrets) error {
	return nil
}

func TestDeriveAppKey(t *testing.T) {
	master := []byte("test-master-key-32-bytes-long")

	// Test that same inputs produce same key (deterministic)
	key1 := DeriveAppKey(master, "app_a")
	key2 := DeriveAppKey(master, "app_a")

	if string(key1) != string(key2) {
		t.Error("DeriveAppKey should be deterministic")
	}

	// Test that different apps produce different keys
	key3 := DeriveAppKey(master, "app_b")
	if string(key1) == string(key3) {
		t.Error("Different apps should produce different keys")
	}

	// Test key length (should be 32 bytes)
	if len(key1) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key1))
	}
}

func TestDeriveDomainKey(t *testing.T) {
	master := []byte("test-master-key-32-bytes-long")

	// Test that same inputs produce same key (deterministic)
	key1 := DeriveDomainKey(master, "domain_a")
	key2 := DeriveDomainKey(master, "domain_a")

	if string(key1) != string(key2) {
		t.Error("DeriveDomainKey should be deterministic")
	}

	// Test that different domains produce different keys
	key3 := DeriveDomainKey(master, "domain_b")
	if string(key1) == string(key3) {
		t.Error("Different domains should produce different keys")
	}

	// Test key length (should be 32 bytes)
	if len(key1) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key1))
	}

	// Test that domain keys are different from app keys
	appKey := DeriveAppKey(master, "domain_a")
	if string(key1) == string(appKey) {
		t.Error("Domain keys should be different from app keys")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := "secret message"

	// Test encrypt/decrypt roundtrip
	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Roundtrip failed: expected %s, got %s", plaintext, decrypted)
	}

	// Test that two encrypt calls produce different ciphertexts (random nonce)
	ciphertext2, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Second Encrypt failed: %v", err)
	}

	if ciphertext == ciphertext2 {
		t.Error("Two encrypt calls should produce different ciphertexts")
	}

	// Test decrypt with wrong key fails
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(i + 1)
	}

	_, err = Decrypt(wrongKey, ciphertext)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}

func TestSecretsGetSetDelete(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long")
	mockDB := newMockDB()
	secrets := NewSecrets(mockDB, masterKey)

	ctx := context.Background()
	appName := "test_app"
	secretName := "test_secret"
	secretValue := "super-secret-value"

	// Test Get on non-existent secret
	_, err := secrets.Get(ctx, appName, secretName)
	if err != ErrSecretNotFound {
		t.Errorf("Expected ErrSecretNotFound, got %v", err)
	}

	// Test Set
	err = secrets.Set(ctx, appName, secretName, secretValue)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get after Set
	retrieved, err := secrets.Get(ctx, appName, secretName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved != secretValue {
		t.Errorf("Expected %s, got %s", secretValue, retrieved)
	}

	// Test audit logging for Get and Set
	if len(mockDB.auditRecords) != 2 {
		t.Errorf("Expected 2 audit records, got %d", len(mockDB.auditRecords))
	}

	// Verify audit record contents for Set and Get (Set is logged first)
	setRecord := mockDB.auditRecords[0]
	if setRecord["secret_name"] != secretName || setRecord["action"] != "set" {
		t.Errorf("Set audit record incorrect: got name=%v, action=%v", setRecord["secret_name"], setRecord["action"])
	}

	getRecord := mockDB.auditRecords[1]
	if getRecord["secret_name"] != secretName || getRecord["action"] != "get" {
		t.Errorf("Get audit record incorrect: got name=%v, action=%v", getRecord["secret_name"], getRecord["action"])
	}

	// Test Delete
	err = secrets.Delete(ctx, appName, secretName)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Test Get after Delete
	_, err = secrets.Get(ctx, appName, secretName)
	if err != ErrSecretNotFound {
		t.Errorf("Expected ErrSecretNotFound after delete, got %v", err)
	}

	// Test audit logging for Delete (should be 4 total: get, set, get (from delete check), delete)
	if len(mockDB.auditRecords) != 4 {
		t.Errorf("Expected 4 audit records, got %d", len(mockDB.auditRecords))
	}

	// Verify delete audit record
	deleteRecord := mockDB.auditRecords[3]
	if deleteRecord["secret_name"] != secretName || deleteRecord["action"] != "delete" {
		t.Error("Delete audit record incorrect")
	}
}

func TestSecretsDeleteNonExistent(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long")
	mockDB := newMockDB()
	secrets := NewSecrets(mockDB, masterKey)

	ctx := context.Background()
	appName := "test_app"
	secretName := "non_existent_secret"

	// Test Delete on non-existent secret
	err := secrets.Delete(ctx, appName, secretName)
	if err != ErrSecretNotFound {
		t.Errorf("Expected ErrSecretNotFound, got %v", err)
	}
}

func TestLogAccess(t *testing.T) {
	mockDB := newMockDB()
	ctx := context.Background()

	err := LogAccess(ctx, mockDB, "test_app", "test_secret", "get")
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}

	if len(mockDB.auditRecords) != 1 {
		t.Errorf("Expected 1 audit record, got %d", len(mockDB.auditRecords))
	}

	record := mockDB.auditRecords[0]
	if record["secret_name"] != "test_secret" || record["action"] != "get" || record["app_name"] != "test_app" {
		t.Error("Audit record incorrect")
	}
}

func TestNewSecrets(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long")
	mockDB := newMockDB()

	secrets := NewSecrets(mockDB, masterKey)
	if secrets == nil {
		t.Error("NewSecrets returned nil")
	}

	// Test that it implements the interface
	var _ Secrets = secrets
}

func TestSecretsInterfaceMethods(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long")
	mockDB := newMockDB()
	secrets := NewSecrets(mockDB, masterKey)
	ctx := context.Background()

	// Test interface methods directly
	_, ok := secrets.(*secretsImpl)
	if !ok {
		t.Fatal("Failed to type assert to secretsImpl")
	}

	// Test with empty master key
	shortKey := []byte("short")
	shortSecrets := NewSecrets(mockDB, shortKey)

	err := shortSecrets.Set(ctx, "test_app", "test", "value")
	if err != nil {
		t.Errorf("Set should work with short key: %v", err)
	}
}

func TestEncryptDecryptEdgeCases(t *testing.T) {
	key := make([]byte, 32)

	// Test empty string
	ciphertext, err := Encrypt(key, "")
	if err != nil {
		t.Fatalf("Encrypt failed for empty string: %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed for empty string: %v", err)
	}

	if decrypted != "" {
		t.Errorf("Expected empty string, got %s", decrypted)
	}

	// Test short ciphertext
	_, err = Decrypt(key, "short")
	if err == nil {
		t.Error("Decrypt should fail with short ciphertext")
	}

	// Test invalid key (wrong length)
	invalidKey := []byte("short")
	_, err = Encrypt(invalidKey, "test")
	if err == nil {
		t.Error("Encrypt should fail with invalid key length")
	}
}

func TestSecretsInvalidEncryptedValue(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long")
	mockDB := newMockDB()
	secrets := NewSecrets(mockDB, masterKey)
	ctx := context.Background()

	// Manually insert invalid encrypted value
	if mockDB.secrets["test_app"] == nil {
		mockDB.secrets["test_app"] = make(map[string]string)
	}
	mockDB.secrets["test_app"]["test_secret"] = "invalid_encrypted_value"

	// Test Get with invalid encrypted value
	_, err := secrets.Get(ctx, "test_app", "test_secret")
	if err == nil {
		t.Error("Get should fail with invalid encrypted value")
	}

	// Test that error is properly wrapped
	if !containsString(err.Error(), "secrets.Get") {
		t.Error("Error should be wrapped with context")
	}
}

func TestSecretsCoverageEdgeCases(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long")
	mockDB := newMockDB()
	secrets := NewSecrets(mockDB, masterKey)
	ctx := context.Background()

	// Test Delete with Get failure (non-existent secret)
	err := secrets.Delete(ctx, "test_app", "non_existent")
	if err != ErrSecretNotFound {
		t.Errorf("Expected ErrSecretNotFound, got %v", err)
	}

	// Test Delete with Get error (invalid encrypted value)
	if mockDB.secrets["test_app"] == nil {
		mockDB.secrets["test_app"] = make(map[string]string)
	}
	mockDB.secrets["test_app"]["bad_secret"] = "invalid"

	err = secrets.Delete(ctx, "test_app", "bad_secret")
	if err == nil {
		t.Error("Delete should fail with invalid encrypted value")
	}

	// Test that error is properly wrapped
	if !strings.Contains(err.Error(), "secrets.Delete") {
		t.Error("Error should be wrapped with context")
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
