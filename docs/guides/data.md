---
title: Data & CRUD
weight: 3
---

backd auto-generates a full REST API for every user-created table in your app's database. No code generation, no ORM — just define tables via SQL migrations and backd serves them immediately.

## Base URL

```
http://localhost:8080/v1/<app_name>/<table_name>
```

- `<app_name>` matches the `name` field in `app.yaml`
- `<table_name>` is any user table (tables starting with `_` are reserved and not exposed)

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/<table>` | List rows (paginated, filterable) |
| `GET` | `/<table>/<id>` | Get a single row |
| `POST` | `/<table>` | Insert a new row |
| `PUT` | `/<table>/<id>` | Replace a row entirely |
| `PATCH` | `/<table>/<id>` | Partial update |
| `DELETE` | `/<table>/<id>` | Delete a row (hard or soft) |

All requests must include an authentication header — see [Authentication]({{< ref "/guides/auth" >}}).

## List Rows

```
GET /v1/my_app/todos
```

### Query Parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `where` | URL-encoded JSON | — | Filter expression (see [Filtering](#filtering)) |
| `select` | string | all columns | Comma-separated column names: `select=id,title,status` |
| `order` | string | — | Sort: `order=created_at:desc,status:asc` |
| `limit` | integer | `50` | Max rows (capped at `1000`) |
| `offset` | integer | `0` | Pagination offset |

### Response

```json
{
  "data": [
    { "id": "abc123", "title": "Buy groceries", "completed": false },
    { "id": "def456", "title": "Write docs", "completed": true }
  ],
  "count": 42,
  "limit": 50,
  "offset": 0
}
```

`count` is the **total** matching rows (ignoring `limit`/`offset`) — useful for pagination UI.

## Get Single Row

```
GET /v1/my_app/todos/abc123
```

```json
{
  "data": {
    "id": "abc123",
    "title": "Buy groceries",
    "completed": false,
    "created_at": "2026-03-22T10:00:00Z"
  }
}
```

Returns `404` if the row doesn't exist or is excluded by the RLS policy.

## Insert Row

```
POST /v1/my_app/todos
Content-Type: application/json

{ "title": "New task", "completed": false }
```

- An `id` is generated automatically (XID)
- `created_at` is stamped with `NOW()`
- RLS defaults (e.g. `user_id: "auth.uid"`) are applied before insert
- Returns the full inserted row via `RETURNING *`

```json
{
  "data": {
    "id": "d6vt1naf...",
    "title": "New task",
    "completed": false,
    "created_at": "2026-03-22T10:00:00Z"
  }
}
```

## Update Row

```
PUT /v1/my_app/todos/abc123
Content-Type: application/json

{ "title": "Updated task", "completed": true }
```

Replaces all mutable fields. `id` and `created_at` are immutable and ignored if sent. `updated_at` is stamped automatically. Returns the full updated row.

## Partial Update

```
PATCH /v1/my_app/todos/abc123
Content-Type: application/json

{ "completed": true }
```

Updates only the provided fields. Same behavior as PUT for the supplied keys.

## Delete Row

```
DELETE /v1/my_app/todos/abc123
```

Returns `204 No Content` on success.

If the RLS policy specifies a `soft` delete column, the row is not actually deleted — instead, the soft delete column is stamped with `NOW()` and the row is excluded from future SELECT queries.

## Filtering

The `where` parameter accepts URL-encoded JSON with field-centric operators:

```
?where={"status":{"$eq":"pending"}}
```

### Operators

| Operator | SQL | Example |
|---|---|---|
| `$eq` | `=` | `{"status":{"$eq":"active"}}` |
| `$ne` | `!=` | `{"status":{"$ne":"deleted"}}` |
| `$gt` | `>` | `{"price":{"$gt":100}}` |
| `$gte` | `>=` | `{"price":{"$gte":100}}` |
| `$lt` | `<` | `{"stock":{"$lt":5}}` |
| `$lte` | `<=` | `{"stock":{"$lte":5}}` |
| `$in` | `= ANY(...)` | `{"status":{"$in":["active","pending"]}}` |
| `$nin` | `!= ALL(...)` | `{"status":{"$nin":["deleted"]}}` |
| `$null` | `IS NULL` / `IS NOT NULL` | `{"deleted_at":{"$null":true}}` |
| `$like` | `LIKE` | `{"name":{"$like":"%widget%"}}` |
| `$ilike` | `ILIKE` | `{"name":{"$ilike":"%Widget%"}}` |

### Logical Operators

```json
{
  "$and": [
    {"status": {"$eq": "active"}},
    {"price": {"$gt": 50}}
  ]
}
```

```json
{
  "$or": [
    {"status": {"$eq": "active"}},
    {"featured": {"$eq": true}}
  ]
}
```

```json
{
  "$not": {"status": {"$eq": "deleted"}}
}
```

Logical operators can be nested arbitrarily.

## Response Envelope

All responses follow a consistent envelope:

| Case | Shape |
|---|---|
| Single row | `{"data": {...}}` |
| List | `{"data": [...], "count": N, "limit": N, "offset": N}` |
| No content | HTTP 204, empty body |
| Error | `{"error": "CODE", "error_detail": "human message"}` |

## Column Handling

### RLS Column Allowlist

If a policy defines `columns: ["id", "title", "status"]`, only those columns appear in responses. Other columns are stripped from both request payloads (on write) and response data (on read).

Use `columns: ["*"]` to allow all columns.

### File Columns

Columns ending with `__file` (double underscore) are automatically resolved to file descriptors in API responses. See [File Storage]({{< ref "/guides/storage" >}}).

### Immutable Fields

The `id` and `created_at` fields are never updated by PUT or PATCH operations, even if included in the request body.
