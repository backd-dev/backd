---
title: Functions Service
weight: 2
---

The functions service executes Deno TypeScript functions. It runs as a separate process from the API service and can be scaled independently.

## Starting

```bash
api functions --config-dir ./apps [--mode both]
```

### Mode Flags

| Mode | Description |
|---|---|
| `both` (default) | HTTP function routes + job worker |
| `functions` | HTTP routes only — no job processing |
| `worker` | Job worker only — no HTTP routes |

This separation allows scaling HTTP function handlers and background workers independently.

## Startup Sequence

1. **Load config and derive keys** (same as API service phases 1–3)
2. **Open DB pools** per app, verify databases exist
3. **Index function files** — scan `functions/` directories for each app
4. **Spawn Deno process pool** (minimum workers per `BACKD_DENO_MIN_WORKERS`)
5. **Register cron schedules** from `app.yaml`
6. **Start internal handler** on `:9191` (localhost only)
7. **Start HTTP server** on `:8081`
8. **Start job worker** goroutine (if mode is `worker` or `both`)

## HTTP Routes

```
POST /v1/<app>/<function_name>     → invoke a public function
GET  /v1/<app>/health              → functions service health check
```

Functions are routed by directory name — no registration or config needed. Renaming a directory immediately changes the route on next startup.

### Private Function Rejection

Functions starting with `_` are rejected at the HTTP layer with `404 FUNCTION_NOT_FOUND` before any Deno process is involved:

```bash
curl -X POST http://localhost:8081/v1/my_app/_send_email
# → 404 {"error":"FUNCTION_NOT_FOUND","error_detail":"function not found"}
```

## Deno Process Pool

The functions service maintains a pool of long-lived Deno processes:

| Config | Env Var | Default | Description |
|---|---|---|---|
| Min workers | `BACKD_DENO_MIN_WORKERS` | 2 | Always-running processes |
| Max workers | `BACKD_DENO_MAX_WORKERS` | 10 | Maximum concurrent processes |
| Idle timeout | `BACKD_DENO_IDLE_TIMEOUT` | 5m | Idle processes above min are terminated |

### Pool Lifecycle

1. On startup, `min` Deno processes are spawned
2. Each process prints `READY\n` to stdout when available
3. `Acquire()` returns an available process or spawns a new one (up to `max`)
4. After execution, `Release()` returns the process to the pool
5. An idle reaper goroutine terminates processes above `min` after the idle timeout
6. Dead processes (socket error) are removed and replaced automatically

## Communication Protocol

Functions communicate with the Go runtime via Unix sockets using JSON-framed messages:

### Request Frame

```json
{
  "id": "xid",
  "app": "my_app",
  "function": "send_invoice",
  "method": "POST",
  "headers": { "Content-Type": "application/json" },
  "body": "{\"order_id\": \"abc123\"}",
  "timeout": 30000
}
```

### Response Frame

```json
{
  "id": "xid",
  "status": 200,
  "headers": { "Content-Type": "application/json" },
  "body": "{\"invoice_id\": \"def456\"}"
}
```

## Embedded Scripts

Two TypeScript files are embedded in the Go binary via `//go:embed`:

- **`runner.ts`** — long-lived Deno host process that listens on a Unix socket
- **`worker_wrapper.ts`** — short-lived Web Worker that loads and executes the user's `index.ts`

These are written to a temp directory at startup — the binary is fully self-contained.

## Environment Variables Injected per Process

Each Deno process receives:

| Variable | Description |
|---|---|
| `BACKD_SOCKET_PATH` | Unix socket path for this process |
| `BACKD_INTERNAL_URL` | Go internal handler URL (`http://127.0.0.1:9191`) |
| `BACKD_APP` | App name |
| `BACKD_SECRET_KEY` | App's secret key (plaintext, from `_secrets`) |
| `BACKD_FUNCTIONS_ROOT` | Root directory for function files |

The `backd-js/deno` SDK reads these automatically — function authors never handle them directly.

## Docker Deployment

The functions service uses a dedicated Dockerfile (`Dockerfile.functions`) that includes both the Go binary and the Deno runtime:

```dockerfile
FROM denoland/deno:alpine-2.x
COPY --from=go-builder /app/api /app/api
ENTRYPOINT ["/app/api"]
CMD ["functions", "--config-dir", "/apps"]
```
