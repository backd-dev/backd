---
title: Authentication
weight: 2
---

backd provides session-based authentication with Argon2id password hashing. Users can be scoped to individual apps or shared across apps via domains.

## Auth Models

### Local Auth (per-app)

When no `auth.domain` is set in `app.yaml`, the app manages its own users:

- Users stored in the app's `_users` table
- Sessions stored in the app's `_sessions` table
- Auth endpoints served at `/v1/_auth/<app_name>/...`

### Domain Auth (shared users)

When `auth.domain` is set, multiple apps share a single user pool:

- Users stored in `backd_domain_<domain>._users`
- Sessions stored in `backd_domain_<domain>._sessions`
- Auth endpoints served at `/v1/_auth/<domain>/...`

```yaml
# hr_portal/app.yaml
auth:
  domain: company    # references _domains/company/domain.yaml
```

```yaml
# expenses/app.yaml
auth:
  domain: company    # same domain — shared user pool
```

Both apps share the same users but have independent RLS policies, tables, and functions.

## Domains

A domain is an independent user pool with its own PostgreSQL database.

### Creating a Domain

```bash
backd bootstrap --domain --name company
```

This creates `_domains/company/domain.yaml`:

```yaml
name: company
description: "Company-wide user pool"
provider: password
session_expiry: 24h
allow_registration: true
```

### Domain Database

Named `backd_domain_<name>`. Contains only reserved system tables:

| Table | Purpose |
|---|---|
| `_migrations` | Migration tracking |
| `_users` | User accounts |
| `_sessions` | Active sessions (with `app_name` column for scoping) |
| `_secrets` | Domain-level secrets |

### Providers

Currently supported: `password` (Argon2id hashing).

Future providers (not yet implemented): `google`, `github`, `saml`, `oidc`.

## Auth Endpoints

All auth operations use the `/v1/_auth/<domain_or_app>/` URL prefix.

### Register

```bash
POST /v1/_auth/<domain>/local/register
Content-Type: application/json

{
  "username": "alice",
  "password": "strong-password-123!"
}
```

Response:
```json
{
  "data": {
    "id": "d6vt1naftvmlb17e02l0",
    "username": "alice",
    "type": "user",
    "metadata": {},
    "created_at": "2026-03-22T10:00:00Z"
  }
}
```

### Login

```bash
POST /v1/_auth/<domain>/local/login
Content-Type: application/json

{
  "username": "alice",
  "password": "strong-password-123!"
}
```

Response:
```json
{
  "data": {
    "id": "session-id",
    "user_id": "d6vt1naftvmlb17e02l0",
    "app_name": "hr_portal",
    "token": "a1b2c3d4e5f6...",
    "expires_at": "2026-03-23T10:00:00Z"
  }
}
```

Store the `token` and include it in subsequent requests via the `X-Session` header.

### Get Current User

```bash
POST /v1/_auth/<domain>/refresh
X-Session: <token>
Content-Type: application/json

{ "token": "<session-token>" }
```

Returns the current user profile.

### Logout

```bash
POST /v1/_auth/<domain>/logout
Content-Type: application/json

{ "token": "<session-token>" }
```

Returns `204 No Content` on success.

### Update Profile

```bash
PATCH /v1/_auth/<domain>/profile
X-Session: <token>
Content-Type: application/json

{ "username": "alice_new" }
```

## Session Scoping

Sessions are scoped to the app they were created for:

- `_sessions.app_name` records which app the session belongs to
- A session for `hr_portal` is only valid when accessing `hr_portal` data
- The session token carries no app information — scoping is enforced server-side

## User Metadata

Users have two metadata namespaces:

- **Global metadata** (`metadata._`) — shared across all apps using the domain
- **App metadata** (`metadata.<app_name>`) — scoped to a specific app

Metadata is stored as JSONB and accessed in RLS policies via `auth.meta` and `auth.meta_app`.

### Setting Metadata (server-side only)

From Deno functions, use the SDK:

```typescript
// Set global metadata
await backd.auth.setGlobalMeta(userId, 'department', 'engineering')

// Set app-specific metadata
await backd.auth.setAppMeta(userId, 'role', 'admin')
```

## Authentication Headers

| Header | Value | Context |
|---|---|---|
| `X-Publishable-Key` | Key from `app.yaml` | Browser clients — identifies the app |
| `X-Session` | Session token | Authenticated users |
| `Cookie: backd_session=<token>` | Session token | Browser cookie alternative |
| `X-Secret-Key` | App secret key | Functions internal use only |

The auth middleware checks headers in order: session → publishable key → secret key. Unauthenticated requests proceed without auth context — RLS policies decide whether access is allowed.
