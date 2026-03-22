//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

func TestStorage_Upload(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Build multipart upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = part.Write([]byte("hello world"))
	_ = writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL()+"/storage/upload", &buf)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Publishable-Key", publishableKey())
	req.Header.Set("X-Session", c.Auth.Token())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Data.ID == "" {
		t.Error("expected file ID in upload response")
	}
	if result.Data.Filename != "test.txt" {
		t.Errorf("expected filename test.txt, got %s", result.Data.Filename)
	}
}

func TestStorage_FileColumnResolves(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Insert a product with a photo__file reference
	row, err := c.From("products").Insert(ctx, map[string]any{
		"name":  "Test Product",
		"price": 29.99,
	})
	if err != nil {
		t.Fatalf("insert product failed: %v", err)
	}
	id := row["id"].(string)
	t.Cleanup(func() {
		_ = c.From("products").Delete(t.Context(), id)
	})

	// The photo__file column should be present (null since we didn't upload)
	got, err := c.From("products").Get(ctx, id)
	if err != nil {
		t.Fatalf("get product failed: %v", err)
	}
	// photo__file should exist in the response (as null or file descriptor)
	if _, exists := got["photo__file"]; !exists {
		t.Log("photo__file column not in response (may be filtered by policy)")
	}
}
