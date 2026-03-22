package backd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFunctionsClient_RejectPrivate(t *testing.T) {
	h := newHTTPClient("pk", "")
	fc := newFunctionsClient(h, "http://localhost:8081")

	ctx := t.Context()
	_, err := fc.Call(ctx, "_send_email", nil)
	if err == nil {
		t.Fatal("expected error for _-prefixed function")
	}

	var fnErr *FunctionError
	if !errors.As(err, &fnErr) {
		t.Fatalf("expected FunctionError, got %T", err)
	}
	if fnErr.Code != "FUNCTION_NOT_FOUND" {
		t.Errorf("expected FUNCTION_NOT_FOUND, got %s", fnErr.Code)
	}
	if fnErr.Fn != "_send_email" {
		t.Errorf("expected fn _send_email, got %s", fnErr.Fn)
	}
}

func TestFunctionsClient_RejectDoubleUnderscore(t *testing.T) {
	h := newHTTPClient("pk", "")
	fc := newFunctionsClient(h, "http://localhost:8081")

	ctx := t.Context()
	_, err := fc.Call(ctx, "__internal", nil)
	if err == nil {
		t.Fatal("expected error for __-prefixed function")
	}
}

func TestFunctionsClient_AllowPublic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/send_invoice" {
			t.Errorf("expected path /send_invoice, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"message": "ok"}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	fc := newFunctionsClient(h, srv.URL)

	ctx := t.Context()
	result, err := fc.Call(ctx, "send_invoice", map[string]any{"order_id": "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["message"] != "ok" {
		t.Errorf("expected message ok, got %v", result["message"])
	}
}

func TestFunctionsClient_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value" {
			t.Error("missing custom header")
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	fc := newFunctionsClient(h, srv.URL)

	ctx := t.Context()
	_, err := fc.Call(ctx, "hello", nil, InvokeOptions{Headers: map[string]string{"X-Custom": "value"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
