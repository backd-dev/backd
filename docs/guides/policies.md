---
title: Policies & RLS
weight: 4
---

backd provides row-level security (RLS) that controls who can read and write data through the REST API. Policies are defined in `app.yaml` using CEL expressions that are transpiled to SQL `WHERE` clauses at request time.

The default posture is **deny all** — if no policy exists for a table+operation, the request is rejected with `403 Forbidden`.

## Defining Policies

Policies are declared in `app.yaml` under the `policies` key, grouped by table name and operation:

```yaml
policies:
  posts:
    select:
      expression: "true"
      columns: ["id", "title", "body", "created_at", "user_id"]
    insert:
      expression: "auth.authenticated"
      columns: ["title", "body"]
      defaults:
        user_id: "auth.uid"
        created_at: "now()"
    update:
      expression: "auth.uid == row.user_id"
      columns: ["title", "body"]
      defaults:
        updated_at: "now()"
    delete:
      expression: "auth.uid == row.user_id"
      soft: deleted_at
```

## Policy Fields

| Field | Operations | Required | Description |
|---|---|---|---|
| `expression` | All | Yes | CEL boolean expression — must evaluate to `true` to allow access |
| `check` | INSERT, UPDATE | No | Additional CEL guard evaluated against the written payload (like SQL `WITH CHECK`) |
| `columns` | SELECT, INSERT, UPDATE | No | Column allowlist. `["*"]` = all columns. Omit to allow all |
| `defaults` | INSERT, UPDATE, DELETE (soft) | No | Map of `column: value` applied before the query — cannot be overridden by the client |
| `soft` | DELETE only | No | Column name for soft deletion — DELETE becomes `UPDATE SET <col> = NOW()` |

## Expression Language

Expressions use a **strict subset of CEL** (Common Expression Language) that is fully transpilable to SQL `WHERE` clauses. Filtering happens inside PostgreSQL — not in Go — so pagination, indexes, and counts all work correctly.

### Available Variables

| Variable | Type | Description |
|---|---|---|
| `auth.uid` | string | Authenticated user's ID (empty if unauthenticated) |
| `auth.authenticated` | bool | Whether the request has a valid session |
| `auth.meta.<key>` | any | Global user metadata value |
| `auth.meta_app.<key>` | any | App-scoped user metadata value |
| `auth.key_type` | string | `"publishable"`, `"secret"`, or `""` |
| `row.<column>` | any | Value of a column in the existing row (SELECT, UPDATE, DELETE) |
| `input.<column>` | any | Value from the request payload (INSERT, UPDATE) |
| `true` / `false` | bool | Literal booleans |
| `now()` | timestamp | Current timestamp |
| `today()` | date | Current date |

### Supported Operators

| CEL Expression | SQL Equivalent |
|---|---|
| `row.x == value` | `x = $N` |
| `row.x != value` | `x != $N` |
| `row.x == null` | `x IS NULL` |
| `row.x != null` | `x IS NOT NULL` |
| `row.x > value` | `x > $N` |
| `row.x >= value` | `x >= $N` |
| `row.x < value` | `x < $N` |
| `row.x <= value` | `x <= $N` |
| `row.x in ['a','b']` | `x = ANY($N)` |
| `has(auth.meta.role)` | `$N IS NOT NULL` |
| `expr && expr` | `(expr AND expr)` |
| `expr \|\| expr` | `(expr OR expr)` |
| `!expr` | `NOT (expr)` |

Expressions that use constructs outside this subset are rejected at startup and by `backd validate`.

## Common Patterns

### Public read, authenticated write

```yaml
posts:
  select:
    expression: "true"
  insert:
    expression: "auth.authenticated"
```

### Owner-only access

```yaml
orders:
  select:
    expression: "auth.uid == row.user_id"
  update:
    expression: "auth.uid == row.user_id"
  delete:
    expression: "auth.uid == row.user_id"
```

### Role-based access

```yaml
admin_settings:
  select:
    expression: "auth.meta.role == 'admin'"
  update:
    expression: "auth.meta.role == 'admin'"
```

### Time-based visibility

```yaml
promotions:
  select:
    expression: "row.starts_at <= now() && row.ends_at >= now()"
```

### Soft delete with auto-stamp

```yaml
orders:
  delete:
    expression: "auth.uid == row.user_id"
    soft: deleted_at
    defaults:
      deleted_by: "auth.uid"
```

When a DELETE request arrives, backd converts it to:
```sql
UPDATE orders SET deleted_at = NOW(), deleted_by = $1 WHERE user_id = $2 AND id = $3
```

Soft-deleted rows are automatically excluded from SELECT queries when a `soft` column is defined.

### Auto-populated fields

```yaml
posts:
  insert:
    expression: "auth.authenticated"
    defaults:
      user_id: "auth.uid"
      created_at: "now()"
      updated_at: "now()"
```

Defaults are applied **after** column stripping and **before** the query executes. They cannot be overridden by the client.

### Available default values

| Value | Resolves to |
|---|---|
| `auth.uid` | Current user's ID |
| `auth.meta.<key>` | Global metadata value |
| `now()` | SQL `NOW()` |
| `today()` | SQL `CURRENT_DATE` |
| Any literal | Used as-is |

## Secret Key Bypass

Requests authenticated with the `X-Secret-Key` header (used by Deno functions) bypass RLS entirely. The policy evaluation returns `TRUE` for all operations, giving functions full database access.

## Unauthenticated Access

If a policy expression references `auth.*` variables and the request is unauthenticated, the request is denied with `403 Forbidden` immediately — no query is executed.

Policies using `expression: "true"` allow unauthenticated access (useful for public read endpoints).

## Policy Lifecycle

1. Developer edits `app.yaml` policies
2. On startup, backd syncs policies to the `_policies` table (full replace)
3. CEL expressions are parsed and validated — invalid expressions fail the app startup
4. At request time, the cached CEL AST is transpiled with the current auth context
5. The resulting SQL `WHERE` clause is injected into the database query
