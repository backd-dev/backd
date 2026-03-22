package backd

import (
	"errors"
	"testing"
)

func TestParseError_AuthCodes(t *testing.T) {
	for _, code := range []string{"UNAUTHORIZED", "SESSION_EXPIRED", "TOO_MANY_REQUESTS"} {
		t.Run(code, func(t *testing.T) {
			err := parseError(code, "test", 401)
			var authErr *AuthError
			if !errors.As(err, &authErr) {
				t.Errorf("expected AuthError for code %s, got %T", code, err)
			}
			if authErr.Code != code {
				t.Errorf("expected code %s, got %s", code, authErr.Code)
			}
		})
	}
}

func TestParseError_FunctionCodes(t *testing.T) {
	for _, code := range []string{"FUNCTION_NOT_FOUND", "FUNCTION_TIMEOUT", "FUNCTION_ERROR"} {
		t.Run(code, func(t *testing.T) {
			err := parseError(code, "test", 500)
			var fnErr *FunctionError
			if !errors.As(err, &fnErr) {
				t.Errorf("expected FunctionError for code %s, got %T", code, err)
			}
		})
	}
}

func TestParseError_QueryCodes(t *testing.T) {
	tests := []struct {
		code   string
		status int
	}{
		{"FORBIDDEN", 403},
		{"NOT_FOUND", 404},
		{"VALIDATION_ERROR", 422},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := parseError(tt.code, "test", tt.status)
			var qErr *QueryError
			if !errors.As(err, &qErr) {
				t.Errorf("expected QueryError for status %d, got %T", tt.status, err)
			}
		})
	}
}

func TestParseError_UnknownCode(t *testing.T) {
	err := parseError("INTERNAL_ERROR", "something broke", 500)
	var backdErr *BackdError
	if !errors.As(err, &backdErr) {
		t.Fatalf("expected BackdError, got %T", err)
	}
	var authErr *AuthError
	if errors.As(err, &authErr) {
		t.Error("should not be AuthError")
	}
}

func TestBackdError_Error(t *testing.T) {
	err := &BackdError{Code: "TEST", Detail: "detail", Status: 400}
	want := "TEST: detail"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestErrorHierarchy(t *testing.T) {
	authErr := &AuthError{BackdError: BackdError{Code: "UNAUTHORIZED", Detail: "test", Status: 401}}
	var backdErr *BackdError
	if !errors.As(authErr, &backdErr) {
		t.Error("AuthError should be unwrappable to BackdError")
	}

	queryErr := &QueryError{BackdError: BackdError{Code: "NOT_FOUND", Detail: "missing", Status: 404}, Table: "orders"}
	if queryErr.Table != "orders" {
		t.Errorf("expected table orders, got %s", queryErr.Table)
	}

	fnErr := &FunctionError{BackdError: BackdError{Code: "FUNCTION_ERROR", Detail: "fail", Status: 500}, Fn: "send"}
	if fnErr.Fn != "send" {
		t.Errorf("expected fn send, got %s", fnErr.Fn)
	}

	netErr := &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: "timeout", Status: 0}}
	if netErr.Status != 0 {
		t.Errorf("expected status 0, got %d", netErr.Status)
	}
}
