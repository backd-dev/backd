package backd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// httpClient is the internal HTTP helper. It is the sole place that calls http.Client.Do.
type httpClient struct {
	client         *http.Client
	publishableKey string
	secretKey      string
	sessionToken   string
}

func newHTTPClient(publishableKey, secretKey string) *httpClient {
	return &httpClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		publishableKey: publishableKey,
		secretKey:      secretKey,
	}
}

type requestOptions struct {
	method  string
	baseURL string
	path    string
	body    any
	params  map[string]string
	headers map[string]string
	noAuth  bool
}

// request performs an HTTP request and decodes the response into T.
func request[T any](ctx context.Context, h *httpClient, opts requestOptions) (T, error) {
	var zero T

	reqURL, err := buildURL(opts.baseURL, opts.path, opts.params)
	if err != nil {
		return zero, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: err.Error()}}
	}

	var bodyReader io.Reader
	if opts.body != nil {
		data, err := json.Marshal(opts.body)
		if err != nil {
			return zero, fmt.Errorf("backd: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, opts.method, reqURL, bodyReader)
	if err != nil {
		return zero, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: err.Error()}}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Publishable-Key", h.publishableKey)

	if h.secretKey != "" {
		req.Header.Set("X-Secret-Key", h.secretKey)
	}

	if !opts.noAuth && h.sessionToken != "" {
		req.Header.Set("X-Session", h.sessionToken)
	}

	for k, v := range opts.headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		// Retry once on network error — rebuild request so body is fresh
		time.Sleep(500 * time.Millisecond)
		var retryBody io.Reader
		if bodyReader != nil {
			retryBody = bodyReader
			if br, ok := bodyReader.(*bytes.Reader); ok {
				br.Seek(0, io.SeekStart)
				retryBody = br
			}
		}
		req2, err2 := http.NewRequestWithContext(ctx, opts.method, reqURL, retryBody)
		if err2 != nil {
			return zero, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: err.Error()}}
		}
		req2.Header = req.Header
		resp, err = h.client.Do(req2)
		if err != nil {
			return zero, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: err.Error()}}
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return zero, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: "read body: " + err.Error()}}
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error  string `json:"error"`
			Detail string `json:"error_detail"`
		}
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			return zero, parseError(errResp.Error, errResp.Detail, resp.StatusCode)
		}
		return zero, parseError("UNKNOWN_ERROR", resp.Status, resp.StatusCode)
	}

	// Try to parse { "data": T } envelope first
	var envelope struct {
		Data  json.RawMessage `json:"data"`
		Count *int            `json:"count,omitempty"`
	}
	if json.Unmarshal(bodyBytes, &envelope) == nil && len(envelope.Data) > 0 {
		var result T
		if err := json.Unmarshal(envelope.Data, &result); err != nil {
			return zero, fmt.Errorf("backd: unmarshal data: %w", err)
		}
		return result, nil
	}

	// Fall back to direct parse
	var result T
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return zero, fmt.Errorf("backd: unmarshal response: %w", err)
	}
	return result, nil
}

// requestWithMeta performs an HTTP request and returns both data and count.
func requestWithMeta[T any](ctx context.Context, h *httpClient, opts requestOptions) (T, int, error) {
	var zero T

	reqURL, err := buildURL(opts.baseURL, opts.path, opts.params)
	if err != nil {
		return zero, 0, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: err.Error()}}
	}

	req, err := http.NewRequestWithContext(ctx, opts.method, reqURL, nil)
	if err != nil {
		return zero, 0, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: err.Error()}}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Publishable-Key", h.publishableKey)

	if h.secretKey != "" {
		req.Header.Set("X-Secret-Key", h.secretKey)
	}

	if !opts.noAuth && h.sessionToken != "" {
		req.Header.Set("X-Session", h.sessionToken)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		time.Sleep(500 * time.Millisecond)
		resp, err = h.client.Do(req)
		if err != nil {
			return zero, 0, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: err.Error()}}
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, 0, &NetworkError{BackdError: BackdError{Code: "NETWORK_ERROR", Detail: "read body: " + err.Error()}}
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error  string `json:"error"`
			Detail string `json:"error_detail"`
		}
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			return zero, 0, parseError(errResp.Error, errResp.Detail, resp.StatusCode)
		}
		return zero, 0, parseError("UNKNOWN_ERROR", resp.Status, resp.StatusCode)
	}

	var envelope struct {
		Data  json.RawMessage `json:"data"`
		Count int             `json:"count"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return zero, 0, fmt.Errorf("backd: unmarshal response: %w", err)
	}

	var data T
	if len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return zero, 0, fmt.Errorf("backd: unmarshal data: %w", err)
		}
	}
	return data, envelope.Count, nil
}

func buildURL(baseURL, path string, params map[string]string) (string, error) {
	u, err := url.Parse(baseURL + path)
	if err != nil {
		return "", err
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}
