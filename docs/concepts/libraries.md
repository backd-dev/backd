---
title: Libraries
weight: 3
---

## Go Dependencies

backd uses a curated set of Go libraries. Each dependency has a specific role and is documented in `go.mod` with a `// reason:` comment.

### Core Dependencies

| Library | Purpose | Used in |
|---|---|---|
| `github.com/jackc/pgx/v5` | PostgreSQL driver with connection pooling | `internal/db` |
| `github.com/go-chi/chi/v5` | Lightweight HTTP router with middleware support | `internal/api`, `cmd/api` |
| `github.com/google/cel-go` | CEL expression parsing and validation for RLS policies | `internal/celql` |
| `gopkg.in/yaml.v3` | YAML parsing for `app.yaml` and `domain.yaml` | `internal/config` |
| `golang.org/x/crypto` | Argon2id password hashing, HKDF key derivation | `internal/auth`, `internal/secrets` |
| `github.com/rs/xid` | Globally unique, sortable ID generation | `internal/db` (centralized) |
| `github.com/fernandezvara/commandkit` | CLI command, flag, and env var management | `cmd/backd`, `cmd/api` |

### Infrastructure Dependencies

| Library | Purpose | Used in |
|---|---|---|
| `github.com/aws/aws-sdk-go-v2` | S3-compatible file storage (MinIO, AWS, R2) | `internal/storage` |
| `github.com/prometheus/client_golang` | Prometheus metrics collection and exposition | `internal/metrics` |
| `github.com/robfig/cron/v3` | Cron expression parsing and scheduling | `internal/deno` |

### Testing Dependencies

| Library | Purpose |
|---|---|
| `github.com/stretchr/testify` | Assertions and mocking for unit tests |

## Key Patterns

### Centralized XID Generation

All primary keys use XIDs — globally unique, sortable, 20-character strings. The `internal/db/xid.go` file is the **only** place that imports `github.com/rs/xid`. All other code calls `db.NewXID()`.

### Centralized Crypto

- **Argon2id** — only in `internal/auth/argon2.go`
- **HKDF** — only in `internal/secrets/hkdf.go`
- **AES-256-GCM** — only in `internal/secrets/aes.go`

No other file imports these crypto primitives directly.

### Centralized Metrics

`internal/metrics/metrics.go` is the only file that calls `prometheus.NewCounter`, `prometheus.NewHistogram`, etc. No other package registers metrics.

### HTTP Response Writing

`internal/api/response.go` is the only file that writes to `http.ResponseWriter`. Handlers return `(any, error)` and the framework converts to HTTP.
