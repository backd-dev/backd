---
title: Applications
weight: 1
---

An application is the fundamental unit in backd. Each app gets its own PostgreSQL database, REST API, and configuration.

## App Directory Structure

Every application is defined by a directory under the config root:

```
<config_root>/
  my_app/
    app.yaml              # Application configuration
    migrations/
      001_init.sql        # Schema migrations (applied in order)
      002_add_orders.sql
    functions/
      hello/
        index.ts          # Public HTTP function
      _send_email/
        index.ts          # Private function (jobs only)
      _shared/
        utils.ts          # Shared utilities (not a function)
```

## Creating an App

### Using the CLI

```bash
backd bootstrap --name my_app
```

This generates:
- `apps/my_app/app.yaml` with a fresh publishable key
- `apps/my_app/migrations/001_init.sql` (empty template)
- `apps/my_app/functions/` directory
- Project-root files (`.gitignore`, `.env.example`) if not present

### App Name Rules

Names must:
- Start with a lowercase letter
- Contain only lowercase letters, digits, and underscores
- Not start with `_` (reserved for system use)

Valid: `my_app`, `hr_portal`, `shop2`
Invalid: `MyApp`, `_internal`, `123app`, `my-app`

## `app.yaml` Reference

```yaml
name: my_app
description: "My application"

database:
  # Optional — inherits DATABASE_URL if absent
  # dsn: postgres://user:pass@host:5432/
  # host: localhost
  # port: 5432
  # user: myapp_user
  # password: ${DB_PASSWORD}

auth:
  domain: company           # Optional — references _domains/company/
  session_expiry: 24h       # Default: 24h
  allow_registration: true  # Default: true

keys:
  publishable_key: vK8mN2pXqR5tL9wA3hE6yJ4cF7uB1nD0

storage:
  endpoint: https://s3.amazonaws.com
  bucket: my-app-bucket
  region: us-east-1
  access_key_id: ${S3_ACCESS_KEY}
  secret_access_key: ${S3_SECRET_KEY}

secrets:
  STRIPE_KEY: ${STRIPE_KEY}
  SENDGRID_KEY: ${SENDGRID_KEY}

jobs:
  max_attempts: 3
  timeout: 15s

cron:
  - schedule: "0 3 * * *"
    function: _cleanup_expired

policies:
  todos:
    select:
      expression: "true"
      columns: ["*"]
    insert:
      expression: "auth.authenticated"
```

See the [Configuration Reference]({{< ref "/reference/configuration" >}}) for full field documentation.

## Database Provisioning

On startup, the API service provisions each app in sequence:

1. **Create database** — `CREATE DATABASE backd_<app_name>` (idempotent)
2. **Bootstrap reserved tables** — system tables (`_migrations`, `_users`, `_sessions`, `_secrets`, `_api_keys`, `_policies`, `_files`, `_jobs`, `_secret_audit`)
3. **Run migrations** — apply pending SQL files from `migrations/`
4. **Store publishable key** — upsert into `_api_keys`
5. **Generate secret key** — create and encrypt if not present
6. **Load RLS policies** — sync `app.yaml` policies to `_policies` table and pre-compile CEL expressions

If any step fails for an app, that app is marked as **FAILED** and returns `503 Service Unavailable` for all requests. Other apps continue normally.

## Database Resolution

backd resolves the PostgreSQL connection for each app in this order:

1. `database.dsn` in `app.yaml` — use directly
2. `database.host` + individual fields — build connection string
3. `DATABASE_URL` environment variable — inherit server default
4. Error — no database configuration found

When using `DATABASE_URL`, backd replaces the database name with `backd_<app_name>` automatically. You don't need per-app DSN configuration if all apps share the same PostgreSQL instance.

## Reserved Tables

These tables are created automatically and must not be modified by migrations:

| Table | Purpose |
|---|---|
| `_migrations` | Tracks applied migration files |
| `_users` | User accounts (skipped if `auth.domain` is set) |
| `_sessions` | Active sessions (skipped if `auth.domain` is set) |
| `_secrets` | Encrypted application secrets |
| `_api_keys` | Publishable and secret API keys |
| `_policies` | RLS policy definitions (synced from `app.yaml`) |
| `_files` | File metadata for S3 storage |
| `_jobs` | Background job queue |
| `_secret_audit` | Secret access audit log |

Tables starting with `_` are not exposed via the CRUD API.

## Validation

Validate your configuration without a running database:

```bash
backd validate --config-dir ./apps
```

Flags:
- `--json` — output structured JSON (for CI)
- `--check-env` — verify all `${VAR}` references exist in environment

The validator collects **all** errors before reporting — it never short-circuits on the first problem.
