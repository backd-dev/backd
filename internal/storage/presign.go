package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/backd-dev/backd/internal/config"
)

// PresignURL generates a presigned URL for secure file access
func (s *storageImpl) PresignURL(ctx context.Context, appName, fileID string) (string, error) {
	// Get app config
	appCfg, exists := s.configs.GetApp(appName)
	if !exists {
		return "", fmt.Errorf("app %q not found", appName)
	}

	if appCfg.Storage == nil {
		return "", ErrStorageDisabled
	}

	// Get file record to get storage key
	fileRecord, err := s.db.QueryOne(ctx, appName, `
		SELECT storage_key, secure FROM _files WHERE id = $1
	`, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to query file record: %w", err)
	}

	if fileRecord == nil {
		return "", ErrFileNotFound
	}

	storageKey, ok := fileRecord["storage_key"].(string)
	if !ok {
		return "", fmt.Errorf("invalid storage_key in database")
	}

	secure, _ := fileRecord["secure"].(bool)
	if !secure {
		// Non-secure files should use direct URL
		return s.directURL(ctx, nil, appCfg.Storage, storageKey)
	}

	// Get S3 client
	client, err := s.clients.getClient(ctx, appName, appCfg.Storage)
	if err != nil {
		return "", fmt.Errorf("failed to get S3 client: %w", err)
	}

	return s.presignURL(ctx, client, appCfg.Storage, storageKey)
}

// DirectURL generates a direct URL for non-secure file access
func (s *storageImpl) DirectURL(ctx context.Context, appName, fileID string) (string, error) {
	// Get app config
	appCfg, exists := s.configs.GetApp(appName)
	if !exists {
		return "", fmt.Errorf("app %q not found", appName)
	}

	if appCfg.Storage == nil {
		return "", ErrStorageDisabled
	}

	// Get file record to get storage key
	fileRecord, err := s.db.QueryOne(ctx, appName, `
		SELECT storage_key, secure FROM _files WHERE id = $1
	`, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to query file record: %w", err)
	}

	if fileRecord == nil {
		return "", ErrFileNotFound
	}

	storageKey, ok := fileRecord["storage_key"].(string)
	if !ok {
		return "", fmt.Errorf("invalid storage_key in database")
	}

	secure, _ := fileRecord["secure"].(bool)
	if secure {
		// Secure files should use presigned URL
		return s.presignURL(ctx, nil, appCfg.Storage, storageKey)
	}

	// Get S3 client
	client, err := s.clients.getClient(ctx, appName, appCfg.Storage)
	if err != nil {
		return "", fmt.Errorf("failed to get S3 client: %w", err)
	}

	return s.directURL(ctx, client, appCfg.Storage, storageKey)
}

// presignURL generates a presigned URL for S3 object access
func (s *storageImpl) presignURL(ctx context.Context, client *s3.Client, cfg *config.StorageConfig, storageKey string) (string, error) {
	if client == nil {
		var err error
		client, err = s.clients.getClient(ctx, "", cfg) // We don't have appName here, but this shouldn't happen
		if err != nil {
			return "", fmt.Errorf("failed to get S3 client: %w", err)
		}
	}

	// Default expiry is 1 hour, can be configured via StorageConfig.PresignExpiry
	expiry := cfg.PresignExpiry
	if expiry == 0 {
		expiry = time.Hour
	}

	// Create presigned client
	presignClient := s3.NewPresignClient(client)

	// Generate presigned GET request
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(storageKey),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	slog.Info("generated presigned URL", "key", storageKey, "expiry_seconds", int(expiry.Seconds()))
	return req.URL, nil
}

// directURL generates a direct URL for S3 object access
func (s *storageImpl) directURL(ctx context.Context, client *s3.Client, cfg *config.StorageConfig, storageKey string) (string, error) {
	if client == nil {
		var err error
		client, err = s.clients.getClient(ctx, "", cfg) // We don't have appName here, but this shouldn't happen
		if err != nil {
			return "", fmt.Errorf("failed to get S3 client: %w", err)
		}
	}

	// For S3-compatible services, construct direct URL
	// Format depends on the endpoint configuration
	if cfg.Endpoint != "" {
		// Custom endpoint (MinIO, Cloudflare R2, etc.)
		return fmt.Sprintf("%s/%s/%s", cfg.Endpoint, cfg.Bucket, storageKey), nil
	} else {
		// AWS S3
		region := cfg.Region
		if region == "" {
			region = "us-east-1" // Default region
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.Bucket, region, storageKey), nil
	}
}
