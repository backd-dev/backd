package backd

import "context"

// AuthClient handles authentication operations.
type AuthClient struct {
	http    *httpClient
	authURL string
}

func newAuthClient(h *httpClient, authURL string) *AuthClient {
	return &AuthClient{http: h, authURL: authURL}
}

// SignIn authenticates and stores the session token.
func (a *AuthClient) SignIn(ctx context.Context, username, password string) error {
	result, err := request[Session](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/local/login",
		body:    map[string]string{"username": username, "password": password},
		noAuth:  true,
	})
	if err != nil {
		return err
	}
	a.http.sessionToken = result.Token
	return nil
}

// SignOut invalidates the current session.
func (a *AuthClient) SignOut(ctx context.Context) error {
	token := a.http.sessionToken
	if token == "" {
		return nil
	}
	_, err := request[any](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/logout",
		body:    map[string]string{"token": token},
	})
	a.http.sessionToken = ""
	return err
}

// SignUp registers a new user.
func (a *AuthClient) SignUp(ctx context.Context, username, password string) (*User, error) {
	result, err := request[User](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/local/register",
		body:    map[string]string{"username": username, "password": password},
		noAuth:  true,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Me returns the current authenticated user.
func (a *AuthClient) Me(ctx context.Context) (*User, error) {
	result, err := request[User](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/refresh",
		body:    map[string]string{"token": a.http.sessionToken},
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update modifies the current user's profile.
func (a *AuthClient) Update(ctx context.Context, params map[string]string) (*User, error) {
	result, err := request[User](ctx, a.http, requestOptions{
		method:  "PATCH",
		baseURL: a.authURL,
		path:    "/profile",
		body:    params,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Token returns the current session token.
func (a *AuthClient) Token() string {
	return a.http.sessionToken
}

// SetToken manually sets the session token (useful for tests).
func (a *AuthClient) SetToken(token string) {
	a.http.sessionToken = token
}

// IsAuthenticated returns true if a session token is set.
func (a *AuthClient) IsAuthenticated() bool {
	return a.http.sessionToken != ""
}

// SetAppMeta sets an app-scoped metadata key for a user (server-side only).
func (a *AuthClient) SetAppMeta(ctx context.Context, userID, key string, value any) error {
	_, err := request[any](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/internal/auth",
		body:    map[string]any{"action": "set_app_meta", "user_id": userID, "key": key, "value": value},
		noAuth:  true,
	})
	return err
}

// SetGlobalMeta sets a global metadata key for a user (server-side only).
func (a *AuthClient) SetGlobalMeta(ctx context.Context, userID, key string, value any) error {
	_, err := request[any](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/internal/auth",
		body:    map[string]any{"action": "set_global_meta", "user_id": userID, "key": key, "value": value},
		noAuth:  true,
	})
	return err
}

// SetPassword force-sets a password for a user (server-side only).
func (a *AuthClient) SetPassword(ctx context.Context, userID, password string) error {
	_, err := request[any](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/internal/auth",
		body:    map[string]any{"action": "set_password", "user_id": userID, "password": password},
		noAuth:  true,
	})
	return err
}

// SetUsername changes a user's username (server-side only).
func (a *AuthClient) SetUsername(ctx context.Context, userID, username string) error {
	_, err := request[any](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/internal/auth",
		body:    map[string]any{"action": "set_username", "user_id": userID, "username": username},
		noAuth:  true,
	})
	return err
}

// GetUser retrieves a user by ID (server-side only).
func (a *AuthClient) GetUser(ctx context.Context, userID string) (*User, error) {
	result, err := request[User](ctx, a.http, requestOptions{
		method:  "POST",
		baseURL: a.authURL,
		path:    "/internal/auth",
		body:    map[string]any{"action": "get_user", "user_id": userID},
		noAuth:  true,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
