# backd — Test Suite

## Quick Reference

```bash
# Unit tests (no infrastructure needed)
go test ./internal/...

# Go SDK tests (no infrastructure needed)
go test ./sdk/backd-go/...

# TypeScript SDK tests (no infrastructure needed)
cd sdk/backd-js && npm test

# E2E tests (requires Docker Compose stack)
make docker-up
go test ./tests/e2e/... -tags e2e -v
make docker-down
```

---

## Unit Test Inventory

All unit tests use mock dependencies — no database or network required.

| Package | Test file(s) | What is tested | Tests |
|---|---|---|---|
| `internal/config` | `config_test.go` | EnvSubst, name validation, YAML loading, app/domain defaults, storage config, secrets validation, cron entries, policies, CLI formatting | 34 |
| `internal/celql` | `celql_test.go`, `debug_test.go` | CEL parsing, validation (allowed/disallowed nodes), transpile to SQL (comparisons, null, has, now, today, logical, in, auth.meta) | 18 |
| `internal/filterql` | `filterql_test.go` | JSON where → SQL transpile: comparison, set/range, null, text, logical operators, multiple fields, parameter counter, empty filter, error cases | 9 |
| `internal/db` | `db_test.go` | *(no test functions currently)* | 0 |
| `internal/secrets` | `secrets_test.go` | HKDF key derivation, AES encrypt/decrypt, Get/Set/Delete secrets, audit logging, edge cases | 11 |
| `internal/auth` | `auth_test.go` | Argon2id hash/verify, password consistency, LoadPolicies, EvaluatePolicy (normal + secret key bypass), ApplyDefaults, key type constants, policy cache key | 10 |
| `internal/storage` | `storage_test.go` | NewStorage, S3 key generation, isFileColumn detection, error types, File/FileDescriptor structs | 6 |
| `internal/functions` | `functions_test.go` | HTTP client creation, function call (success, network error, non-2xx, context cancellation, header forwarding, empty body, URL construction) | 9 |
| `internal/api` | `api_test.go` | stripColumns, filterResponseColumns, CRUD pipeline (all 6 operations), error constructors, RequestContext, ParseQueryParams, auth routes, CRUD routes, function routes (public + private rejection) | 9 |
| `internal/metrics` | `metrics_test.go` | Prometheus counters (request, function, storage), health/ready handlers, request middleware (with/without app) | 10 |
| `internal/deno` | `cron_test.go`, `deno_test.go`, `invoke_test.go`, `pool_test.go`, `worker_test.go` | Cron parsing/scheduling, pool acquire/release, runner lifecycle, job processing, env vars, permission flags, socket invocation, timeout handling, memory limits, context cancellation, concurrent access | 55+ |

**Total unit tests:** ~170+

---

## SDK Test Inventory

### Go SDK (`sdk/backd-go`)

| Test file | What is tested | Tests |
|---|---|---|
| `errors_test.go` | Error type mapping (auth/function/query codes), error hierarchy, Unwrap support | 6 |
| `client_test.go` | Client creation, app extraction, From(), HTTP headers (publishable key, secret key, session), noAuth, error parsing, 204 handling | 8 |
| `query_test.go` | BuildParams, default params, chain methods, Get, Insert, Update, Patch, Delete, List (with where URL-encoding), Single (not found), InsertMany | 13 |
| `functions_test.go` | `_`-prefix rejection, double underscore rejection, public call, custom headers | 4 |
| `auth_test.go` | SignIn stores token, SignOut clears token, SignOut with no token, SignUp, SetToken | 5 |

**Total Go SDK tests:** 36

### TypeScript SDK (`sdk/backd-js`)

| Test file | What is tested | Tests |
|---|---|---|
| `errors.test.ts` | parseError maps codes to correct subclass, error hierarchy, NetworkError | 15 |
| `storage.test.ts` | memoryStorage: get/set/delete, overwrite, isolation | 5 |
| `functions.test.ts` | `_`-prefix rejection, double prefix, allows non-prefixed | 3 |
| `http.test.ts` | Headers (publishable key, session, secret key, noAuth), URL-encoding, error parsing, 204, retry on network error, NetworkError on double failure | 9 |
| `query.test.ts` | Params from chain, Get, Insert, Update, Patch, Delete, InsertMany, thenable ListResult | 8 |

