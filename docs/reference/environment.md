---
title: Environment Variables
weight: 4
---

Server configuration for `cmd/api`. Passed via environment variables — never in a file, never committed to version control.

## Required

| Variable | Description |
|---|---|
| `BACKD_ENCRYPTION_KEY` | Base64url-encoded 32-byte master key. Derives per-app encryption keys via HKDF. Generate with: `openssl rand -base64 32` |

## Database

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — | Default PostgreSQL DSN. Used when an app has no `database` block in `app.yaml`. Format: `postgres://user:pass@host:5432/backd_platform` |

When `DATABASE_URL` is used as fallback, backd automatically replaces the database name with `backd_<app>` or `backd_domain_<domain>` for each app/domain.

## Ports

| Variable | Default | Description |
|---|---|---|
| `BACKD_API_PORT` | `8080` | Public API HTTP server port |
| `BACKD_FUNCTIONS_PORT` | `8081` | Functions HTTP server port |
| `BACKD_METRICS_PORT` | `9090` | Prometheus metrics + health server port |
| `BACKD_INTERNAL_PORT` | `9191` | Internal handler port (Deno ↔ Go communication) |

## Service URLs

| Variable | Default | Description |
|---|---|---|
| `BACKD_FUNCTIONS_URL` | `http://functions:8081` | Base URL of the functions service. Used by the API service to proxy public function calls. |

## Logging

| Variable | Default | Description |
|---|---|---|
| `BACKD_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `BACKD_LOG_FORMAT` | `json` | Log format: `json`, `text` |

## Deno Pool

| Variable | Default | Description |
|---|---|---|
| `BACKD_DENO_MIN_WORKERS` | `2` | Minimum Deno processes per app in the pool |
| `BACKD_DENO_MAX_WORKERS` | `10` | Maximum Deno processes per app |
| `BACKD_DENO_IDLE_TIMEOUT` | `5m` | Duration before idle processes above min are terminated |

## Job Worker

| Variable | Default | Description |
|---|---|---|
| `BACKD_JOB_POLL_INTERVAL` | `1s` | How often the worker polls `_jobs` for pending work |

## Development Helpers

| Variable | Default | Description |
|---|---|---|
| `BACKD_FUNCTIONS_ROOT` | `./testdata/apps` | Root directory for function files (used in development) |
| `BACKD_CONFIG_DIR` | `./testdata/apps` | Default config directory (used in development) |

## Docker Compose Defaults

The included `docker-compose.yaml` sets these values for local development:

```yaml
environment:
  BACKD_ENCRYPTION_KEY: dGVzdC1lbmNyeXB0aW9uLWtleS0zMi1jaGFycy1sb25n
  DATABASE_URL: postgres://backd:backd123@postgres:5432/backd_platform
  BACKD_FUNCTIONS_URL: http://functions:8081
  LOG_LEVEL: debug
  MINIO_ACCESS_KEY: minioadmin
  MINIO_SECRET_KEY: minioadmin123
```

{{% details title="Generating a production encryption key" closed="true" %}}

```bash
# Generate a cryptographically random 32-byte key
openssl rand -base64 32

# Example output: dGhpcyBpcyBhIHRlc3Qga2V5IDMyIGJ5dGVz
```

Store this securely (e.g. in a secrets manager) and inject it as `BACKD_ENCRYPTION_KEY` at deploy time. Changing this key requires re-running `api secrets apply` to re-encrypt all secrets.

{{% /details %}}
