package api

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterStorageRoutes registers file storage routes using resource-based routing
// Routes are: /upload, /files/{fileId}, /{fileId} (nested under /v1/storage/{app})
func RegisterStorageRoutes(r chi.Router, deps *Deps) {
	// Storage routes: directly under /v1/storage/{app}
	// Upload file - POST /upload
	r.Post("/upload", Handler(handleFileUpload(deps)).Handle(deps))

	// Get file metadata + access URL - GET /files/{fileId}
	r.Get("/files/{fileId}", Handler(handleFileGet(deps)).Handle(deps))

	// Delete file - DELETE /{fileId}
	r.Delete("/{fileId}", Handler(handleFileDelete(deps)).Handle(deps))
}

// File upload handler
func handleFileUpload(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// Check if storage is configured for this app
		if deps.Storage == nil {
			return nil, ErrStorageDisabled("Storage not configured for this app")
		}

		// Parse multipart form (max 32MB)
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return nil, ErrBadRequest("INVALID_FORM", "Failed to parse multipart form")
		}

		// Get file from form
		file, header, err := r.FormFile("file")
		if err != nil {
			return nil, ErrBadRequest("MISSING_FILE", "No file provided")
		}
		defer file.Close()

		// Buffer into bytes.Reader so S3 SDK can seek for payload hashing
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, ErrInternal("Failed to read uploaded file")
		}
		body := bytes.NewReader(data)

		// Upload file using storage package
		uploadedFile, err := deps.Storage.Upload(r.Context(), rc.AppName, header.Filename, false, body)
		if err != nil {
			slog.Error("file upload failed", "app", rc.AppName, "filename", header.Filename, "error", err)
			return nil, ErrInternal("Failed to upload file: " + err.Error())
		}

		// Return file descriptor
		return map[string]any{
			"id":       uploadedFile.ID,
			"filename": uploadedFile.Filename,
			"size":     uploadedFile.Size,
			"secure":   uploadedFile.Secure,
		}, nil
	}
}

// File delete handler
func handleFileDelete(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// Check if storage is configured for this app
		if deps.Storage == nil {
			return nil, ErrStorageDisabled("Storage not configured for this app")
		}

		// Get file ID from URL
		fileId := chi.URLParam(r, "fileId")
		if fileId == "" {
			return nil, ErrBadRequest("MISSING_FILE_ID", "File ID is required")
		}

		// Delete file using storage package
		err := deps.Storage.Delete(r.Context(), rc.AppName, fileId)
		if err != nil {
			return nil, ErrNotFound("File not found")
		}

		return nil, nil // No content on successful delete
	}
}

// File get handler
func handleFileGet(deps *Deps) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// Check if storage is configured for this app
		if deps.Storage == nil {
			return nil, ErrStorageDisabled("Storage not configured for this app")
		}

		// Get file ID from URL
		fileId := chi.URLParam(r, "fileId")
		if fileId == "" {
			return nil, ErrBadRequest("MISSING_FILE_ID", "File ID is required")
		}

		// Get file metadata and URL
		fileDescriptor, err := deps.Storage.Get(r.Context(), rc.AppName, fileId)
		if err != nil {
			slog.Error("file get failed", "app", rc.AppName, "fileId", fileId, "error", err)
			return nil, ErrNotFound("File not found")
		}

		return fileDescriptor, nil
	}
}
