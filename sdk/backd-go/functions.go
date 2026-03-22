package backd

import (
	"context"
	"strings"
)

// FunctionsClient handles function invocations.
type FunctionsClient struct {
	http         *httpClient
	functionsURL string
}

func newFunctionsClient(h *httpClient, functionsURL string) *FunctionsClient {
	return &FunctionsClient{http: h, functionsURL: functionsURL}
}

// Call invokes a named function with an optional payload.
// Returns an error immediately if the function name starts with "_".
func (f *FunctionsClient) Call(ctx context.Context, name string, payload any, opts ...InvokeOptions) (map[string]any, error) {
	if strings.HasPrefix(name, "_") {
		return nil, &FunctionError{
			BackdError: BackdError{Code: "FUNCTION_NOT_FOUND", Detail: "function not found", Status: 404},
			Fn:         name,
		}
	}

	var headers map[string]string
	if len(opts) > 0 && opts[0].Headers != nil {
		headers = opts[0].Headers
	}

	return request[map[string]any](ctx, f.http, requestOptions{
		method:  "POST",
		baseURL: f.functionsURL,
		path:    "/" + name,
		body:    payload,
		headers: headers,
	})
}
