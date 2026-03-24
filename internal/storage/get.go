package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Get retrieves file metadata and generates access URL
func (s *storageImpl) Get(ctx context.Context, appName, fileID string) (*FileDescriptor, error) {
	// Get app config
	appCfg, exists := s.configs.GetApp(appName)
	if !exists {
		return nil, fmt.Errorf("app %q not found", appName)
	}

	if appCfg.Storage == nil {
		return nil, ErrStorageDisabled
	}

	// Query file record
	fileRecord, err := s.db.QueryOne(ctx, appName, `
		SELECT id, filename, content_type, size_bytes, secure, storage_key, bucket, created_at 
		FROM _files 
		WHERE id = $1
	`, fileID)
	if err != nil {
		slog.Error("failed to query file record", "app", appName, "fileID", fileID, "error", err)
		return nil, fmt.Errorf("failed to query file record: %w", err)
	}

	if fileRecord == nil {
		return nil, ErrFileNotFound
	}

	// Extract file fields
	id, ok := fileRecord["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid file ID in database")
	}

	filename, ok := fileRecord["filename"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid filename in database")
	}

	mimeType, _ := fileRecord["content_type"].(string)
	size, _ := fileRecord["size_bytes"].(int64)
	secure, _ := fileRecord["secure"].(bool)
	_, _ = fileRecord["storage_key"].(string)   // storageKey (not used directly but needed for URL generation)
	_, _ = fileRecord["bucket"].(string)        // bucket (not used directly but needed for URL generation)
	_, _ = fileRecord["created_at"].(time.Time) // createdAt (not used currently)

	// Generate appropriate URL
	var url string
	var expiresIn *int

	if secure {
		// Generate presigned URL for secure files
		presignURL, err := s.PresignURL(ctx, appName, fileID)
		if err != nil {
			slog.Error("failed to generate presigned URL", "app", appName, "fileID", fileID, "error", err)
			return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
		}
		url = presignURL

		// Set expiry time (default 1 hour)
		expiry := appCfg.Storage.PresignExpiry
		if expiry == 0 {
			expiry = time.Hour
		}
		expiresInSec := int(expiry.Seconds())
		expiresIn = &expiresInSec
	} else {
		// Generate direct URL for non-secure files
		directURL, err := s.DirectURL(ctx, appName, fileID)
		if err != nil {
			slog.Error("failed to generate direct URL", "app", appName, "fileID", fileID, "error", err)
			return nil, fmt.Errorf("failed to generate direct URL: %w", err)
		}
		url = directURL
	}

	// Return file descriptor
	return &FileDescriptor{
		ID:        id,
		Filename:  filename,
		MimeType:  mimeType,
		Size:      size,
		Secure:    secure,
		URL:       url,
		ExpiresIn: expiresIn,
	}, nil
}
