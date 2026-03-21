package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
)

// Storage interface defines file storage operations
type Storage interface {
	// Upload streams a file to S3 and creates a database record
	Upload(ctx context.Context, appName, filename string, secure bool, body io.Reader) (*File, error)

	// Delete removes a file from S3 and the database
	Delete(ctx context.Context, appName, fileID string) error

	// ResolveFiles resolves all __file columns in query results
	ResolveFiles(ctx context.Context, appName string, rows []map[string]any) ([]map[string]any, error)
}

// File represents a file record in the _files table
type File struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	Secure    bool      `json:"secure"`
	Key       string    `json:"key"` // S3 key: <app>/<id>/<filename>
	Bucket    string    `json:"bucket"`
	CreatedAt time.Time `json:"created_at"`
}

// FileDescriptor represents a file in API responses
type FileDescriptor struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	Secure    bool   `json:"secure"`
	URL       string `json:"url"`
	ExpiresIn *int   `json:"expires_in,omitempty"` // seconds, only for secure=true
}

// Custom errors
var (
	ErrStorageDisabled = fmt.Errorf("storage is disabled for this app")
	ErrFileNotFound    = fmt.Errorf("file not found")
	ErrInvalidConfig   = fmt.Errorf("invalid storage configuration")
)

// storageImpl implements the Storage interface
type storageImpl struct {
	db      db.DB
	clients *clientRegistry
	configs *config.ConfigSet
}

// NewStorage creates a new Storage instance
func NewStorage(database db.DB, configs *config.ConfigSet) Storage {
	return &storageImpl{
		db:      database,
		clients: newClientRegistry(),
		configs: configs,
	}
}

// Helper function to generate S3 key
func generateS3Key(appName, fileID, filename string) string {
	return fmt.Sprintf("%s/%s/%s", appName, fileID, filename)
}

// Helper function to detect __file columns
func isFileColumn(colName string) bool {
	return strings.HasSuffix(colName, "__file")
}
