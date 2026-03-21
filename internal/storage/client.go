package storage

import (
	"context"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/backd-dev/backd/internal/config"
)

// clientRegistry manages S3 clients per app
type clientRegistry struct {
	clients map[string]*s3.Client
	mu      sync.RWMutex
}

// newClientRegistry creates a new client registry
func newClientRegistry() *clientRegistry {
	return &clientRegistry{
		clients: make(map[string]*s3.Client),
	}
}

// getClient returns an S3 client for the given app
func (r *clientRegistry) getClient(ctx context.Context, appName string, cfg *config.StorageConfig) (*s3.Client, error) {
	if cfg == nil {
		return nil, ErrStorageDisabled
	}

	r.mu.RLock()
	client, exists := r.clients[appName]
	r.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Create new client
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := r.clients[appName]; exists {
		return client, nil
	}

	// Create S3 client with custom endpoint if specified
	options := s3.Options{
		Region: cfg.Region,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	}

	if cfg.Endpoint != "" {
		options.BaseEndpoint = &cfg.Endpoint
		options.UsePathStyle = true // Required for MinIO compatibility
	}

	client = s3.New(options)

	r.clients[appName] = client
	slog.Info("created S3 client for app", "app", appName, "endpoint", cfg.Endpoint, "bucket", cfg.Bucket)

	return client, nil
}

// removeClient removes an S3 client from the registry
func (r *clientRegistry) removeClient(appName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, appName)
}

// clear removes all clients from the registry
func (r *clientRegistry) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients = make(map[string]*s3.Client)
}
