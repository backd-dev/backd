---
title: Architecture
weight: 1
---

## Overview

backd is a self-hosted Backend-as-a-Service (BaaS) for internal applications. It eliminates repetitive backend infrastructure — auth, CRUD APIs, secrets management — by letting teams define an application via a config directory. backd provisions everything automatically.

All application configuration, functions, and migrations are stored in a version-controlled directory structure, making the entire platform state auditable and reproducible.

## System Components

```
┌─────────────────────────────────────────────────────────────┐
│                        Clients                              │
│         Browser App          │        Deno Functions         │
│      (backd-js SDK)          │      (backd-js injected)     │
└──────────────┬───────────────┴──────────────┬───────────────┘
               │                              │
               ▼                              ▼
┌──────────────────────┐      ┌───────────────────────────────┐
│     API Service      │      │      Functions Service        │
│     :8080            │      │      :8081                    │
│                      │      │                               │
│  - CRUD routing      │◄─────│  - Deno process pool          │
│  - RLS enforcement   │      │  - Function execution          │
│  - Session auth      │      │  - Secret resolution           │
│  - File storage      │      │  - Full DB access              │
│  - Prometheus metrics│      │  - Prometheus metrics          │
└──────────┬───────────┘      └──────────────┬────────────────┘
           │                                 │
           ▼                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                   PostgreSQL Instance                        │
│                                                             │
│   backd_myapp  │  backd_shop  │  backd_domain_company       │
└─────────────────────────────────────────────────────────────┘
           ▲
           │ reads on startup
┌──────────┴──────────────────────────────────────────────────┐
│                     Config Root (volume)                     │
│  <app_name>/app.yaml                                        │
│  <app_name>/functions/*.ts                                  │
│  <app_name>/migrations/*.sql                                │
│  _domains/<domain_name>/domain.yaml                         │
└─────────────────────────────────────────────────────────────┘
```

<!-- TODO: Replace ASCII diagram with a proper SVG or PNG component diagram -->

## Services

backd runs as four services, deployable together or independently:

| Service | Binary | Port | Purpose |
|---|---|---|---|
| **API** | `api start` | 8080 | REST CRUD, auth, storage, metrics |
| **Functions** | `api functions` | 8081 | Deno HTTP function execution |
| **Worker** | `api functions --mode worker` | — | Background job processing |
| **CLI** | `backd` | — | Scaffold apps, validate configs (no DB needed) |

See [Services]({{< ref "/services" >}}) for detailed documentation on each service.

## Data Flow

### Browser → API

1. Browser sends request with `X-Publishable-Key` or `X-Session` header
2. Auth middleware validates the session or key
3. RLS policy is evaluated — CEL expression transpiled to SQL `WHERE` clause
4. Query executes against the app's PostgreSQL database
5. File references (`__file` columns) are resolved to URLs
6. Response returned in `{"data": ...}` envelope

### Function → API (internal)

1. Deno function receives an injected `backd` client authenticated with the app's `secret_key`
2. Secret key requests **bypass RLS entirely** — full database access
3. Functions can read secrets, enqueue jobs, and mutate user metadata
4. Communication happens via an internal HTTP handler on `127.0.0.1:9191`

## Design Decisions

### Sessions over JWT

backd uses session-based authentication stored in the `_sessions` table. Sessions are immediately revocable — unlike JWTs, which cannot be invalidated before expiry. The extra DB lookup per request is acceptable for internal tools.

### CEL for Policies

Policy expressions use a subset of [CEL (Common Expression Language)](https://github.com/google/cel-go) that is transpiled to SQL `WHERE` clauses. Filtering happens inside PostgreSQL, so pagination, indexing, and count queries all work correctly. No rows are ever fetched and then filtered in Go.

### One Database per App

Each application gets a dedicated PostgreSQL database (`backd_<app_name>`) on a shared instance. This provides strong data isolation without the operational cost of separate Postgres instances.

### GitOps Configuration

All state — config, functions, migrations — lives in a version-controlled directory. No admin UI is needed. Changes are reviewed via pull requests and deployed via standard CI/CD.

### Deno for Functions

Deno provides V8 JIT performance, native TypeScript, and a security-by-default permission model. Functions access the database via the backd internal API rather than direct PostgreSQL connections, maintaining runtime isolation.

## Security Model

| Layer | Mechanism |
|---|---|
| **Transport** | TLS (configured at load balancer / reverse proxy) |
| **Authentication** | Session tokens (Argon2id password hashing) |
| **Authorization** | CEL-based RLS policies evaluated per request |
| **Key separation** | Publishable key (client-safe) vs Secret key (server-only) |
| **Secrets** | AES-256-GCM encrypted at rest, HKDF-derived per-app keys |
| **Audit** | All secret access logged with caller and timestamp |
| **Functions** | Deno permission flags restrict network, filesystem, and env access |
