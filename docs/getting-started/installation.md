---
title: Installation
weight: 1
---

## Prerequisites

- **Go 1.26+** — [install instructions](https://go.dev/doc/install)
- **Docker** or **Podman** — for running PostgreSQL, MinIO, and backd services
- **Docker Compose** (or `podman-compose`) — for the development stack
- **Deno 2.x** — required only if running functions locally outside Docker

## Clone and Build

```bash
git clone https://github.com/backd-dev/backd.git
cd backd

# Build both binaries
make build
```

This produces two binaries in `build/`:

| Binary | Purpose |
|---|---|
| `build/backd` | CLI — scaffold apps, validate configs (no database required) |
| `build/api` | Runtime — API server, functions service, worker, migrations |

## Start the Development Stack

The included `docker-compose.yaml` starts all required services:

```bash
make docker-up
```

This starts:

| Service | Port | Description |
|---|---|---|
| **postgres** | 5432 | PostgreSQL 16 database |
| **minio** | 9000 / 9001 | S3-compatible object storage |
| **api** | 8080 | backd API service |
| **functions** | 8081 | Deno functions HTTP service |
| **worker** | — | Background job processor |

## Verify

```bash
# Health check
curl http://localhost:8080/health
# → {"status":"ok"}

# Readiness check (shows app status)
curl http://localhost:8080/ready
```

## Generate an Encryption Key

backd requires a 32-byte encryption key for secrets management. Generate one:

```bash
openssl rand -base64 32
```

Set it as the `BACKD_ENCRYPTION_KEY` environment variable before starting the API service. The development `docker-compose.yaml` includes a pre-generated test key.

{{% details title="Environment variables" closed="true" %}}

See the [Environment Variables Reference]({{< ref "/reference/environment" >}}) for the full list of configuration options.

{{% /details %}}
