package backd

import (
	"context"
	"encoding/json"
	"time"
)

// JobsClient handles background job operations (server-side only).
type JobsClient struct {
	http        *httpClient
	internalURL string
	app         string
}

func newJobsClient(h *httpClient, internalURL, app string) *JobsClient {
	return &JobsClient{http: h, internalURL: internalURL, app: app}
}

// Enqueue submits a background job for processing.
func (j *JobsClient) Enqueue(ctx context.Context, fn string, payload any, opts ...EnqueueOptions) (string, error) {
	body := map[string]any{
		"app":      j.app,
		"function": fn,
		"input":    "{}",
		"trigger":  "sdk",
	}

	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		body["input"] = string(data)
	}

	if len(opts) > 0 {
		o := opts[0]
		if o.Delay != "" {
			ms := parseDuration(o.Delay)
			body["run_at"] = time.Now().Add(ms).UTC().Format(time.RFC3339)
		}
		if o.MaxAttempts > 0 {
			body["max_attempts"] = o.MaxAttempts
		}
	}

	result, err := request[map[string]string](ctx, j.http, requestOptions{
		method:  "POST",
		baseURL: j.internalURL,
		path:    "/internal/jobs",
		body:    body,
		noAuth:  true,
	})
	if err != nil {
		return "", err
	}
	return result["job_id"], nil
}

func parseDuration(s string) time.Duration {
	if len(s) < 2 {
		return 0
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	// Handle "ms" suffix
	if len(s) >= 3 && s[len(s)-2:] == "ms" {
		numStr = s[:len(s)-2]
		unit = 'M' // sentinel
	}

	var n int
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}

	switch unit {
	case 'M': // ms sentinel
		return time.Duration(n) * time.Millisecond
	case 's':
		return time.Duration(n) * time.Second
	case 'm':
		return time.Duration(n) * time.Minute
	case 'h':
		return time.Duration(n) * time.Hour
	case 'd':
		return time.Duration(n) * 24 * time.Hour
	default:
		return 0
	}
}
