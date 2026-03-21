package functions

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client defines the interface for forwarding function invocations
// to the functions service. This is a thin HTTP proxy with no business logic.
type Client interface {
	// Call forwards a function invocation to the functions service.
	// It forwards the original method, all headers, and full request body.
	// The target URL is: <baseURL>/v1/<appName>/<fnName>
	Call(ctx context.Context, appName, fnName string, r *http.Request) (*FunctionResponse, error)
}

// FunctionResponse represents the response from the functions service
type FunctionResponse struct {
	Status  int               // HTTP status code
	Headers map[string]string // Response headers (string values only)
	Body    []byte            // Response body
}

// HTTPClient implements Client interface as a thin HTTP proxy
type HTTPClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPClient creates a new functions client that proxies requests
// to the functions service at the given base URL.
func NewHTTPClient(baseURL string) Client {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Call forwards the request to the functions service.
// It preserves the original HTTP method, all headers, and request body.
func (c *HTTPClient) Call(ctx context.Context, appName, fnName string, r *http.Request) (*FunctionResponse, error) {
	// Construct target URL
	url := fmt.Sprintf("%s/v1/%s/%s", c.baseURL, appName, fnName)

	// Read and restore request body
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Create new request with context
	req, err := http.NewRequestWithContext(ctx, r.Method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("functions.Call: creating request: %w", err)
	}

	// Forward all headers
	for k, v := range r.Header {
		req.Header[k] = v
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("functions.Call: network error: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("functions.Call: reading response: %w", err)
	}

	// Convert headers to string map (only first value per key)
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &FunctionResponse{
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    respBody,
	}, nil
}
