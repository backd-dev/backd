---
title: API Service
weight: 1
---

The API service is the main runtime — it serves the REST CRUD API, authentication endpoints, file storage, and metrics.

## Starting

```bash
api start --config-dir ./apps [--mode both]
```

### Mode Flags

| Mode | Description |
|---|---|
| `both` (default) | Full API + auth service |
| `api` | CRUD and storage only (no auth endpoints) |
| `auth` | Auth endpoints only |

## Startup Sequence

The API follows a strict 7-phase startup:

1. **Load server config** from environment variables (fatal if `BACKD_ENCRYPTION_KEY` missing)
2. **Load and validate** all `app.yaml` / `domain.yaml` (fatal on error)
3. **Derive encryption keys** via HKDF for each app and domain (held in memory)
4. **Provision domains** — create databases, bootstrap tables, generate secret keys
5. **Provision apps** — create databases, bootstrap tables, run migrations, verify API keys, load RLS policies
6. **Start metrics server** on `:9090`
7. **Start HTTP server** on `:8080`

If any app fails during provisioning (e.g. migration error, policy compile error), it is marked as **FAILED** and returns `503 Service Unavailable` for all requests. Other apps continue normally.

## Router Topology

The API service runs three HTTP servers:

| Server | Bind Address | Purpose |
|---|---|---|
| **Public** | `0.0.0.0:8080` | CRUD, auth, storage, functions proxy |
| **Internal** | `0.0.0.0:9191` | Deno → Go communication, jobs endpoint |
| **Metrics** | `0.0.0.0:9090` | `/health`, `/ready`, `/metrics` |

### Public Routes

```
/v1/<app>/<table>              → CRUD endpoints
/v1/<app>/auth/...             → Auth (local apps only)
/v1/_auth/<domain>/...         → Auth (domain-scoped)
/v1/<app>/storage/...          → File upload/delete
/v1/<app>/functions/<name>     → Function proxy → functions service
/health                        → Health check
/ready                         → Readiness check with per-app status
```

### Internal Routes

```
/internal/jobs                 → Job enqueue (from SDK/functions)
/health                        → Internal health check
```

## Middleware Chain

Requests pass through middleware in this order:

1. **RequestID** — generates or reads `X-Request-ID` header
2. **Logger** — structured logging on completion
3. **Recovery** — panic → 500 + stack trace (process doesn't crash)
4. **Metrics** — Prometheus request counters and histograms
5. **Auth** — validates session/key, populates request context

## Health Endpoints

### `/health`

Always returns `200 OK`:

```json
{"status": "ok"}
```

### `/ready`

Returns per-app status:

```json
{
  "status": "ok",
  "apps": {
    "my_app": { "status": "ready" },
    "shop": { "status": "failed", "reason": "migration failed" }
  },
  "domains": {
    "company": { "status": "ready" }
  }
}
```

Top-level `status`: `"ok"` (all ready), `"degraded"` (some failed), `"unavailable"` (all failed).

## Graceful Shutdown

On `SIGINT` or `SIGTERM`, the API server:
1. Stops accepting new connections
2. Waits up to 30 seconds for in-flight requests to complete
3. Closes all database connection pools
4. Exits cleanly

## Prometheus Metrics

Available at `:9090/metrics`:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `backd_requests_total` | Counter | app, method, status | Total HTTP requests |
| `backd_request_duration_seconds` | Histogram | app, method | Request latency |
| `backd_function_invocations_total` | Counter | app, function, status | Function calls |
| `backd_function_duration_seconds` | Histogram | app, function | Function latency |
| `backd_jobs_enqueued_total` | Counter | app, function, trigger | Jobs enqueued |
| `backd_jobs_completed_total` | Counter | app, function, trigger | Jobs completed |
| `backd_jobs_failed_total` | Counter | app, function, trigger | Jobs failed |
| `backd_job_duration_seconds` | Histogram | app, function | Job execution time |
| `backd_db_pool_connections` | Gauge | app | DB pool connections |
| `backd_active_sessions` | Gauge | app | Active user sessions |
