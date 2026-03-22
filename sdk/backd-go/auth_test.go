package backd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthClient_SignIn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/local/login" {
			t.Errorf("expected path /local/login, got %s", r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["username"] != "alice" || body["password"] != "secret" {
			t.Error("unexpected credentials")
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"token": "tok_abc"}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	auth := newAuthClient(h, srv.URL)

	ctx := t.Context()
	err := auth.SignIn(ctx, "alice", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.Token() != "tok_abc" {
		t.Errorf("expected token tok_abc, got %s", auth.Token())
	}
	if !auth.IsAuthenticated() {
		t.Error("should be authenticated after sign in")
	}
}

func TestAuthClient_SignOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/logout" {
			t.Errorf("expected path /logout, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	h.sessionToken = "tok_abc"
	auth := newAuthClient(h, srv.URL)

	ctx := t.Context()
	err := auth.SignOut(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.Token() != "" {
		t.Error("token should be cleared after sign out")
	}
	if auth.IsAuthenticated() {
		t.Error("should not be authenticated after sign out")
	}
}

func TestAuthClient_SignOutNoToken(t *testing.T) {
	h := newHTTPClient("pk", "")
	auth := newAuthClient(h, "http://localhost")

	ctx := t.Context()
	err := auth.SignOut(ctx)
	if err != nil {
		t.Fatalf("sign out with no token should not error: %v", err)
	}
}

func TestAuthClient_SignUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/local/register" {
			t.Errorf("expected path /local/register, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"id": "user1", "username": "bob",
		}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	auth := newAuthClient(h, srv.URL)

	ctx := t.Context()
	user, err := auth.SignUp(ctx, "bob", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user1" {
		t.Errorf("expected id user1, got %s", user.ID)
	}
	if user.Username != "bob" {
		t.Errorf("expected username bob, got %s", user.Username)
	}
}

func TestAuthClient_SetToken(t *testing.T) {
	h := newHTTPClient("pk", "")
	auth := newAuthClient(h, "http://localhost")

	auth.SetToken("manual_token")
	if auth.Token() != "manual_token" {
		t.Errorf("expected manual_token, got %s", auth.Token())
	}
}
