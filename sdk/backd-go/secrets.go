package backd

import "context"

// SecretsClient handles secret retrieval (server-side only).
type SecretsClient struct {
	http        *httpClient
	internalURL string
	app         string
}

func newSecretsClient(h *httpClient, internalURL, app string) *SecretsClient {
	return &SecretsClient{http: h, internalURL: internalURL, app: app}
}

// Get retrieves a secret by name. Returns empty string if not found.
func (s *SecretsClient) Get(ctx context.Context, name string) (string, error) {
	result, err := request[map[string]string](ctx, s.http, requestOptions{
		method:  "POST",
		baseURL: s.internalURL,
		path:    "/internal/secret",
		body:    map[string]string{"app": s.app, "name": name},
		noAuth:  true,
	})
	if err != nil {
		return "", err
	}
	return result["secret"], nil
}
