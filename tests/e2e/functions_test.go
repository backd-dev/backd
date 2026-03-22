//go:build e2e

package e2e

import (
	"errors"
	"testing"

	backd "github.com/backd-dev/backd/sdk/backd-go"
)

func TestFunctions_PublicCall(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	result, err := c.Functions.Call(ctx, "hello", map[string]any{})
	if err != nil {
		t.Fatalf("function call failed: %v", err)
	}
	if result["message"] != "hello" {
		t.Errorf("expected message 'hello', got %v", result["message"])
	}
}

func TestFunctions_PrivateRejected(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	_, err := c.Functions.Call(ctx, "_send_email", map[string]any{})
	if err == nil {
		t.Fatal("expected error calling private function")
	}

	var fnErr *backd.FunctionError
	if !errors.As(err, &fnErr) {
		t.Fatalf("expected FunctionError, got %T: %v", err, err)
	}
	if fnErr.Code != "FUNCTION_NOT_FOUND" {
		t.Errorf("expected FUNCTION_NOT_FOUND, got %s", fnErr.Code)
	}
}

func TestFunctions_WithPayload(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	result, err := c.Functions.Call(ctx, "hello", map[string]any{
		"name": "world",
	})
	if err != nil {
		t.Fatalf("function call with payload failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}
