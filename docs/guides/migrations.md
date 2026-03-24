---
title: Migrations
weight: 9
---

backd uses SQL migration files to manage your database schema. Migrations run automatically on startup and can also be triggered manually.

## Migration Files

Migrations live in `<app_name>/migrations/` and follow a strict naming convention:

```
migrations/
  001_init.sql
  002_add_orders.sql
  003_add_items.sql
```

### Naming Rules

- **Format:** `NNN_description.sql` — zero-padded 3-digit prefix, underscore, description
- **Sorted by prefix:** `001`, `002`, `003` — applied in numeric order
- **Immutable:** once applied, a migration file should never be modified
- At least `001_init.sql` must exist — `backd validate` enforces this

### Example Migration

```sql
-- 001_init.sql

CREATE TABLE posts (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    content    TEXT,
    user_id    TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE orders (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    total      NUMERIC(10, 2) DEFAULT 0,
    status     TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
```

## How Migrations Run

1. backd reads all `.sql` files from the `migrations/` directory
2. Files are sorted by numeric prefix
3. Already-applied migrations are skipped (tracked in `_migrations` table)
4. Each pending migration runs in its **own transaction**
5. If a migration fails, its transaction is rolled back — previously applied migrations remain
6. The failed app is marked as FAILED (returns 503) — other apps continue normally

## Auto-Run on Startup

Migrations run automatically during `api start`:

```
INFO applied migration  file=001_init.sql
INFO applied migration  file=002_add_orders.sql
INFO migration completed  app=my_app applied_count=2
```

## Manual Migration

Run migrations explicitly without starting the full API server:

```bash
# Migrate all apps
api migrate --config-dir ./apps

# Migrate a specific app
api migrate --config-dir ./apps --app my_app
```

This is useful for CI/CD pipelines where you want to migrate before deploying.

## Idempotency

Running `api migrate` twice is safe — applied migrations are tracked and skipped:

```bash
api migrate --config-dir ./apps
# INFO migration completed  app=my_app applied_count=2

api migrate --config-dir ./apps
# INFO migration completed  app=my_app applied_count=0
```

## Reserved Tables

Never create tables starting with `_` in your migrations — they are reserved for backd:

```sql
-- ✗ Wrong — conflicts with backd system tables
CREATE TABLE _custom_logs (...);

-- ✓ Correct — user tables start with a letter
CREATE TABLE custom_logs (...);
```

Reserved tables (`_migrations`, `_users`, `_sessions`, etc.) are created automatically during bootstrap and must not appear in migration files.

## Best Practices

- **One concern per migration** — don't mix schema changes in a single file
- **Never modify applied migrations** — create a new migration for changes
- **Use `IF NOT EXISTS`** for safety — though each migration runs in a transaction
- **Test migrations locally** before deploying — `backd validate` catches syntax errors in policies but not SQL
- **Keep migrations small** — large migrations lock tables longer
