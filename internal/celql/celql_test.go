package celql

import (
	"testing"
)

func TestParse(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name        string
		expression  string
		expectError bool
	}{
		{
			name:        "valid simple expression",
			expression:  "row.user_id == auth.uid",
			expectError: false,
		},
		{
			name:        "valid complex expression",
			expression:  "row.status == 'active' && (auth.meta.role == 'admin' || has(auth.meta.department))",
			expectError: false,
		},
		{
			name:        "valid with functions",
			expression:  "row.created_at > now() - 86400",
			expectError: false,
		},
		{
			name:        "invalid syntax",
			expression:  "row.user_id == && auth.uid",
			expectError: true,
		},
		{
			name:        "invalid method call",
			expression:  "row.title.contains('urgent')",
			expectError: false, // Parse succeeds, validation fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := c.Parse(tt.expression)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but parse succeeded")
				}
				return
			}
			if err != nil {
				t.Errorf("Parse failed: %v", err)
				return
			}
			if ast == nil {
				t.Errorf("Parse returned nil AST")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name        string
		expression  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple comparison",
			expression:  "row.user_id == auth.uid",
			expectError: false,
		},
		{
			name:        "valid null check",
			expression:  "row.deleted_at == null",
			expectError: false,
		},
		{
			name:        "valid with has function",
			expression:  "has(auth.meta.role)",
			expectError: false,
		},
		{
			name:        "valid with now function",
			expression:  "row.expires_at > now()",
			expectError: false,
		},
		{
			name:        "valid with today function",
			expression:  "row.created_at == today()",
			expectError: false,
		},
		{
			name:        "valid logical operators",
			expression:  "row.active == true && auth.authenticated",
			expectError: false,
		},
		{
			name:        "valid in operator",
			expression:  "row.status in ['active', 'pending']",
			expectError: false,
		},
		{
			name:        "valid auth meta access",
			expression:  "auth.meta.role == 'admin'",
			expectError: false,
		},
		{
			name:        "valid auth metaApp access",
			expression:  "auth.metaApp.permission == 'read'",
			expectError: false,
		},
		{
			name:        "invalid method call",
			expression:  "row.title.contains('urgent')",
			expectError: true,
			errorMsg:    "method calls",
		},
		{
			name:        "invalid comprehension",
			expression:  "[x for x in [1, 2, 3]]",
			expectError: true,
			errorMsg:    "comprehensions",
		},
		{
			name:        "invalid struct literal",
			expression:  "User{name: 'test'}",
			expectError: true,
			errorMsg:    "struct",
		},
		{
			name:        "invalid map literal",
			expression:  "{key: 'value'}",
			expectError: true,
			errorMsg:    "map",
		},
		{
			name:        "invalid has on non-auth",
			expression:  "has(row.user_id)",
			expectError: true,
			errorMsg:    "has() can only be used on auth fields",
		},
		{
			name:        "invalid identifier",
			expression:  "unknown_var == true",
			expectError: true,
			errorMsg:    "identifier",
		},
		{
			name:        "invalid arithmetic",
			expression:  "row.count + 1 > 10",
			expectError: true,
			errorMsg:    "unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := c.Parse(tt.expression)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			err = c.Validate(ast)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error but validation succeeded")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate failed: %v", err)
			}
		})
	}
}

