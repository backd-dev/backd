package api

import (
	"fmt"
	"net/http"
)

// BackdError represents a structured API error with code and detail
type BackdError struct {
	Code       string `json:"error"`
	Detail     string `json:"error_detail"`
	StatusCode int    `json:"-"`
}

func (e *BackdError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

// Error constructors - these are the ONLY way to create BackdError instances
// All other files must use these constructors, never create BackdError directly

func ErrUnauthorized(detail string) *BackdError {
	return &BackdError{
		Code:       "UNAUTHORIZED",
		Detail:     detail,
		StatusCode: http.StatusUnauthorized,
	}
}

func ErrSessionExpired(detail string) *BackdError {
	return &BackdError{
		Code:       "SESSION_EXPIRED",
		Detail:     detail,
		StatusCode: http.StatusUnauthorized,
	}
}

func ErrForbidden(detail string) *BackdError {
	return &BackdError{
		Code:       "FORBIDDEN",
		Detail:     detail,
		StatusCode: http.StatusForbidden,
	}
}

func ErrNotFound(detail string) *BackdError {
	return &BackdError{
		Code:       "NOT_FOUND",
		Detail:     detail,
		StatusCode: http.StatusNotFound,
	}
}

func ErrBadRequest(code, detail string) *BackdError {
	return &BackdError{
		Code:       code,
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
	}
}

func ErrValidation(detail string) *BackdError {
	return &BackdError{
		Code:       "VALIDATION_ERROR",
		Detail:     detail,
		StatusCode: http.StatusUnprocessableEntity,
	}
}

func ErrInternal(detail string) *BackdError {
	return &BackdError{
		Code:       "INTERNAL_ERROR",
		Detail:     detail,
		StatusCode: http.StatusInternalServerError,
	}
}

func ErrServiceUnavailable(detail string) *BackdError {
	return &BackdError{
		Code:       "SERVICE_UNAVAILABLE",
		Detail:     detail,
		StatusCode: http.StatusServiceUnavailable,
	}
}

func ErrStorageDisabled(detail string) *BackdError {
	return &BackdError{
		Code:       "STORAGE_DISABLED",
		Detail:     detail,
		StatusCode: http.StatusNotImplemented,
	}
}

func ErrFunctionNotFound(detail string) *BackdError {
	return &BackdError{
		Code:       "FUNCTION_NOT_FOUND",
		Detail:     detail,
		StatusCode: http.StatusNotFound,
	}
}

func ErrFunctionTimeout(detail string) *BackdError {
	return &BackdError{
		Code:       "FUNCTION_TIMEOUT",
		Detail:     detail,
		StatusCode: http.StatusGatewayTimeout,
	}
}
