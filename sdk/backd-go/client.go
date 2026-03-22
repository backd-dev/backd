package backd

// Client is the main entry point for the backd Go SDK.
type Client struct {
	Auth      *AuthClient
	Functions *FunctionsClient
	Jobs      *JobsClient
	Secrets   *SecretsClient

	http   *httpClient
	apiURL string
}

// NewClient creates a new backd client with the given options.
func NewClient(opts ClientOptions) *Client {
	h := newHTTPClient(opts.PublishableKey, opts.SecretKey)

	internalURL := opts.InternalURL
	if internalURL == "" {
		internalURL = "http://127.0.0.1:9191"
	}

	// Extract app name from API URL for internal calls
	app := extractApp(opts.APIBaseURL)

	c := &Client{
		http:   h,
		apiURL: opts.APIBaseURL,
	}

	c.Auth = newAuthClient(h, opts.AuthBaseURL)
	c.Functions = newFunctionsClient(h, opts.FunctionsBaseURL)
	c.Jobs = newJobsClient(h, internalURL, app)
	c.Secrets = newSecretsClient(h, internalURL, app)

	return c
}

// From starts a query builder for the given table/collection.
func (c *Client) From(table string) *QueryBuilder {
	return newQueryBuilder(c.http, c.apiURL, table)
}

// extractApp extracts the app name from a URL like "http://host:port/v1/myapp".
func extractApp(apiURL string) string {
	// Simple extraction: take the last path segment
	for i := len(apiURL) - 1; i >= 0; i-- {
		if apiURL[i] == '/' {
			return apiURL[i+1:]
		}
	}
	return apiURL
}