func TestTranspile(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	auth := AuthContext{
		UID:           "user123",
		Meta:          map[string]any{"role": "admin", "department": "engineering"},
		MetaApp:       map[string]any{"permission": "read"},
		Authenticated: true,
		KeyType:       "session",
	}

	tests := []struct {
		name           string
		expression     string
		expectedSQL    string
		expectedParams []any
		expectError    bool
	}{
		{
			name:           "simple equality",
			expression:     "row.user_id == auth.uid",
			expectedSQL:    "(user_id = $1)",
			expectedParams: []any{"user123"},
		},
		{
			name:           "null check",
			expression:     "row.deleted_at == null",
			expectedSQL:    "(deleted_at IS NULL)",
			expectedParams: []any{},
		},
		{
			name:           "not null check",
			expression:     "row.deleted_at != null",
			expectedSQL:    "(deleted_at IS NOT NULL)",
			expectedParams: []any{},
		},
		{
			name:           "comparison operators",
			expression:     "row.count > 10",
			expectedSQL:    "(count > $1)",
			expectedParams: []any{int64(10)},
		},
		{
			name:           "has function",
			expression:     "has(auth.meta.role)",
			expectedSQL:    "$1 IS NOT NULL",
			expectedParams: []any{"admin"},
		},
		{
			name:           "now function",
			expression:     "row.expires_at > now()",
			expectedSQL:    "(expires_at > NOW())",
			expectedParams: []any{},
		},
		{
			name:           "today function",
			expression:     "row.created_at == today()",
			expectedSQL:    "(created_at = CURRENT_DATE())",
			expectedParams: []any{},
		},
		{
			name:           "logical and",
			expression:     "row.active == true && auth.authenticated",
			expectedSQL:    "(active = TRUE AND $1)",
			expectedParams: []any{true},
		},
		{
			name:           "logical or",
			expression:     "row.status == 'active' || row.status == 'pending'",
			expectedSQL:    "((status = $1) OR (status = $2))",
			expectedParams: []any{"active", "pending"},
		},
		{
			name:           "logical not",
			expression:     "!row.deleted",
			expectedSQL:    "NOT (deleted)",
			expectedParams: []any{},
		},
		{
			name:           "in operator",
			expression:     "row.status in ['active', 'pending']",
			expectedSQL:    "(status = ANY(ARRAY[$1, $2]))",
			expectedParams: []any{"active", "pending"},
		},
		{
			name:           "auth meta access",
			expression:     "auth.meta.role == 'admin'",
			expectedSQL:    "$1 = $2",
			expectedParams: []any{"admin", "admin"},
		},
		{
			name:           "auth metaApp access",
			expression:     "auth.metaApp.permission == 'read'",
			expectedSQL:    "$1 = $2",
			expectedParams: []any{"read", "read"},
		},
		{
			name:           "complex expression",
			expression:     "row.user_id == auth.uid && row.status in ['active'] && has(auth.meta.role)",
			expectedSQL:    "((user_id = $1) AND ((status = ANY(ARRAY[$2])) AND ($3 IS NOT NULL)))",
			expectedParams: []any{"user123", "active", "admin"},
		},
		{
			name:        "invalid expression",
			expression:  "row.title.contains('urgent')",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := c.Parse(tt.expression)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			result, err := c.Transpile(ast, auth)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected transpilation error but transpilation succeeded")
				}
				return
			}
			if err != nil {
				t.Errorf("Transpile failed: %v", err)
				return
			}

			if result.SQL != tt.expectedSQL {
				t.Errorf("SQL mismatch:\nExpected: %s\nGot:      %s", tt.expectedSQL, result.SQL)
			}

			if !equalParams(result.Params, tt.expectedParams) {
				t.Errorf("Params mismatch:\nExpected: %v\nGot:      %v", tt.expectedParams, result.Params)
			}
		})
	}
}

func TestValidationBeforeTranspile(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	auth := AuthContext{
		UID:           "user123",
		Meta:          map[string]any{},
		MetaApp:       map[string]any{},
		Authenticated: true,
	}

	// Try to transpile an invalid expression
	ast, err := c.Parse("row.title.contains('urgent')")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Transpile should fail with validation error
	_, err = c.Transpile(ast, auth)
	if err == nil {
		t.Errorf("Expected transpilation to fail with validation error")
	}

	if !contains(err.Error(), "validation failed") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func equalParams(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalValue(a[i], b[i]) {
			return false
		}
	}
	return true
}

func equalValue(a, b any) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	default:
		return a == b
	}
}
