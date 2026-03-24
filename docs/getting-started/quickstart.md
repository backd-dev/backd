---
title: Quickstart
weight: 2
---

Create your first backd application and make API calls in 5 minutes.

## 1. Scaffold a New App

```bash
build/backd bootstrap --name my_app
```

This creates:

```
apps/
  my_app/
    app.yaml              # Application configuration
    migrations/
      001_init.sql        # Initial migration (empty template)
    functions/            # Deno functions directory
```

It also generates project-root files (`docker-compose.yaml`, `.gitignore`, `.env.example`) if they don't already exist.

## 2. Define Your Schema

Edit `apps/my_app/migrations/001_init.sql`:

```sql
CREATE TABLE todos (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    completed  BOOLEAN DEFAULT false,
    user_id    TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

## 3. Add an RLS Policy

Edit `apps/my_app/app.yaml` and add policies:

```yaml
policies:
  todos:
    select:
      expression: "true"
      columns: ["*"]
    insert:
      expression: "true"
      defaults:
        created_at: "now()"
    update:
      expression: "true"
    delete:
      expression: "true"
```

This allows all operations on the `todos` table. See [Policies]({{< ref "/guides/policies" >}}) for auth-based access control.

## 4. Validate Configuration

```bash
build/backd validate --config-dir ./apps
```

Expected output:
```
Validation Results
=================

Apps (1):
  ✓ my_app

✓ All configurations are valid
```

## 5. Start the Stack

```bash
make docker-up
```

Wait for the API to provision your app (check logs with `docker logs backd-api`). You should see:

```
INFO app provisioned  app=my_app status=ready
INFO starting API server  port=8080
```

## 6. Make API Calls

### Insert a record

```bash
curl -X POST http://localhost:8080/v1/my_app/todos \
  -H "Content-Type: application/json" \
  -H "X-Publishable-Key: <your-key-from-app.yaml>" \
  -d '{"title": "Learn backd", "completed": false}'
```

Response:
```json
{
  "data": {
    "id": "d6vt1naftvmlb17e02l0",
    "title": "Learn backd",
    "completed": false,
    "created_at": "2026-03-22T10:00:00Z"
  }
}
```

### List records

```bash
curl http://localhost:8080/v1/my_app/todos \
  -H "X-Publishable-Key: <your-key>"
```

Response:
```json
{
  "data": [
    { "id": "d6vt1naftvmlb17e02l0", "title": "Learn backd", "completed": false }
  ],
  "count": 1,
  "limit": 50,
  "offset": 0
}
```

### Filter with `where`

```bash
curl "http://localhost:8080/v1/my_app/todos?where=%7B%22completed%22%3A%7B%22%24eq%22%3Afalse%7D%7D" \
  -H "X-Publishable-Key: <your-key>"
```

The `where` parameter is URL-encoded JSON: `{"completed":{"$eq":false}}`

## 7. Use the SDK

Install the JavaScript SDK:

```bash
npm install backd-js
```

```typescript
import { createClient } from 'backd-js'

const backd = createClient({
  api: 'http://localhost:8080/v1/my_app',
  auth: 'http://localhost:8080/v1/_auth/my_app',
  functions: 'http://localhost:8081/v1/my_app',
  publishableKey: '<your-key-from-app.yaml>',
})

// List todos
const { data, count } = await backd.from('todos').list()

// Insert a todo
const todo = await backd.from('todos').insert({
  title: 'Build something great',
})
```

## Next Steps

- [Applications Guide]({{< ref "/guides/apps" >}}) — understand the full app lifecycle
- [Authentication]({{< ref "/guides/auth" >}}) — add user registration and login
- [Policies]({{< ref "/guides/policies" >}}) — restrict access based on user identity
- [Functions]({{< ref "/guides/functions" >}}) — add custom server-side logic with Deno
