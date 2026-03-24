---
title: CLI
weight: 4
---

The `backd` CLI is a lightweight binary for local development. It performs filesystem-only operations — no database connection required.

## Commands

### `backd bootstrap`

Scaffolds a new application or domain directory.

```bash
# Create a new app
backd bootstrap --name my_app

# Create a new domain
backd bootstrap --domain --name company

# Overwrite existing files
backd bootstrap --name my_app --force
```

#### What Gets Created

**App bootstrap:**
```
apps/my_app/
  app.yaml              # Config with generated publishable key
  migrations/
    001_init.sql        # Empty migration template
  functions/            # Empty functions directory
```

**Domain bootstrap:**
```
_domains/company/
  domain.yaml           # Domain config with password provider
```

**Project root files** (first bootstrap only, skipped if exist):
- `docker-compose.yaml`
- `.gitignore`
- `.env.example`
- `.env.secrets.example`

#### Publishable Key Generation

Each bootstrapped app gets a fresh 32-byte random publishable key encoded as base64url:

```yaml
keys:
  publishable_key: vK8mN2pXqR5tL9wA3hE6yJ4cF7uB1nD0
```

This key is safe to commit to version control — it identifies the app but doesn't grant privileged access.

### `backd validate`

Validates all configuration files without a running database.

```bash
# Validate all apps and domains
backd validate --config-dir ./apps

# JSON output (for CI)
backd validate --config-dir ./apps --json

# Also check environment variables
backd validate --config-dir ./apps --check-env
```

#### What Gets Validated

1. **Domain configs** — name format, provider field, required fields
2. **App configs** — name format, publishable key, database config, storage config
3. **Cross-references** — `auth.domain` references a real domain
4. **CEL expressions** — all RLS policy expressions are parsed and validated against the supported CEL subset
5. **Environment variables** (with `--check-env`) — all `${VAR}` references resolved

#### Output Symbols

| Symbol | Meaning |
|---|---|
| `✓` | Check passed |
| `✗` | Hard error — blocks deployment |
| `⚠` | Warning — non-blocking |

#### Exit Codes

| Code | Meaning |
|---|---|
| `0` | All valid |
| `1` | Validation errors found |
| `2` | Config root not found |

#### JSON Output

With `--json`, produces structured output for CI tooling:

```json
{
  "valid": true,
  "apps": [
    { "name": "my_app", "valid": true, "errors": [], "warnings": [] }
  ],
  "domains": [
    { "name": "company", "valid": true, "errors": [], "warnings": [] }
  ]
}
```

#### Collecting All Errors

The validator **never short-circuits** — it collects all errors across all apps and domains before reporting. This avoids fix-one-rerun-find-another loops.

## Import Restrictions

The `backd` CLI binary only imports:
- `internal/config` — YAML parsing and validation
- `internal/celql` — CEL expression validation

It **never** imports `internal/db`, `internal/auth`, `internal/secrets`, or any package that requires a database connection. This ensures the CLI works on developer machines without infrastructure.

## Runtime Commands

These commands are part of the `api` binary (not `backd`) because they require database access:

| Command | Description |
|---|---|
| `api start` | Start the API service |
| `api functions` | Start the functions/worker service |
| `api migrate` | Run pending migrations |
| `api secrets apply` | Encrypt and store secrets |

See [API Service]({{< ref "/services/api" >}}), [Functions Service]({{< ref "/services/functions-service" >}}), and [Secrets]({{< ref "/guides/secrets" >}}) for details.
