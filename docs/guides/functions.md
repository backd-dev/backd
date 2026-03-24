---
title: Functions
weight: 5
---

backd functions are Deno TypeScript scripts that run server-side with full database access. They serve as HTTP endpoints (public functions) and background job handlers (private functions).

## Directory Layout

Functions live under `<app_name>/functions/`:

```
my_app/
  functions/
    send_invoice/           → POST /v1/my_app/send_invoice  (public)
      index.ts              ← entrypoint (always index.ts)
      helpers.ts            ← local helpers
    process_order/          → POST /v1/my_app/process_order (public)
      index.ts
    _send_email/            → not HTTP-exposed (private, jobs only)
      index.ts
    _cleanup/               → not HTTP-exposed (private, cron)
      index.ts
    _shared/                → import-only utilities (not a function)
      stripe.ts
      email.ts
```

## Public vs Private

The underscore prefix (`_`) is the single rule governing visibility:

| Directory | HTTP endpoint | Enqueueable as job | Importable |
|---|---|---|---|
| `send_invoice/` | `POST /v1/<app>/send_invoice` | Yes | Yes |
| `_send_email/` | Not reachable via HTTP | Yes | Yes |
| `_shared/` | Not reachable | No (no `index.ts`) | Yes |

- Directories starting with a **letter or digit** are public HTTP endpoints
- Directories starting with **`_`** are private — never reachable via HTTP
- Private functions with `index.ts` can be enqueued as background jobs
- Directories without `index.ts` are import-only utilities

## Writing a Function

Every function must export a default async handler:

```typescript
// functions/hello/index.ts

export default async function handler(
  req: Request,
  backd: BackdClient
): Promise<Response> {
  return new Response(
    JSON.stringify({ message: "hello" }),
    { headers: { "Content-Type": "application/json" } }
  );
}
```

### Parameters

- **`req`** — standard [Request](https://developer.mozilla.org/en-US/docs/Web/API/Request) object with method, headers, body
- **`backd`** — pre-authenticated backd client with the app's secret key (bypasses RLS)

### Return Value

Return a standard [Response](https://developer.mozilla.org/en-US/docs/Web/API/Response) object. The status code, headers, and body are forwarded to the caller.

## Using the backd Client

The injected `backd` client has full database access via the secret key:

```typescript
export default async function handler(req: Request, backd: BackdClient) {
  // Read data (bypasses RLS)
  const { data: orders } = await backd.from('orders')
    .where({ status: { $eq: 'pending' } })
    .list()

  // Insert data
  const invoice = await backd.from('invoices').insert({
    order_id: orders[0].id,
    total: orders[0].total,
    status: 'draft',
  })

  // Read secrets
  const stripeKey = await backd.secrets.get('STRIPE_KEY')

  // Enqueue a background job
  await backd.jobs.enqueue('_send_email', {
    to: orders[0].user_email,
    template: 'invoice_created',
    invoice_id: invoice.id,
  })

  // Access user metadata
  const user = await backd.auth.getUser(orders[0].user_id)
  await backd.auth.setAppMeta(user.id, 'last_invoice', invoice.id)

  return new Response(JSON.stringify({ invoice_id: invoice.id }), {
    headers: { "Content-Type": "application/json" },
  })
}
```

## Calling Functions

### From the Browser (SDK)

```typescript
import { createClient } from 'backd-js'

const backd = createClient({ /* ... */ })

const result = await backd.functions.call('send_invoice', {
  order_id: 'abc123',
})
// result = { invoice_id: "..." }
```

### Via HTTP

```bash
curl -X POST http://localhost:8081/v1/my_app/send_invoice \
  -H "Content-Type: application/json" \
  -H "X-Publishable-Key: <key>" \
  -H "X-Session: <token>" \
  -d '{"order_id": "abc123"}'
```

### Private Functions

Calling a private function (`_` prefix) from the browser returns `404 FUNCTION_NOT_FOUND`:

```typescript
// This fails with 404 — private functions are not HTTP-accessible
await backd.functions.call('_send_email', { to: 'test@example.com' })
```

Private functions can only be invoked as background jobs from other Deno functions.

## Shared Code

Use a `_shared/` directory for cross-function utilities:

```typescript
// functions/_shared/stripe.ts
export function createStripeClient(apiKey: string) {
  // ...
}

// functions/process_payment/index.ts
import { createStripeClient } from '../_shared/stripe.ts'

export default async function handler(req: Request, backd: BackdClient) {
  const stripeKey = await backd.secrets.get('STRIPE_KEY')
  const stripe = createStripeClient(stripeKey)
  // ...
}
```

Import paths are relative to the function directory. `../` navigates to the `functions/` root.

## Error Handling

Return appropriate HTTP status codes for errors:

```typescript
export default async function handler(req: Request, backd: BackdClient) {
  const body = await req.json()

  if (!body.order_id) {
    return new Response(
      JSON.stringify({ error: "order_id is required" }),
      { status: 400, headers: { "Content-Type": "application/json" } }
    )
  }

  try {
    const result = await processOrder(body.order_id, backd)
    return new Response(JSON.stringify(result), {
      headers: { "Content-Type": "application/json" },
    })
  } catch (err) {
    return new Response(
      JSON.stringify({ error: "Processing failed", detail: err.message }),
      { status: 500, headers: { "Content-Type": "application/json" } }
    )
  }
}
```

## Execution Model

1. The functions service maintains a pool of long-lived Deno processes
2. Each invocation spawns a short-lived Web Worker inside a pool process
3. The worker loads the function's `index.ts` via dynamic import
4. The `backd` client is pre-configured with environment variables (secret key, app name, internal URL)
5. After execution, the worker terminates and the pool process is returned to the pool
6. Timeout: 30 seconds by default (configurable per function via `jobs.custom` in `app.yaml`)

## Deno Permissions

Functions run with restricted Deno permissions:

- `--allow-net` — network access (needed for backd API calls)
- `--allow-read=<functions_root>,/tmp/backd` — read access to function files and temp
- `--allow-env` — environment variable access (for backd client config)
- `--allow-write=/tmp/backd` — write to temp directory only
- **No `--allow-all`** — explicit permission model

## Next Steps

- [Background Jobs]({{< ref "/guides/jobs" >}}) — enqueue and schedule async work
- [Secrets]({{< ref "/guides/secrets" >}}) — access encrypted secrets from functions
- [SDK Reference]({{< ref "/sdk" >}}) — full SDK documentation
