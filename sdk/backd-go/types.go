package backd

// XID is a 20-character unique identifier string.
type XID = string

// User represents a backd user.
type User struct {
	ID        XID                    `json:"id"`
	Username  string                 `json:"username"`
	Meta      map[string]any         `json:"meta,omitzero"`
	MetaApp   map[string]any         `json:"meta_app,omitzero"`
	CreatedAt string                 `json:"created_at,omitzero"`
}

// Session represents an authenticated session.
type Session struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at,omitzero"`
}

// ListResult holds a paginated list response.
type ListResult[T any] struct {
	Data  []T `json:"data"`
	Count int `json:"count"`
}

// FileDescriptor describes a resolved file reference.
type FileDescriptor struct {
	ID        XID    `json:"id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	Secure    bool   `json:"secure"`
	URL       string `json:"url"`
	ExpiresIn *int   `json:"expires_in,omitempty"`
}

// WhereFilter is a JSON-serializable filter object for queries.
// Example: map[string]any{"status": map[string]any{"$eq": "pending"}}
type WhereFilter = map[string]any

// EnqueueOptions configures job enqueue behavior.
type EnqueueOptions struct {
	Delay       string `json:"delay,omitzero"`
	MaxAttempts int    `json:"max_attempts,omitzero"`
}

// InvokeOptions configures function invocation.
type InvokeOptions struct {
	Headers map[string]string
}
