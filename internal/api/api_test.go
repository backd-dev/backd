package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/backd-dev/backd/internal/auth"
	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/functions"
	"github.com/backd-dev/backd/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Mock implementations for testing

type mockAuth struct{}

func (m *mockAuth) SignIn(ctx context.Context, appName, domainName, username, password string) (*auth.Session, error) {
	return &auth.Session{ID: "session-123", UserID: "user-123"}, nil
}

func (m *mockAuth) SignOut(ctx context.Context, token string) error {
	return nil
}

func (m *mockAuth) ValidateSession(ctx context.Context, token string) (*auth.RequestContext, error) {
	return &auth.RequestContext{
		UID:           "user-123",
		Meta:          make(map[string]any),
		MetaApp:       make(map[string]any),
		Authenticated: true,
		KeyType:       "",
	}, nil
}

func (m *mockAuth) ValidateKey(ctx context.Context, appName, key string) (auth.KeyType, error) {
	return auth.KeyTypePublishable, nil
}

func (m *mockAuth) UpsertPublishableKey(ctx context.Context, appName, key string) error {
	return nil
}

func (m *mockAuth) VerifyPublishableKey(ctx context.Context, appName, key string) error {
	return nil
}

func (m *mockAuth) Register(ctx context.Context, appName, username, password string) (*auth.User, error) {
	return &auth.User{ID: "user-123", Username: username}, nil
}

func (m *mockAuth) UpdateUsername(ctx context.Context, appName, userID, username string) error {
	return nil
}

func (m *mockAuth) UpdatePassword(ctx context.Context, appName, userID, password string) error {
	return nil
}

func (m *mockAuth) GetUser(ctx context.Context, appName, userID string) (*auth.User, error) {
	return &auth.User{ID: userID, Username: "testuser"}, nil
}

func (m *mockAuth) SetGlobalMeta(ctx context.Context, appName, userID, key string, value any) error {
	return nil
}

func (m *mockAuth) SetAppMeta(ctx context.Context, appName, userID, key string, value any) error {
	return nil
}

func (m *mockAuth) LoadPolicies(ctx context.Context, appName string, cfg *config.AppConfig) error {
	return nil
}

func (m *mockAuth) EvaluatePolicy(ctx context.Context, appName, table, operation string, rc *auth.RequestContext) (auth.PolicyResult, error) {
	return auth.PolicyResult{
		SQLClause: "TRUE",
		Params:    []any{},
		Defaults:  make(map[string]string),
		SoftCol:   "",
	}, nil
}

type mockDB struct{}

type mockFunctions struct{}

