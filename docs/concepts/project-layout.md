---
title: Project Layout
weight: 2
---

## Repository Structure

```
backd/
├── cmd/
│   ├── backd/              # CLI binary (no DB contact, filesystem only)
│   │   ├── main.go
│   │   └── commands/
│   │       ├── bootstrap.go
│   │       ├── validate.go
│   │       └── templates/   # Embedded scaffold templates
│   └── api/                # Runtime binary (all DB operations)
│       ├── main.go
│       └── commands/
│           ├── start.go     # API server startup
│           ├── functions.go # Functions + worker service
│           ├── migrate.go   # Migration runner
│           └── secrets.go   # Secrets apply
├── internal/
│   ├── config/             # YAML parsing, validation, env substitution
│   ├── celql/              # CEL → SQL transpiler for RLS policies
│   ├── filterql/           # JSON where filter → SQL transpiler
│   ├── db/                 # PostgreSQL pools, schema bootstrap, migrations
│   ├── secrets/            # HKDF key derivation, AES-256-GCM encryption
│   ├── auth/               # Sessions, Argon2id, RLS policy evaluation
│   ├── storage/            # S3 uploads, file resolution
│   ├── functions/          # HTTP proxy client for functions service
│   ├── api/                # HTTP handlers, middleware, CRUD pipeline
│   ├── metrics/            # Prometheus metrics, health server
│   └── deno/               # Deno process pool, job worker, cron
├── sdk/
│   ├── backd-js/           # TypeScript SDK (browser + Deno)
│   └── backd-go/           # Go SDK (used by E2E tests)
├── testdata/
│   └── apps/               # E2E test fixtures
│       ├── test_app/
│       │   ├── app.yaml
│       │   ├── migrations/
│       │   └── functions/
│       └── _domains/
│           └── test_domain/
├── tests/
│   └── e2e/                # End-to-end tests (require docker stack)
├── _docs/                  # Internal design documents
├── docs/                   # Public documentation (this site)
├── Dockerfile.api          # Multi-stage build for API service
├── Dockerfile.functions    # Multi-stage build with Deno runtime
├── docker-compose.yaml     # Development stack
├── Makefile
├── go.mod
└── AGENTS.md               # AI coding guidelines
```

## Two Binaries, Hard Separation

| Binary | Imports | Purpose |
|---|---|---|
| `cmd/backd` | `internal/config`, `internal/celql` only | Filesystem operations — scaffold apps, validate configs. Works without infrastructure. |
| `cmd/api` | All `internal/` packages | Runtime operations — requires PostgreSQL. Serves REST API, functions, worker, migrations, secrets. |

The `backd` CLI never imports database, auth, or secrets packages. It can be used on developer machines without a running database.

## Internal Package Dependency Graph

```
config   → (nothing)
celql    → (nothing)
filterql → (nothing)
db       → config
secrets  → db
auth     → db, celql, config
storage  → db, config
functions → (nothing — thin HTTP client)
api      → auth, db, secrets, storage, functions, metrics, filterql
metrics  → db
deno     → db, auth, secrets
```

Dependencies flow strictly downward — no circular imports. Each package defines an interface in its main file and implements it in separate files.

## Package Responsibilities

### `internal/config`
Parses `app.yaml` and `domain.yaml` into Go structs. Loads `ServerConfig` from environment variables. Performs `${VAR}` substitution. Validates all configs collecting all errors (never short-circuits).

### `internal/celql`
Transpiles CEL (Common Expression Language) policy expressions into PostgreSQL `WHERE` clauses with `$N` parameterized values. Validates that expressions use only the supported CEL subset at startup.

### `internal/filterql`
Transpiles user-supplied JSON `where` filters (e.g. `{"status":{"$eq":"pending"}}`) into SQL `WHERE` clauses. Supports 14 operators including `$eq`, `$gt`, `$in`, `$and`, `$or`, `$not`, `$null`.

### `internal/db`
Manages PostgreSQL connection pools per app, database provisioning (`CREATE DATABASE`), schema bootstrap (reserved tables), migration runner, and generic query execution (`Exec`, `Query`, `QueryOne`). XID generation is centralized here.

### `internal/secrets`
HKDF-SHA256 key derivation (per-app keys from a master key) and AES-256-GCM encryption/decryption. Audit logging for every secret access. No other package imports `crypto/aes` or `hkdf`.

### `internal/auth`
User registration, password hashing (Argon2id), session management, API key validation, RLS policy loading and evaluation, and JSONB metadata mutations. Routes queries to the correct database (app or domain) based on `auth.domain` config.

### `internal/storage`
S3-compatible file uploads (streaming, no full-file buffering), file deletion, and batch `__file` column resolution. Creates per-app S3 clients with path-style addressing for MinIO compatibility.

### `internal/functions`
Thin HTTP proxy client that forwards function invocations from the API service to the functions service. Preserves method, headers, and body.

### `internal/api`
HTTP abstraction layer. Defines `BackdError` types, request/response envelopes, auth middleware, the 8-step CRUD pipeline, and all route registration. `response.go` is the only file that writes to `http.ResponseWriter`.

### `internal/metrics`
Prometheus metric registration (counters, histograms, gauges), request middleware, and the metrics/health HTTP server on a separate port.

### `internal/deno`
Deno process pool management, Unix socket communication protocol, job worker with `FOR UPDATE SKIP LOCKED`, cron scheduler, and the internal HTTP handler for Deno-to-Go communication.

## Testing

| Test type | Command | Location |
|---|---|---|
| **Unit tests** | `make test` | `internal/*_test.go` alongside source |
| **Integration tests** | `go test -tags integration ./internal/db/...` | `internal/db/db_test.go` (requires Postgres) |
| **Go SDK tests** | `make test-sdk` | `sdk/backd-go/*_test.go` |
| **E2E tests** | `make test-e2e` | `tests/e2e/*_test.go` (requires docker stack) |
| **TS SDK typecheck** | `make sdk-check` | `sdk/backd-js/src/` |

E2E tests use the `//go:build e2e` tag and the Go SDK (`sdk/backd-go`) to exercise the full system through the REST API.
