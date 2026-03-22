package backd

// ClientOptions configures a backd client.
type ClientOptions struct {
	// APIBaseURL is the base URL for CRUD operations (e.g. "http://localhost:8080/v1/myapp").
	APIBaseURL string

	// AuthBaseURL is the base URL for auth operations (e.g. "http://localhost:8080/v1/_auth/mydomain").
	AuthBaseURL string

	// FunctionsBaseURL is the base URL for function calls (e.g. "http://localhost:8081/v1/myapp").
	FunctionsBaseURL string

	// PublishableKey is the app's publishable API key.
	PublishableKey string

	// SecretKey is the app's secret key (server-side only).
	SecretKey string

	// InternalURL is the internal Deno handler URL (server-side only, default "http://127.0.0.1:9191").
	InternalURL string
}
