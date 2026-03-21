package functions

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	baseURL := "http://localhost:8080"
	client := NewHTTPClient(baseURL)
	
	if hc, ok := client.(*HTTPClient); !ok {
		t.Fatal("NewHTTPClient should return *HTTPClient")
	} else {
		if hc.baseURL != baseURL {
			t.Errorf("Expected baseURL %s, got %s", baseURL, hc.baseURL)
		}
		if hc.client.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", hc.client.Timeout)
		}
	}
}

func TestCall_Success(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL format
		if r.URL.Path != "/v1/testapp/hello" {
			t.Errorf("Expected path /v1/testapp/hello, got %s", r.URL.Path)
		}
		
		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		
		// Verify headers
		if authHeader := r.Header.Get("X-Session"); authHeader != "test-session" {
			t.Errorf("Expected X-Session header, got %s", authHeader)
		}
		
		// Verify body
		body, _ := io.ReadAll(r.Body)
		expected := `{"name":"world"}`
		if string(body) != expected {
			t.Errorf("Expected body %s, got %s", expected, string(body))
		}
		
		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"hello world"}`))
	}))
	defer server.Close()
	
	// Create client
	client := NewHTTPClient(server.URL)
	
	// Create request
	req := httptest.NewRequest("POST", "/functions/hello", strings.NewReader(`{"name":"world"}`))
	req.Header.Set("X-Session", "test-session")
	
	// Call function
	resp, err := client.Call(context.Background(), "testapp", "hello", req)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	
	// Verify response
	if resp.Status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}
	
	if contentType := resp.Headers["Content-Type"]; contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
	
	expectedBody := `{"message":"hello world"}`
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(resp.Body))
	}
}

func TestCall_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewHTTPClient("http://localhost:99999")
	
	req := httptest.NewRequest("GET", "/functions/test", nil)
	
	_, err := client.Call(context.Background(), "testapp", "test", req)
	if err == nil {
		t.Error("Expected network error")
	}
	
	if !strings.Contains(err.Error(), "functions.Call: network error") {
		t.Errorf("Expected wrapped error, got: %v", err)
	}
}

func TestCall_Non2xxResponse(t *testing.T) {
	// Setup mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("function not found"))
	}))
	defer server.Close()
	
	client := NewHTTPClient(server.URL)
	req := httptest.NewRequest("GET", "/functions/missing", nil)
	
	resp, err := client.Call(context.Background(), "testapp", "missing", req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Should pass through non-2xx response as-is
	if resp.Status != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.Status)
	}
	
	expectedBody := "function not found"
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(resp.Body))
	}
}

func TestCall_ContextCancellation(t *testing.T) {
	// Setup slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewHTTPClient(server.URL)
	req := httptest.NewRequest("GET", "/functions/slow", nil)
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := client.Call(ctx, "testapp", "slow", req)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	
	// The error should be wrapped
	if !strings.Contains(err.Error(), "functions.Call: creating request") && 
	   !strings.Contains(err.Error(), "functions.Call: network error") {
		t.Errorf("Expected wrapped error, got: %v", err)
	}
}

func TestCall_HeaderForwarding(t *testing.T) {
	// Setup mock server to verify headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check multiple headers
		headers := map[string]string{
			"X-Session":        r.Header.Get("X-Session"),
			"Content-Type":     r.Header.Get("Content-Type"),
			"X-Custom-Header":  r.Header.Get("X-Custom-Header"),
			"Authorization":    r.Header.Get("Authorization"),
		}
		
		// Return headers in response for verification
		w.Header().Set("X-Received-Session", headers["X-Session"])
		w.Header().Set("X-Received-Custom", headers["X-Custom-Header"])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewHTTPClient(server.URL)
	req := httptest.NewRequest("POST", "/functions/test", strings.NewReader("test body"))
	req.Header.Set("X-Session", "session123")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("Authorization", "Bearer token123")
	
	resp, err := client.Call(context.Background(), "testapp", "test", req)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	
	// Verify headers were forwarded
	if resp.Headers["X-Received-Session"] != "session123" {
		t.Errorf("Expected session header forwarded, got %s", resp.Headers["X-Received-Session"])
	}
	
	if resp.Headers["X-Received-Custom"] != "custom-value" {
		t.Errorf("Expected custom header forwarded, got %s", resp.Headers["X-Received-Custom"])
	}
}

func TestCall_EmptyBody(t *testing.T) {
	// Setup server that handles empty body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body to verify it's empty
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("Expected empty body, got %s", string(body))
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	
	client := NewHTTPClient(server.URL)
	req := httptest.NewRequest("GET", "/functions/test", nil) // No body
	
	resp, err := client.Call(context.Background(), "testapp", "test", req)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	
	if resp.Status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}
}

func TestCall_URLConstruction(t *testing.T) {
	testCases := []struct {
		baseURL   string
		appName   string
		fnName    string
		expected  string
	}{
		{"http://localhost:8080", "myapp", "hello", "http://localhost:8080/v1/myapp/hello"},
		{"https://functions.example.com", "test", "api/user", "https://functions.example.com/v1/test/api/user"},
		{"http://localhost:8080/", "app", "fn", "http://localhost:8080/v1/app/fn"},
		{"http://localhost:8080/api", "myapp", "nested/function", "http://localhost:8080/api/v1/myapp/nested/function"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.baseURL, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/" + tc.appName + "/" + tc.fnName
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			
			client := NewHTTPClient(server.URL)
			req := httptest.NewRequest("GET", "/functions/test", nil)
			
			_, err := client.Call(context.Background(), tc.appName, tc.fnName, req)
			if err != nil {
				t.Fatalf("Call failed: %v", err)
			}
		})
	}
}

func TestFunctionResponse_HeaderHandling(t *testing.T) {
	// Setup server with multiple header values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set multiple values for same header
		w.Header()["Set-Cookie"] = []string{"session=abc", "theme=dark"}
		w.Header()["X-Multi"] = []string{"value1", "value2", "value3"}
		w.Header()["X-Single"] = []string{"single-value"}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewHTTPClient(server.URL)
	req := httptest.NewRequest("GET", "/functions/test", nil)
	
	resp, err := client.Call(context.Background(), "testapp", "test", req)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	
	// Verify only first value is preserved (as per implementation)
	if resp.Headers["Set-Cookie"] != "session=abc" {
		t.Errorf("Expected first cookie value, got %s", resp.Headers["Set-Cookie"])
	}
	
	if resp.Headers["X-Multi"] != "value1" {
		t.Errorf("Expected first multi value, got %s", resp.Headers["X-Multi"])
	}
	
	if resp.Headers["X-Single"] != "single-value" {
		t.Errorf("Expected single value, got %s", resp.Headers["X-Single"])
	}
}
