package storage

import (
	"context"
	"testing"
	"time"

	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of the DB interface
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Provision(ctx context.Context, name string, dbType db.DBType) error {
	args := m.Called(ctx, name, dbType)
	return args.Error(0)
}

func (m *MockDB) Bootstrap(ctx context.Context, name string, dbType db.DBType) error {
	args := m.Called(ctx, name, dbType)
	return args.Error(0)
}

func (m *MockDB) Migrate(ctx context.Context, appName, migrationsPath string) error {
	args := m.Called(ctx, appName, migrationsPath)
	return args.Error(0)
}

func (m *MockDB) Pool(name string) (*pgxpool.Pool, error) {
	args := m.Called(name)
	return args.Get(0).(*pgxpool.Pool), args.Error(1)
}

func (m *MockDB) Exec(ctx context.Context, app, query string, args ...any) error {
	callArgs := m.Called(ctx, app, query, args)
	return callArgs.Error(0)
}

func (m *MockDB) Query(ctx context.Context, app, query string, args ...any) ([]map[string]any, error) {
	callArgs := m.Called(ctx, app, query, args)
	return callArgs.Get(0).([]map[string]any), callArgs.Error(1)
}

func (m *MockDB) QueryOne(ctx context.Context, app, query string, args ...any) (map[string]any, error) {
	callArgs := m.Called(ctx, app, query, args)
	return callArgs.Get(0).(map[string]any), callArgs.Error(1)
}

func (m *MockDB) Tables(ctx context.Context, appName string) ([]db.TableInfo, error) {
	args := m.Called(ctx, appName)
	return args.Get(0).([]db.TableInfo), args.Error(1)
}

func (m *MockDB) Columns(ctx context.Context, appName, table string) ([]db.ColumnInfo, error) {
	args := m.Called(ctx, appName, table)
	return args.Get(0).([]db.ColumnInfo), args.Error(1)
}

func (m *MockDB) UpsertPublishableKey(ctx context.Context, appName, key string) error {
	args := m.Called(ctx, appName, key)
	return args.Error(0)
}

func (m *MockDB) VerifyPublishableKey(ctx context.Context, appName, key string) error {
	args := m.Called(ctx, appName, key)
	return args.Error(0)
}

func (m *MockDB) EnsureSecretKey(ctx context.Context, appName string, secrets db.Secrets) error {
	args := m.Called(ctx, appName, secrets)
	return args.Error(0)
}

func TestNewStorage(t *testing.T) {
	mockDB := &MockDB{}
	configs := config.NewConfigSet()

	storage := NewStorage(mockDB, configs)

	assert.NotNil(t, storage, "Storage should not be nil")
	assert.Implements(t, (*Storage)(nil), storage, "Should implement Storage interface")
}

func TestGenerateS3Key(t *testing.T) {
	appName := "test_app"
	fileID := "abc123"
	filename := "test.pdf"

	expected := "test_app/abc123/test.pdf"
	actual := generateS3Key(appName, fileID, filename)

	assert.Equal(t, expected, actual, "S3 key should be in correct format")
}

func TestIsFileColumn(t *testing.T) {
	tests := []struct {
		name     string
		colName  string
		expected bool
	}{
		{"file column", "document__file", true},
		{"non-file column", "document_name", false},
		{"file column with underscore prefix", "_private__file", true},
		{"file column with numbers", "image123__file", true},
		{"single underscore", "document_file", false},
		{"triple underscore", "document___file", true},
		{"empty string", "", false},
		{"exactly __file", "__file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isFileColumn(tt.colName)
			assert.Equal(t, tt.expected, actual, "Column detection should match expected")
		})
	}
}

func TestStorageErrors(t *testing.T) {
	assert.Equal(t, "storage is disabled for this app", ErrStorageDisabled.Error())
	assert.Equal(t, "file not found", ErrFileNotFound.Error())
	assert.Equal(t, "invalid storage configuration", ErrInvalidConfig.Error())
}

func TestFileStruct(t *testing.T) {
	now := time.Now()
	file := &File{
		ID:        "test123",
		Filename:  "test.pdf",
		MimeType:  "application/pdf",
		Size:      1024,
		Secure:    true,
		Key:       "test_app/test123/test.pdf",
		Bucket:    "test-bucket",
		CreatedAt: now,
	}

	assert.Equal(t, "test123", file.ID)
	assert.Equal(t, "test.pdf", file.Filename)
	assert.Equal(t, "application/pdf", file.MimeType)
	assert.Equal(t, int64(1024), file.Size)
	assert.True(t, file.Secure)
	assert.Equal(t, "test_app/test123/test.pdf", file.Key)
	assert.Equal(t, "test-bucket", file.Bucket)
	assert.Equal(t, now, file.CreatedAt)
}

func TestFileDescriptorStruct(t *testing.T) {
	expiresIn := 3600
	desc := &FileDescriptor{
		ID:        "test123",
		Filename:  "test.pdf",
		MimeType:  "application/pdf",
		Size:      1024,
		Secure:    true,
		URL:       "https://example.com/test.pdf",
		ExpiresIn: &expiresIn,
	}

	assert.Equal(t, "test123", desc.ID)
	assert.Equal(t, "test.pdf", desc.Filename)
	assert.Equal(t, "application/pdf", desc.MimeType)
	assert.Equal(t, int64(1024), desc.Size)
	assert.True(t, desc.Secure)
	assert.Equal(t, "https://example.com/test.pdf", desc.URL)
	assert.NotNil(t, desc.ExpiresIn)
	assert.Equal(t, 3600, *desc.ExpiresIn)
}
