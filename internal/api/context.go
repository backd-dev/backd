package api

import (
	"context"
)

// requestContextKey is the type for context keys to avoid collisions
type requestContextKey string

const (
	contextKeyRequest requestContextKey = "request"
)

// RequestContext holds authentication and request metadata
type RequestContext struct {
	AppName       string            `json:"app_name"`
	UserID        string            `json:"user_id"`
	KeyType       string            `json:"key_type"`       // "publishable" | "secret" | ""
	Authenticated bool              `json:"authenticated"`
	Meta          map[string]any   `json:"meta"`           // metadata._ (global)
	MetaApp       map[string]any   `json:"meta_app"`       // metadata.<app_name>
	RequestID     string            `json:"request_id"`
}

// WithRequestContext stores the RequestContext in the context
func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, contextKeyRequest, rc)
}

// RequestContextFrom retrieves the RequestContext from the context
// Always returns a non-nil RequestContext (Meta and MetaApp are empty maps, never nil)
func RequestContextFrom(ctx context.Context) *RequestContext {
	if rc, ok := ctx.Value(contextKeyRequest).(*RequestContext); ok && rc != nil {
		// Ensure maps are never nil
		if rc.Meta == nil {
			rc.Meta = make(map[string]any)
		}
		if rc.MetaApp == nil {
			rc.MetaApp = make(map[string]any)
		}
		return rc
	}
	
	// Return zero value with non-nil maps
	return &RequestContext{
		Meta:    make(map[string]any),
		MetaApp: make(map[string]any),
	}
}
