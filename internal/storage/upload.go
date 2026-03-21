package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/backd-dev/backd/internal/db"
)

// detectContentType detects the MIME type of a file
func detectContentType(filename string, body io.Reader) string {
	// Try to detect from the first 512 bytes
	buffer := make([]byte, 512)
	n, err := body.Read(buffer)
	if err != nil {
		// If we can't read, fall back to a default
		return "application/octet-stream"
	}

	// Create a new reader that includes the buffer and the rest
	body = io.MultiReader(
		strings.NewReader(string(buffer[:n])),
		body,
	)

	contentType := http.DetectContentType(buffer[:n])
	return contentType
}

// Upload streams a file to S3 and creates a database record
func (s *storageImpl) Upload(ctx context.Context, appName, filename string, secure bool, body io.Reader) (*File, error) {
	fileID := db.NewXID()
	slog.Info("starting file upload", "app", appName, "file_id", fileID, "filename", filename, "secure", secure)

	// Get app config
	appCfg, exists := s.configs.GetApp(appName)
	if !exists {
		return nil, fmt.Errorf("app %q not found", appName)
	}

	if appCfg.Storage == nil {
		return nil, ErrStorageDisabled
	}

	// Get S3 client
	client, err := s.clients.getClient(ctx, appName, appCfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 client: %w", err)
	}

	// Generate S3 key
	s3Key := generateS3Key(appName, fileID, filename)

	// Detect content type
	contentType := detectContentType(filename, body)

	// Upload to S3
	slog.Info("uploading to S3", "key", s3Key, "bucket", appCfg.Storage.Bucket)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(appCfg.Storage.Bucket),
		Key:         aws.String(s3Key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Get file size (we need to track this during upload)
	// For now, we'll get it from S3 metadata after upload
	headResp, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(appCfg.Storage.Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		// Try to delete from S3 since DB insert will fail
		_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(appCfg.Storage.Bucket),
			Key:    aws.String(s3Key),
		})
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Create database record
	now := time.Now()
	file := &File{
		ID:        fileID,
		Filename:  filename,
		MimeType:  contentType,
		Size:      *headResp.ContentLength,
		Secure:    secure,
		Key:       s3Key,
		Bucket:    appCfg.Storage.Bucket,
		CreatedAt: now,
	}

	// Insert into database
	err = s.db.Exec(ctx, appName, `
		INSERT INTO _files (_id, filename, content_type, size_bytes, storage_key, secure, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, fileID, filename, contentType, headResp.ContentLength, s3Key, secure, now, now)
	if err != nil {
		// Try to delete from S3 on database failure
		slog.Error("database insert failed, cleaning up S3", "error", err)
		_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(appCfg.Storage.Bucket),
			Key:    aws.String(s3Key),
		})
		return nil, fmt.Errorf("failed to insert file record: %w", err)
	}

	slog.Info("file uploaded successfully", "file_id", fileID, "size", headResp.ContentLength)
	return file, nil
}

// Delete removes a file from S3 and the database
func (s *storageImpl) Delete(ctx context.Context, appName, fileID string) error {
	slog.Info("deleting file", "app", appName, "file_id", fileID)

	// Get app config
	appCfg, exists := s.configs.GetApp(appName)
	if !exists {
		return fmt.Errorf("app %q not found", appName)
	}

	if appCfg.Storage == nil {
		return ErrStorageDisabled
	}

	// Get file record from database
	fileRecord, err := s.db.QueryOne(ctx, appName, `
		SELECT storage_key, bucket FROM _files WHERE _id = $1
	`, fileID)
	if err != nil {
		return fmt.Errorf("failed to query file record: %w", err)
	}

	if fileRecord == nil {
		return ErrFileNotFound
	}

	storageKey, ok := fileRecord["storage_key"].(string)
	if !ok {
		return fmt.Errorf("invalid storage_key in database")
	}

	// Get S3 client
	client, err := s.clients.getClient(ctx, appName, appCfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to get S3 client: %w", err)
	}

	// Delete from S3
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(appCfg.Storage.Bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		slog.Error("failed to delete from S3", "error", err, "key", storageKey)
		// Continue with database deletion even if S3 deletion fails
	}

	// Delete from database
	err = s.db.Exec(ctx, appName, `DELETE FROM _files WHERE _id = $1`, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	slog.Info("file deleted successfully", "file_id", fileID)
	return nil
}
