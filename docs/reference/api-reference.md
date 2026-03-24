---
title: API Reference
weight: 2
---

Complete REST API endpoint reference for all backd services.

## CRUD Endpoints

Base URL: `/v1/<app>/<table>`

### List Rows

```
GET /v1/<app>/<table>
```

| Parameter | Type | Default | Max | Description |
|---|---|---|---|---|
| `where` | URL-encoded JSON | — | — | Filter expression |
| `select` | string | all | — | Comma-separated columns |
| `order` | string | — | — | `column:asc\|desc`, comma-separated |
| `limit` | int | `50` | `1000` | Rows to return |
| `offset` | int | `0` | — | Pagination offset |

**Response `200`:**
```json
{
  "data": [ { "id": "...", ... } ],
  "count": 42,
  "limit": 50,
  "offset": 0
}
```

### Get Row

```
GET /v1/<app>/<table>/<id>
```

**Response `200`:**
```json
{ "data": { "id": "...", ... } }
```

**Response `404`:** Row not found or excluded by RLS.

### Insert Row

```
POST /v1/<app>/<table>
Content-Type: application/json

{ "title": "...", "status": "..." }
```

- `id` auto-generated (XID)
- `created_at` auto-stamped
- RLS defaults applied before insert

**Response `200`:**
```json
{ "data": { "id": "d6vt...", "title": "...", "created_at": "..." } }
```

### Replace Row

```
PUT /v1/<app>/<table>/<id>
Content-Type: application/json

{ "title": "...", "status": "..." }
```

- `id` and `created_at` are immutable (ignored if sent)
- `updated_at` auto-stamped

**Response `200`:** Full updated row.

### Partial Update

```
PATCH /v1/<app>/<table>/<id>
Content-Type: application/json

{ "status": "shipped" }
```

Only updates the provided fields. Same response as PUT.

### Delete Row

```
DELETE /v1/<app>/<table>/<id>
```

**Response `204`:** No content. If soft delete is configured, stamps the soft column with `NOW()` instead of removing the row.

---

## Auth Endpoints

Base URL: `/v1/_auth/<domain_or_app>`

### Register

```
POST /v1/_auth/<domain>/local/register
Content-Type: application/json

{ "username": "alice", "password": "strong-pass-123!" }
```

**Response `200`:**
```json
{
  "data": {
    "id": "d6vt...",
    "username": "alice",
    "type": "user",
    "metadata": {},
    "created_at": "2026-03-22T10:00:00Z"
  }
}
```

| Error | Status | Condition |
|---|---|---|
| `USER_EXISTS` | 400 | Username already taken |
| `MISSING_FIELDS` | 400 | Username or password empty |

### Login

```
POST /v1/_auth/<domain>/local/login
Content-Type: application/json

{ "username": "alice", "password": "strong-pass-123!" }
```

**Response `200`:**
```json
{
  "data": {
    "id": "session-id",
    "user_id": "d6vt...",
    "app_name": "my_app",
    "token": "a1b2c3...",
    "expires_at": "2026-03-23T10:00:00Z"
  }
}
```

| Error | Status | Condition |
|---|---|---|
| `UNAUTHORIZED` | 401 | Invalid credentials |
| `MISSING_FIELDS` | 400 | Username or password empty |

### Refresh / Me

```
POST /v1/_auth/<domain>/refresh
Content-Type: application/json

{ "token": "<session-token>" }
```

Returns the current user profile if the session is valid.

| Error | Status | Condition |
|---|---|---|
| `SESSION_EXPIRED` | 401 | Token expired or invalid |

### Logout

```
POST /v1/_auth/<domain>/logout
Content-Type: application/json

{ "token": "<session-token>" }
```

**Response `204`:** No content.

### Update Profile

```
PATCH /v1/_auth/<domain>/profile
X-Session: <token>
Content-Type: application/json

{ "username": "new_name" }
```

Supports `username` and `password` fields.

---

## Storage Endpoints

Base URL: `/v1/<app>/storage`

### Upload File

```
POST /v1/<app>/storage/upload
Content-Type: multipart/form-data

file=@photo.jpg
```

**Response `200`:**
```json
{
  "data": {
    "id": "d6vt...",
    "filename": "photo.jpg",
    "size": 245760,
    "secure": false
  }
}
```

| Error | Status | Condition |
|---|---|---|
| `STORAGE_DISABLED` | 501 | No storage config for this app |
| `MISSING_FILE` | 400 | No file in form |

### Delete File

```
DELETE /v1/<app>/storage/<file_id>
```

**Response `204`:** No content.

---

## Functions Endpoints

### Invoke Function

```
POST /v1/<app>/<function_name>
Content-Type: application/json

{ "order_id": "abc123" }
```

Response is the raw function return value.

| Error | Status | Condition |
|---|---|---|
| `FUNCTION_NOT_FOUND` | 404 | Function doesn't exist or starts with `_` |
| `FUNCTION_TIMEOUT` | 504 | Execution exceeded timeout |

---

## Internal Endpoints

Base URL: `http://127.0.0.1:9191` (localhost only)

### Enqueue Job

```
POST /internal/jobs
Content-Type: application/json

{
  "app": "my_app",
  "function": "_send_email",
  "input": "{\"to\":\"user@example.com\"}",
  "trigger": "sdk"
}
```

**Response `200`:**
```json
{ "data": { "job_id": "d6vt..." } }
```

### Health Check

```
GET /health
```

**Response `200`:**
```json
{ "status": "ok", "service": "backd-internal-api" }
```

---

## Authentication Headers

Every CRUD request must include one of:

| Header | Value | Context |
|---|---|---|
| `X-Publishable-Key` | Key from `app.yaml` | Browser clients — RLS enforced |
| `X-Session` | Session token | Authenticated users — RLS enforced |
| `Cookie: backd_session` | Session token | Browser cookie — RLS enforced |
| `X-Secret-Key` | App secret key | Functions internal — RLS bypassed |

---

## Response Envelope

| Case | Shape |
|---|---|
| Single row | `{ "data": {...} }` |
| List | `{ "data": [...], "count": N, "limit": N, "offset": N }` |
| No content | HTTP 204, empty body |
| Error | `{ "error": "CODE", "error_detail": "human message" }` |
