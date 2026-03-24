---
title: JavaScript / TypeScript SDK
weight: 1
---

`backd-js` is the TypeScript client SDK for backd. It works in browsers, Node.js, and Deno with a single API surface.

## Installation

```bash
# npm
npm install backd-js

# Deno (JSR)
deno add @backd/js
```

### CDN Usage

```html
<script type="module">
  import { createClient } from 'https://cdn.jsdelivr.net/npm/backd-js/dist/index.esm.js'
</script>
```

## Client Initialization

### Browser / Node.js

```typescript
import { createClient } from 'backd-js'

const backd = createClient({
  api:            'https://api.example.com/v1/my_app',
  auth:           'https://api.example.com/v1/_auth/company',
  functions:      'https://functions.example.com/v1/my_app',
  publishableKey: 'vK8mN2pXqR5tL9wA3hE6yJ4cF7uB1nD0',
  storage:        localStorage,   // optional session persistence
})
```

### Deno (inside functions)

```typescript
import { createClient } from 'backd-js/deno'

// No arguments needed — self-configures from env vars
const backd = createClient()
```

### Options

| Option | Type | Required | Description |
|---|---|---|---|
| `api` | string | Yes | Base URL for CRUD API |
| `auth` | string | Yes | Base URL for auth endpoints |
| `functions` | string | Yes | Base URL for functions service |
| `publishableKey` | string | Yes | App publishable key from `app.yaml` |
| `storage` | StorageAdapter | No | Session persistence (default: in-memory) |

## Authentication

```typescript
// Register
const user = await backd.auth.signUp('alice', 'strong-password-123!')

// Login — stores session token automatically
await backd.auth.signIn('alice', 'strong-password-123!')

// Get current user
const me = await backd.auth.me()

// Check auth state
if (backd.auth.isAuthenticated()) {
  console.log('Token:', backd.auth.token())
}

// Logout
await backd.auth.signOut()
```

### Auth State Changes

```typescript
backd.auth.onAuthStateChange((event, session) => {
  if (event === 'SIGNED_IN') {
    console.log('User signed in:', session.user)
  }
  if (event === 'SIGNED_OUT') {
    console.log('User signed out')
  }
})
```

## Data Queries

### List with Filtering

```typescript
const { data, count } = await backd.from('orders')
  .where({ status: { $eq: 'pending' } })
  .order('created_at', 'desc')
  .limit(20)
  .offset(0)
  .list()
```

### Get Single Row

```typescript
const order = await backd.from('orders').get('abc123')
```

### Insert

```typescript
const newOrder = await backd.from('orders').insert({
  total: 99.99,
  status: 'pending',
})
```

### Update (Replace)

```typescript
const updated = await backd.from('orders').update('abc123', {
  total: 149.99,
  status: 'confirmed',
})
```

### Partial Update

```typescript
const patched = await backd.from('orders').patch('abc123', {
  status: 'shipped',
})
```

### Delete

```typescript
await backd.from('orders').delete('abc123')
```

### Select Columns

```typescript
const { data } = await backd.from('orders')
  .select('id', 'total', 'status')
  .list()
```

### Chaining

All query methods are chainable:

```typescript
const { data } = await backd.from('products')
  .where({
    $and: [
      { price: { $gte: 10 } },
      { price: { $lte: 100 } },
    ]
  })
  .select('id', 'name', 'price')
  .order('price', 'asc')
  .limit(10)
  .list()
```

## Functions

```typescript
// Call a public function
const result = await backd.functions.call('send_invoice', {
  order_id: 'abc123',
})

// With custom headers
const result = await backd.functions.call('webhook', payload, {
  headers: { 'X-Webhook-Secret': '...' },
})
```

Private functions (`_` prefix) are rejected at the SDK level before any HTTP call:

```typescript
// Throws FunctionError immediately — no network request
await backd.functions.call('_send_email', {})
```

## Jobs (Deno only)

Available only in the Deno entry point (`backd-js/deno`):

```typescript
import { createClient } from 'backd-js/deno'
const backd = createClient()

// Enqueue immediate job
const jobId = await backd.jobs.enqueue('_send_email', {
  to: 'user@example.com',
})

// Delayed job
await backd.jobs.enqueue('_generate_report', { type: 'daily' }, {
  delay: '5m',
})

// Custom retry
await backd.jobs.enqueue('_process_payment', { order_id: '...' }, {
  maxAttempts: 5,
})
```

## Secrets (Deno only)

```typescript
const stripeKey = await backd.secrets.get('STRIPE_KEY')
```

Each call decrypts the secret from the database — values are never cached.

## Error Handling

The SDK returns typed errors:

```typescript
import { AuthError, QueryError, FunctionError } from 'backd-js'

try {
  await backd.auth.signIn('alice', 'wrong-password')
} catch (err) {
  if (err instanceof AuthError) {
    console.log(err.code)    // "UNAUTHORIZED"
    console.log(err.detail)  // "Invalid credentials"
  }
}
```

| Error Type | Codes |
|---|---|
| `AuthError` | `UNAUTHORIZED`, `SESSION_EXPIRED`, `TOO_MANY_REQUESTS` |
| `QueryError` | `FORBIDDEN`, `NOT_FOUND`, `VALIDATION_ERROR` |
| `FunctionError` | `FUNCTION_NOT_FOUND`, `FUNCTION_TIMEOUT`, `FUNCTION_ERROR` |
| `NetworkError` | `NETWORK_ERROR` |

## Session Storage Adapters

```typescript
import { createClient, memoryStorage, localStorageAdapter } from 'backd-js'

// In-memory (default) — lost on refresh
createClient({ ..., storage: memoryStorage() })

// localStorage — persists across tabs
createClient({ ..., storage: localStorageAdapter() })

// Custom adapter
createClient({ ..., storage: {
  get: (key) => sessionStorage.getItem(key),
  set: (key, value) => sessionStorage.setItem(key, value),
  delete: (key) => sessionStorage.removeItem(key),
}})
```

Interface:
```typescript
interface StorageAdapter {
  get(key: string): string | null | Promise<string | null>
  set(key: string, value: string): void | Promise<void>
  delete(key: string): void | Promise<void>
}
```
