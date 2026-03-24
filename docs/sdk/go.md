---
title: Go SDK
weight: 2
---

`backd-go` is the Go client SDK for backd. It is primarily used for server-side integration and E2E testing.

## Installation

```bash
go get github.com/backd-dev/backd/sdk/backd-go
```

## Client Initialization

```go
import backd "github.com/backd-dev/backd/sdk/backd-go"

client := backd.NewClient(backd.ClientOptions{
    APIBaseURL:       "http://localhost:8080/v1/my_app",
    AuthBaseURL:      "http://localhost:8080/v1/_auth/company",
    FunctionsBaseURL: "http://localhost:8081/v1/my_app",
    PublishableKey:   "vK8mN2pXqR5tL9wA3hE6yJ4cF7uB1nD0",
    // SecretKey:     "...",       // optional, for server-side use
    // InternalURL:   "...",       // optional, defaults to http://127.0.0.1:9191
})
```

## Authentication

```go
ctx := context.Background()

// Register
user, err := client.Auth.SignUp(ctx, "alice", "strong-password-123!")

// Login
err = client.Auth.SignIn(ctx, "alice", "strong-password-123!")

// Get current user
me, err := client.Auth.Me(ctx)

// Check auth state
if client.Auth.IsAuthenticated() {
    fmt.Println("Token:", client.Auth.Token())
}

// Logout
err = client.Auth.SignOut(ctx)
```

## Data Queries

### List with Filtering

```go
data, count, err := client.From("orders").
    Where(backd.WhereFilter{"status": map[string]any{"$eq": "pending"}}).
    Order("created_at", "desc").
    Limit(20).
    Offset(0).
    List(ctx)
```

### Get Single Row

```go
order, err := client.From("orders").Get(ctx, "abc123")
```

### Insert

```go
row, err := client.From("orders").Insert(ctx, map[string]any{
    "total":  99.99,
    "status": "pending",
})
// row["id"] contains the generated ID
```

### Update

```go
updated, err := client.From("orders").Update(ctx, "abc123", map[string]any{
    "status": "confirmed",
})
```

### Patch

```go
patched, err := client.From("orders").Patch(ctx, "abc123", map[string]any{
    "status": "shipped",
})
```

### Delete

```go
err := client.From("orders").Delete(ctx, "abc123")
```

### Select Columns

```go
data, count, err := client.From("orders").
    Select("id", "total", "status").
    List(ctx)
```

## Functions

```go
result, err := client.Functions.Call(ctx, "send_invoice", map[string]any{
    "order_id": "abc123",
})
```

Private functions (`_` prefix) return an error immediately without making a network request.

## Jobs

```go
// Enqueue a background job
jobID, err := client.Jobs.Enqueue(ctx, "_send_email", map[string]any{
    "to": "user@example.com",
})

// With options
jobID, err := client.Jobs.Enqueue(ctx, "_process", payload,
    backd.EnqueueOptions{
        Delay:       "5m",
        MaxAttempts: 5,
    },
)
```

## Error Handling

```go
import "errors"

var authErr *backd.AuthError
var queryErr *backd.QueryError
var fnErr *backd.FunctionError

if errors.As(err, &authErr) {
    fmt.Println("Auth error:", authErr.Code, authErr.Detail)
}

if errors.As(err, &queryErr) {
    fmt.Println("Query error:", queryErr.Code, queryErr.Status)
}

if errors.As(err, &fnErr) {
    fmt.Println("Function error:", fnErr.Code)
}
```

| Error Type | Codes |
|---|---|
| `*AuthError` | `UNAUTHORIZED`, `SESSION_EXPIRED`, `TOO_MANY_REQUESTS` |
| `*QueryError` | `FORBIDDEN`, `NOT_FOUND`, `VALIDATION_ERROR` |
| `*FunctionError` | `FUNCTION_NOT_FOUND`, `FUNCTION_TIMEOUT` |
| `*NetworkError` | `NETWORK_ERROR` |

## Types

```go
type User struct {
    ID        string         `json:"id"`
    Username  string         `json:"username"`
    Meta      map[string]any `json:"meta,omitzero"`
    MetaApp   map[string]any `json:"meta_app,omitzero"`
    CreatedAt string         `json:"created_at,omitzero"`
}

type Session struct {
    Token     string `json:"token"`
    ExpiresAt string `json:"expires_at,omitzero"`
}

type WhereFilter = map[string]any

type FileDescriptor struct {
    ID        string `json:"id"`
    Filename  string `json:"filename"`
    MimeType  string `json:"mime_type"`
    Size      int64  `json:"size"`
    Secure    bool   `json:"secure"`
    URL       string `json:"url"`
    ExpiresIn *int   `json:"expires_in,omitempty"`
}
```

## Use in E2E Tests

The Go SDK is used by the E2E test suite (`tests/e2e/`):

```go
//go:build e2e

func TestCRUD_InsertAndGet(t *testing.T) {
    c := newAuthenticatedClient(t)
    ctx := t.Context()

    row, err := c.From("posts").Insert(ctx, map[string]any{
        "title": "Test Post",
    })
    if err != nil {
        t.Fatalf("insert failed: %v", err)
    }

    got, err := c.From("posts").Get(ctx, row["id"].(string))
    if err != nil {
        t.Fatalf("get failed: %v", err)
    }
    if got["title"] != "Test Post" {
        t.Errorf("expected 'Test Post', got %v", got["title"])
    }
}
```
