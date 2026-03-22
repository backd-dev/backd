package backd

import "fmt"

// BackdError is the base error type for all SDK errors.
type BackdError struct {
	Code   string `json:"error"`
	Detail string `json:"error_detail"`
	Status int    `json:"-"`
}

func (e *BackdError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

// AuthError represents authentication errors.
type AuthError struct {
	BackdError
}

func (e *AuthError) Unwrap() error { return &e.BackdError }

// QueryError represents query/CRUD errors.
type QueryError struct {
	BackdError
	Table string
}

func (e *QueryError) Unwrap() error { return &e.BackdError }

// FunctionError represents function invocation errors.
type FunctionError struct {
	BackdError
	Fn string
}

func (e *FunctionError) Unwrap() error { return &e.BackdError }

// NetworkError represents transport-level errors.
type NetworkError struct {
	BackdError
}

func (e *NetworkError) Unwrap() error { return &e.BackdError }

var authCodes = map[string]bool{
	"UNAUTHORIZED":      true,
	"SESSION_EXPIRED":   true,
	"TOO_MANY_REQUESTS": true,
}

var functionCodes = map[string]bool{
	"FUNCTION_NOT_FOUND": true,
	"FUNCTION_TIMEOUT":   true,
	"FUNCTION_ERROR":     true,
}

var storageCodes = map[string]bool{
	"STORAGE_DISABLED": true,
}

// parseError maps an API error response to the appropriate typed error.
func parseError(code, detail string, status int) error {
	base := BackdError{Code: code, Detail: detail, Status: status}

	if authCodes[code] {
		return &AuthError{BackdError: base}
	}
	if functionCodes[code] {
		return &FunctionError{BackdError: base}
	}
	if storageCodes[code] {
		return &BackdError{Code: code, Detail: detail, Status: status}
	}
	if status == 403 || status == 404 || status == 422 {
		return &QueryError{BackdError: base}
	}
	return &base
}
