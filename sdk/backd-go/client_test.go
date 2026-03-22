package backd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient(ClientOptions{
		APIBaseURL:       "http://localhost:8080/v1/myapp",
		AuthBaseURL:      "http://localhost:8080/v1/_auth/mydomain",
		FunctionsBaseURL: "http://localhost:8081/v1/myapp",
		PublishableKey:   "pk_test",
	})

	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.Auth == nil {
		t.Error("Auth client should not be nil")
	}
	if c.Functions == nil {
		t.Error("Functions client should not be nil")
	}
	if c.Jobs == nil {
		t.Error("Jobs client should not be nil")
	}
	if c.Secrets == nil {
		t.Error("Secrets client should not be nil")
	}
}

func TestExtractApp(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"http://localhost:8080/v1/myapp", "myapp"},
		{"http://localhost:8080/v1/hr_portal", "hr_portal"},
		{"myapp", "myapp"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractApp(tt.url)
			if got != tt.want {
				t.Errorf("extractApp(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestClientFrom(t *testing.T) {
	c := NewClient(ClientOptions{
		APIBaseURL:     "http://localhost:8080/v1/myapp",
		PublishableKey: "pk_test",
	})

	qb := c.From("orders")
	if qb == nil {
		t.Fatal("From returned nil")
	}
	if qb.table != "orders" {
		t.Errorf("expected table orders, got %s", qb.table)
	}
}

func TestHTTPHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Publishable-Key") != "pk_test_123" {
			t.Errorf("missing X-Publishable-Key header")
		}
		if r.Header.Get("X-Secret-Key") != "sk_secret" {
			t.Errorf("missing X-Secret-Key header")
		}
		if r.Header.Get("X-Session") != "tok_abc" {
			t.Errorf("missing X-Session header")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "1"}})
	}))
	defer srv.Close()

	c := NewClient(ClientOptions{
		APIBaseURL:     srv.URL + "/v1/myapp",
		AuthBaseURL:    srv.URL + "/v1/_auth/test",
		PublishableKey: "pk_test_123",
		SecretKey:      "sk_secret",
	})
	c.Auth.SetToken("tok_abc")

	ctx := t.Context()
	_, err := c.From("orders").Get(ctx, "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPNoAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Session") != "" {
			t.Error("X-Session should not be set when noAuth is true")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"token": "tok_new"}})
	}))
	defer srv.Close()

	c := NewClient(ClientOptions{
		APIBaseURL:     srv.URL + "/v1/myapp",
		AuthBaseURL:    srv.URL + "/v1/_auth/test",
		PublishableKey: "pk_test",
	})
	c.Auth.SetToken("tok_existing")

	ctx := t.Context()
	err := c.Auth.SignIn(ctx, "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPErrorParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "UNAUTHORIZED", "error_detail": "invalid token"})
	}))
	defer srv.Close()

	c := NewClient(ClientOptions{
		APIBaseURL:     srv.URL + "/v1/myapp",
		PublishableKey: "pk_test",
	})

	ctx := t.Context()
	_, err := c.From("orders").Get(ctx, "1")
	if err == nil {
		t.Fatal("expected error")
	}

	var authErr *AuthError
	if !isAuthError(err, &authErr) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
	if authErr.Code != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED, got %s", authErr.Code)
	}
}

func TestHTTP204NoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(ClientOptions{
		APIBaseURL:     srv.URL + "/v1/myapp",
		PublishableKey: "pk_test",
	})

	ctx := t.Context()
	err := c.From("orders").Delete(ctx, "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// isAuthError is a test helper to check error type (avoid import "errors" in test for simplicity)
func isAuthError(err error, target **AuthError) bool {
	if ae, ok := err.(*AuthError); ok {
		*target = ae
		return true
	}
	return false
}
