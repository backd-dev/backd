package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/backd-dev/backd/internal/auth"
	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/functions"
	"github.com/backd-dev/backd/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
		Columns:   []string{"*"}, // Allow all columns for testing
		SoftCol:   "",
	}, nil
}

func (m *mockAuth) ApplyDefaults(defaults map[string]string, rc *auth.RequestContext) map[string]any {
	return make(map[string]any)
}

type mockDB struct{}

func (m *mockDB) Provision(ctx context.Context, name string, dbType db.DBType) error {
	return nil
}

func (m *mockDB) Bootstrap(ctx context.Context, name string, dbType db.DBType) error {
	return nil
}

func (m *mockDB) Migrate(ctx context.Context, appName, migrationsPath string) error {
	return nil
}

func (m *mockDB) Pool(name string) (*pgxpool.Pool, error) {
	return nil, nil
}

func (m *mockDB) Exec(ctx context.Context, app, query string, args ...any) error {
	return nil
}

func (m *mockDB) Query(ctx context.Context, app, query string, args ...any) ([]map[string]any, error) {
	return []map[string]any{
		{"id": "test-id", "name": "test-post"},
	}, nil
}

func (m *mockDB) QueryOne(ctx context.Context, app, query string, args ...any) (map[string]any, error) {
	return map[string]any{
		"id":   "test-id",
		"name": "test-post",
	}, nil
}

func (m *mockDB) Tables(ctx context.Context, appName string) ([]db.TableInfo, error) {
	return []db.TableInfo{}, nil
}

func (m *mockDB) Columns(ctx context.Context, appName, table string) ([]db.ColumnInfo, error) {
	return []db.ColumnInfo{}, nil
}

func (m *mockDB) UpsertPublishableKey(ctx context.Context, appName, key string) error {
	return nil
}

func (m *mockDB) VerifyPublishableKey(ctx context.Context, appName, key string) error {
	return nil
}

func (m *mockDB) EnsureSecretKey(ctx context.Context, appName string, s db.Secrets) error {
	return nil
}

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

func (m *mockStorage) Get(ctx context.Context, appName, fileID string) (*storage.FileDescriptor, error) {
	return &storage.FileDescriptor{
		ID:       fileID,
		Filename: "test-file.txt",
		MimeType: "text/plain",
		Size:     1024,
		Secure:   false,
		URL:      "http://example.com/test-file.txt",
	}, nil
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
		DB:              &mockDB{},
		Auth:            &mockAuth{},
		Storage:         &mockStorage{},
		FunctionsClient: &mockFunctions{},
		Config:          &config.ConfigSet{}, // Add mock config
	}
}

