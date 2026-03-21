package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
)

// ResolveFiles resolves all __file columns in query results
func (s *storageImpl) ResolveFiles(ctx context.Context, appName string, rows []map[string]any) ([]map[string]any, error) {
	slog.Info("resolving files", "app", appName, "rows", len(rows))

	if len(rows) == 0 {
		return rows, nil
	}

	// Step 1: Scan all rows for columns ending in "__file"
	fileIDs := make([]string, 0)
	fileIDToRows := make(map[string][]fileRef) // fileID -> list of (rowIndex, colName) pairs

	for rowIndex, row := range rows {
		for colName, value := range row {
			if isFileColumn(colName) && value != nil {
				if fileID, ok := value.(string); ok && fileID != "" {
					// Avoid duplicates
					if !slices.Contains(fileIDs, fileID) {
						fileIDs = append(fileIDs, fileID)
					}
					fileIDToRows[fileID] = append(fileIDToRows[fileID], fileRef{
						rowIndex: rowIndex,
						colName:  colName,
					})
				}
			}
		}
	}

	if len(fileIDs) == 0 {
		return rows, nil
	}

	slog.Info("found file references", "unique_files", len(fileIDs), "total_refs", len(fileIDToRows))

	// Step 2: Single query to get all file records
	query := `SELECT _id, filename, content_type, size_bytes, secure, storage_key, bucket, created_at 
	          FROM _files WHERE _id = ANY($1)`

	fileRecords, err := s.db.Query(ctx, appName, query, fileIDs)
	if err != nil {
		return rows, fmt.Errorf("failed to query file records: %w", err)
	}

	// Step 3: Build lookup map and generate URLs
	fileLookup := make(map[string]*FileDescriptor)
	for _, record := range fileRecords {
		fileID, ok := record["_id"].(string)
		if !ok {
			continue
		}

		filename, _ := record["filename"].(string)
		mimeType, _ := record["content_type"].(string)
		size, _ := record["size_bytes"].(int64)
		secure, _ := record["secure"].(bool)
		storageKey, _ := record["storage_key"].(string)
		bucket, _ := record["bucket"].(string)

		// Generate URL (will be implemented in presign.go)
		url, err := s.generateFileURL(ctx, appName, fileID, secure, storageKey, bucket)
		if err != nil {
			slog.Error("failed to generate file URL", "file_id", fileID, "error", err)
			// Continue with empty URL rather than failing the entire operation
			url = ""
		}

		var expiresIn *int
		if secure {
			// Set expiry for secure files (will be configurable in presign.go)
			defaultExpiry := 3600 // 1 hour
			expiresIn = &defaultExpiry
		}

		fileLookup[fileID] = &FileDescriptor{
			ID:        fileID,
			Filename:  filename,
			MimeType:  mimeType,
			Size:      size,
			Secure:    secure,
			URL:       url,
			ExpiresIn: expiresIn,
		}
	}

	// Step 4: Replace UUID values in rows with FileDescriptor objects
	result := make([]map[string]any, len(rows))
	copy(result, rows) // Deep copy to avoid modifying original

	for fileID, refs := range fileIDToRows {
		fileDesc, exists := fileLookup[fileID]
		if !exists {
			slog.Warn("file not found in database", "file_id", fileID)
			// Set to nil for missing files
			fileDesc = nil
		}

		for _, ref := range refs {
			result[ref.rowIndex][ref.colName] = fileDesc
		}
	}

	slog.Info("file resolution completed", "resolved", len(fileLookup))
	return result, nil
}

// fileRef tracks where a file ID appears in the data
type fileRef struct {
	rowIndex int
	colName  string
}

// generateFileURL generates a URL for a file (placeholder - will be implemented in presign.go)
func (s *storageImpl) generateFileURL(ctx context.Context, appName, fileID string, secure bool, storageKey, bucket string) (string, error) {
	// Get app config
	appCfg, exists := s.configs.GetApp(appName)
	if !exists {
		return "", fmt.Errorf("app %q not found", appName)
	}

	if appCfg.Storage == nil {
		return "", ErrStorageDisabled
	}

	// Get S3 client
	client, err := s.clients.getClient(ctx, appName, appCfg.Storage)
	if err != nil {
		return "", fmt.Errorf("failed to get S3 client: %w", err)
	}

	if secure {
		// Generate presigned URL for secure files
		return s.presignURL(ctx, client, appCfg.Storage, storageKey)
	} else {
		// Generate direct URL for non-secure files
		return s.directURL(ctx, client, appCfg.Storage, storageKey)
	}
}
