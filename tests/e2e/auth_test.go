//go:build e2e

package e2e

import (
	"errors"
	"testing"

	backd "github.com/backd-dev/backd/sdk/backd-go"
)

func TestAuth_RegisterAndSignIn(t *testing.T) {
	c := newTestClient(t)
	ctx := t.Context()

	username := randomUsername()
	password := "strong-pass-123!"

	user, err := c.Auth.SignUp(ctx, username, password)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.ID == "" {
		t.Error("expected user ID")
	}
	if user.Username != username {
		t.Errorf("expected username %s, got %s", username, user.Username)
	}

	err = c.Auth.SignIn(ctx, username, password)
	if err != nil {
		t.Fatalf("sign in failed: %v", err)
	}
	if !c.Auth.IsAuthenticated() {
		t.Error("should be authenticated after sign in")
	}
	if c.Auth.Token() == "" {
		t.Error("token should not be empty after sign in")
	}
}

func TestAuth_SignOut(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	if !c.Auth.IsAuthenticated() {
		t.Fatal("should be authenticated before sign out")
	}

	err := c.Auth.SignOut(ctx)
	if err != nil {
		t.Fatalf("sign out failed: %v", err)
	}
	if c.Auth.IsAuthenticated() {
		t.Error("should not be authenticated after sign out")
	}
}

func TestAuth_InvalidCredentials(t *testing.T) {
	c := newTestClient(t)
	ctx := t.Context()

	err := c.Auth.SignIn(ctx, "nonexistent_user", "wrong_password")
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}

	var authErr *backd.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
}

func TestAuth_Me(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	user, err := c.Auth.Me(ctx)
	if err != nil {
		t.Fatalf("me failed: %v", err)
	}
	if user.ID == "" {
		t.Error("expected user ID from me()")
	}
	if user.Username == "" {
		t.Error("expected username from me()")
	}
}

func TestAuth_UpdateUsername(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	newUsername := randomUsername()
	user, err := c.Auth.Update(ctx, map[string]string{"username": newUsername})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if user.Username != newUsername {
		t.Errorf("expected username %s, got %s", newUsername, user.Username)
	}
}