func (m *mockFunctions) Call(ctx context.Context, appName, fnName string, r *http.Request) (*functions.FunctionResponse, error) {
	return &functions.FunctionResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"message": "Hello from function"}`),
	}, nil
}

type mockStorage struct{}

func (m *mockStorage) Upload(ctx context.Context, appName, filename string, secure bool, body io.Reader) (*storage.File, error) {
	return &storage.File{
		ID:       "file-123",
		Filename: filename,
		Size:     1024,
		Secure:   secure,
	}, nil
}

func (m *mockStorage) Delete(ctx context.Context, appName, fileID string) error {
	return nil
}

func (m *mockStorage) ResolveFiles(ctx context.Context, appName string, rows []map[string]any) ([]map[string]any, error) {
	return rows, nil
}

type mockMetrics struct{}

func (m *mockMetrics) RecordRequest(method, path, status string)                  {}
func (m *mockMetrics) RecordFunctionCall(app, fn string, success bool)            {}
func (m *mockMetrics) RecordStorageOperation(app, operation string, success bool) {}

type mockSecrets struct{}

// Helper function to create test dependencies
func createTestDeps() *Deps {
	return &Deps{
		Auth:            &mockAuth{},
		Storage:         &mockStorage{},
		FunctionsClient: &mockFunctions{},
	}
}

// Test error constructors
func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name        string
		constructor func(string) *BackdError
		code        string
		statusCode  int
	}{
		{"ErrUnauthorized", ErrUnauthorized, "UNAUTHORIZED", http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, "FORBIDDEN", http.StatusForbidden},
		{"ErrNotFound", ErrNotFound, "NOT_FOUND", http.StatusNotFound},
		{"ErrInternal", ErrInternal, "INTERNAL_ERROR", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor("test detail")
			if err.Code != tt.code {
				t.Errorf("Expected code %s, got %s", tt.code, err.Code)
			}
			if err.StatusCode != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, err.StatusCode)
			}
		})
	}
}

// Test request context
func TestRequestContext(t *testing.T) {
	rc := &RequestContext{
		AppName:       "test-app",
		UserID:        "user-123",
		Authenticated: true,
		Meta:          make(map[string]any),
		MetaApp:       make(map[string]any),
		RequestID:     "req-123",
	}

	// Test context storage
	ctx := WithRequestContext(context.Background(), rc)
	retrieved := RequestContextFrom(ctx)

	if retrieved.AppName != rc.AppName {
		t.Errorf("Expected app name %s, got %s", rc.AppName, retrieved.AppName)
	}

	// Test that RequestContextFrom never returns nil
	emptyCtx := context.Background()
	emptyRC := RequestContextFrom(emptyCtx)
	if emptyRC == nil {
		t.Error("RequestContextFrom should never return nil")
	}
	if emptyRC.Meta == nil || emptyRC.MetaApp == nil {
		t.Error("Meta and MetaApp should never be nil")
	}
}

// Test query parameter parsing
func TestParseQueryParams(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *QueryParams
		wantErr bool
	}{
		{
			name: "default values",
			url:  "/test",
			want: &QueryParams{
				Limit:  defaultLimit,
				Offset: defaultOffset,
			},
			wantErr: false,
		},
		{
			name: "custom limit and offset",
			url:  "/test?limit=100&offset=10",
			want: &QueryParams{
				Limit:  100,
				Offset: 10,
			},
			wantErr: false,
		},
		{
			name:    "invalid limit",
			url:     "/test?limit=invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "limit too high",
			url:     "/test?limit=2000",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			got, err := ParseQueryParams(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQueryParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Limit != tt.want.Limit {
					t.Errorf("Expected limit %d, got %d", tt.want.Limit, got.Limit)
				}
				if got.Offset != tt.want.Offset {
					t.Errorf("Expected offset %d, got %d", tt.want.Offset, got.Offset)
				}
			}
		})
	}
}

// Test auth routes
func TestAuthRoutes(t *testing.T) {
	deps := createTestDeps()
	r := chi.NewRouter()
	RegisterAuthRoutes(r, deps)

	t.Run("register user", func(t *testing.T) {
		body := map[string]string{
			"username": "testuser",
			"password": "testpass",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/local/register", bytes.NewReader(bodyBytes))
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{AppName: "testapp"}))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}
	})

	t.Run("sign in", func(t *testing.T) {
		body := map[string]string{
			"username": "testuser",
			"password": "testpass",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/local/login", bytes.NewReader(bodyBytes))
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{AppName: "testapp"}))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// Test CRUD routes
func TestCRUDRoutes(t *testing.T) {
	deps := createTestDeps()
	r := chi.NewRouter()
	RegisterCRUDRoutes(r, deps)

	t.Run("list collection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts", nil)
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{
			AppName:       "testapp",
			UserID:        "user-123",
			Authenticated: true,
		}))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// Test function routes
func TestFunctionRoutes(t *testing.T) {
	deps := createTestDeps()
	r := chi.NewRouter()
	RegisterFunctionRoutes(r, deps)

	t.Run("call public function", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/functions/hello", bytes.NewReader([]byte(`{}`)))
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{
			AppName:       "testapp",
			UserID:        "user-123",
			Authenticated: true,
		}))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("reject private function", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/functions/_private", bytes.NewReader([]byte(`{}`)))
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{
			AppName:       "testapp",
			UserID:        "user-123",
			Authenticated: true,
		}))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for private function, got %d", w.Code)
		}
	})
}
