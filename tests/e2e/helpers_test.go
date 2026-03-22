//go:build e2e

package e2e

import (
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	backd "github.com/backd-dev/backd/sdk/backd-go"
)

const (
	defaultAPIURL       = "http://localhost:8080/v1/test_app"
	defaultAuthURL      = "http://localhost:8080/v1/_auth/test_domain"
	defaultFunctionsURL = "http://localhost:8081/v1/test_app"
	defaultPublishable  = "pk_test_e2e_key_1234567890abcdef"
)

func apiURL() string {
	if v := os.Getenv("BACKD_API_URL"); v != "" {
		return v
	}
	return defaultAPIURL
}

func authURL() string {
	if v := os.Getenv("BACKD_AUTH_URL"); v != "" {
		return v
	}
	return defaultAuthURL
}

func functionsURL() string {
	if v := os.Getenv("BACKD_FUNCTIONS_URL"); v != "" {
		return v
	}
	return defaultFunctionsURL
}

func publishableKey() string {
	if v := os.Getenv("BACKD_PUBLISHABLE_KEY"); v != "" {
		return v
	}
	return defaultPublishable
}

// newTestClient returns a backd client configured for the E2E test environment.
func newTestClient(t *testing.T) *backd.Client {
	t.Helper()
	return backd.NewClient(backd.ClientOptions{
		APIBaseURL:       apiURL(),
		AuthBaseURL:      authURL(),
		FunctionsBaseURL: functionsURL(),
		PublishableKey:   publishableKey(),
	})
}

// newAuthenticatedClient registers a fresh user, signs in, and returns the client.
func newAuthenticatedClient(t *testing.T) *backd.Client {
	t.Helper()
	c := newTestClient(t)
	ctx := t.Context()

	username := randomUsername()
	password := "test-pass-123!"

	_, err := c.Auth.SignUp(ctx, username, password)
	if err != nil {
		t.Fatalf("failed to register user %s: %v", username, err)
	}

	err = c.Auth.SignIn(ctx, username, password)
	if err != nil {
		t.Fatalf("failed to sign in as %s: %v", username, err)
	}

	return c
}

func randomUsername() string {
	return fmt.Sprintf("e2e_user_%d", rand.IntN(1_000_000))
}