**Total TypeScript SDK tests:** 40

---

## E2E Test Inventory

All E2E tests are guarded with `//go:build e2e` — they never run during `go test ./...`.

Requires the full Docker Compose stack:
- **API:** `http://localhost:8080`
- **Functions:** `http://localhost:8081`
- **PostgreSQL:** `localhost:5432`
- **MinIO:** `http://localhost:9000`

| Test file | What is tested | Assertions |
|---|---|---|
| `api_test.go` | CRUD insert+get, list, update, patch, delete; `$eq` where filter; pagination (count vs limit); select column subset | 8 tests |
| `auth_test.go` | Register + sign in flow; sign out; invalid credentials → AuthError; me(); update username | 5 tests |
| `functions_test.go` | Public function call; private function → FUNCTION_NOT_FOUND; function with payload | 3 tests |
| `jobs_test.go` | Enqueue + process (poll status); failed job retry | 2 tests |
| `storage_test.go` | Multipart upload → file descriptor; `__file` column in CRUD response | 2 tests |
| `rls_test.go` | Unauthenticated → 403; ownership allows own rows; denies other user; column allowlist; defaults on insert; soft delete | 6 tests |
| `migrate_test.go` | User tables exist (posts, orders, products, items); reserved tables exist (via auth.Me) | 2 tests |

**Total E2E tests:** 28

---

## Test Fixtures (`testdata/`)

| File | Purpose |
|---|---|
| `testdata/apps/test_app/app.yaml` | Test app config: PG, MinIO storage, RLS policies on orders/posts/products |
| `testdata/apps/test_app/migrations/001_init.sql` | Creates `posts`, `orders`, `products` tables |
| `testdata/apps/test_app/migrations/002_add_items.sql` | Creates `items` table |
| `testdata/apps/test_app/functions/hello/index.ts` | Public function returning `{ message: "hello" }` |
| `testdata/apps/test_app/functions/_send_email/index.ts` | Private function for job tests |
| `testdata/apps/_domains/test_domain/domain.yaml` | Test auth domain: password provider, registration enabled |

---

## Coverage Map

| Feature / Service | Unit tests | SDK tests | E2E tests |
|---|---|---|---|
| Config parsing & validation | `internal/config` | — | — |
| CEL → SQL transpilation | `internal/celql` | — | `rls_test.go` |
| JSON where → SQL | `internal/filterql` | — | `api_test.go` |
| Database provisioning | — | — | `migrate_test.go` |
| Secret encryption | `internal/secrets` | — | — |
| Auth (hash, sessions, RLS) | `internal/auth` | `auth_test.go` (both SDKs) | `auth_test.go` |
| Storage (S3, file resolve) | `internal/storage` | — | `storage_test.go` |
| Functions (HTTP proxy) | `internal/functions` | `functions_test.go` (both SDKs) | `functions_test.go` |
| API (CRUD pipeline) | `internal/api` | `query_test.go` (both SDKs) | `api_test.go` |
| Metrics (Prometheus) | `internal/metrics` | — | — |
| Deno (pool, cron, worker) | `internal/deno` | — | `jobs_test.go` |
| RLS policies | `internal/auth` | — | `rls_test.go` |
| Error types & codes | `internal/api` | `errors_test.go` (both SDKs) | all E2E files |
| Go SDK HTTP layer | — | `client_test.go` | all E2E files |
| TS SDK HTTP layer | — | `http.test.ts` | — |
| TS SDK storage adapters | — | `storage.test.ts` | — |

---

## Environment Variables (E2E)

| Variable | Default | Description |
|---|---|---|
| `BACKD_API_URL` | `http://localhost:8080/v1/test_app` | API base URL |
| `BACKD_AUTH_URL` | `http://localhost:8080/v1/_auth/test_domain` | Auth base URL |
| `BACKD_FUNCTIONS_URL` | `http://localhost:8081/v1/test_app` | Functions base URL |
| `BACKD_PUBLISHABLE_KEY` | `pk_test_e2e_key_1234567890abcdef` | Test publishable key |