// Test stripColumns function
func TestStripColumns(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]any
		allowedColumns []string
		expected       map[string]any
	}{
		{
			name: "filter allowed columns",
			payload: map[string]any{
				"id":     "123",
				"name":   "test",
				"secret": "hidden",
			},
			allowedColumns: []string{"id", "name"},
			expected: map[string]any{
				"id":   "123",
				"name": "test",
			},
		},
		{
			name: "allow all columns with wildcard",
			payload: map[string]any{
				"id":     "123",
				"name":   "test",
				"secret": "hidden",
			},
			allowedColumns: []string{"*"},
			expected: map[string]any{
				"id":     "123",
				"name":   "test",
				"secret": "hidden",
			},
		},
		{
			name:           "empty payload",
			payload:        nil,
			allowedColumns: []string{"id", "name"},
			expected:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripColumns(tt.payload, tt.allowedColumns)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("stripColumns() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test filterResponseColumns function
func TestFilterResponseColumns(t *testing.T) {
	tests := []struct {
		name           string
		result         any
		allowedColumns []string
		expected       any
	}{
		{
			name: "filter single record",
			result: map[string]any{
				"id":     "123",
				"name":   "test",
				"secret": "hidden",
			},
			allowedColumns: []string{"id", "name"},
			expected: map[string]any{
				"id":   "123",
				"name": "test",
			},
		},
		{
			name: "filter paginated response",
			result: map[string]any{
				"data": []map[string]any{
					{"id": "1", "name": "test1", "secret": "hidden1"},
					{"id": "2", "name": "test2", "secret": "hidden2"},
				},
				"count":  2,
				"limit":  10,
				"offset": 0,
			},
			allowedColumns: []string{"id", "name"},
			expected: map[string]any{
				"data": []map[string]any{
					{"id": "1", "name": "test1"},
					{"id": "2", "name": "test2"},
				},
				"count":  2,
				"limit":  10,
				"offset": 0,
			},
		},
		{
			name:           "allow all columns with wildcard",
			result:         []map[string]any{{"id": "1", "name": "test", "secret": "hidden"}},
			allowedColumns: []string{"*"},
			expected:       []map[string]any{{"id": "1", "name": "test", "secret": "hidden"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterResponseColumns(tt.result, tt.allowedColumns)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterResponseColumns() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test CRUD pipeline integration
func TestCRUDPipeline(t *testing.T) {
	deps := createTestDeps()

	// Create a test router with CRUD routes
	r := chi.NewRouter()
	r.Route("/api/v1/{app}/data", func(r chi.Router) {
		RegisterCRUDRoutes(r, deps)
	})

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "create record",
			method:         "POST",
			path:           "/api/v1/testapp/data/posts",
			body:           map[string]any{"title": "Test Post", "content": "Test Content"},
			expectedStatus: 200,
		},
		{
			name:           "list records",
			method:         "GET",
			path:           "/api/v1/testapp/data/posts",
			expectedStatus: 200,
		},
		{
			name:           "get single record",
			method:         "GET",
			path:           "/api/v1/testapp/data/posts/test-id",
			expectedStatus: 200,
		},
		{
			name:           "update record",
			method:         "PUT",
			path:           "/api/v1/testapp/data/posts/test-id",
			body:           map[string]any{"title": "Updated Post"},
			expectedStatus: 200,
		},
		{
			name:           "patch record",
			method:         "PATCH",
			path:           "/api/v1/testapp/data/posts/test-id",
			body:           map[string]any{"content": "Patched Content"},
			expectedStatus: 200,
		},
		{
			name:           "delete record",
			method:         "DELETE",
			path:           "/api/v1/testapp/data/posts/test-id",
			expectedStatus: 204,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			req.Header.Set("X-Session", "test-session")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.expectedStatus == 204 {
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}
			} else {
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}

				var response map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				if response["error"] != nil {
					t.Errorf("Unexpected error in response: %v", response["error"])
				}
			}
		})
	}
}
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

	// Test new auth route structure
	r.Route("/v1/auth/testapp", func(r chi.Router) {
		RegisterDomainAuthRoutes(r, deps)
	})

	t.Run("register user", func(t *testing.T) {
		body := map[string]string{
			"username": "testuser",
			"password": "testpass",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/v1/auth/testapp/local/register", bytes.NewReader(bodyBytes))
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

		req := httptest.NewRequest("POST", "/v1/auth/testapp/local/login", bytes.NewReader(bodyBytes))
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{AppName: "testapp"}))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// Test storage routes
func TestStorageRoutes(t *testing.T) {
	deps := createTestDeps()
	r := chi.NewRouter()

	r.Route("/v1/storage/testapp", func(r chi.Router) {
		RegisterStorageRoutes(r, deps)
	})

	t.Run("upload file", func(t *testing.T) {
		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("test content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/v1/storage/testapp/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{
			AppName:       "testapp",
			Authenticated: true,
		}))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("get file metadata", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/storage/testapp/files/test-file-id", nil)
		req = req.WithContext(WithRequestContext(req.Context(), &RequestContext{
			AppName:       "testapp",
			Authenticated: true,
		}))
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
