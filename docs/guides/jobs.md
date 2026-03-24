---
title: Background Jobs
weight: 6
---

backd provides a background job system for async work. Jobs are functions invoked asynchronously — the same Deno runtime that handles HTTP requests also processes jobs from the `_jobs` table.

## How Jobs Work

```
HTTP request  →  Deno  →  Worker(index.ts)   synchronous, returns HTTP response
Job from _jobs → Deno  →  Worker(index.ts)   asynchronous, result written to _jobs
Cron trigger  →  Deno  →  Worker(index.ts)   scheduled, result written to _jobs
```

Any function — public or private — can be enqueued as a job. Private functions (`_` prefix) are the natural home for job handlers since they have no HTTP surface.

## Enqueuing Jobs

Jobs can only be enqueued from **Deno functions** (server-side) — never directly from the browser. This prevents clients from flooding the job queue.

### Immediate Job

```typescript
// functions/process_order/index.ts

export default async function handler(req: Request, backd: BackdClient) {
  const order = await req.json()

  // Enqueue immediately — runs as soon as a worker is available
  const jobId = await backd.jobs.enqueue('_send_email', {
    to: order.user_email,
    template: 'order_confirmed',
    order_id: order.id,
  })

  return new Response(JSON.stringify({ job_id: jobId }), {
    headers: { "Content-Type": "application/json" },
  })
}
```

### Delayed Job

```typescript
// Execute after 5 minutes
await backd.jobs.enqueue('_generate_report', {
  report_type: 'daily',
}, { delay: '5m' })
```

### Custom Retry

```typescript
await backd.jobs.enqueue('_process_payment', {
  order_id: 'abc123',
}, { maxAttempts: 5 })
```

## The `_jobs` Table

Jobs are stored in the app's `_jobs` system table:

| Column | Type | Description |
|---|---|---|
| `id` | TEXT | Unique job ID (XID) |
| `app_name` | TEXT | App that owns the job |
| `function` | TEXT | Function directory name (e.g. `_send_email`) |
| `payload` | JSONB | Data passed to the function as request body |
| `trigger` | TEXT | `background`, `cron`, or `sdk` |
| `status` | TEXT | `pending`, `running`, `completed`, `failed` |
| `attempts` | INT | Number of execution attempts |
| `max_attempts` | INT | Maximum retry attempts (default: 3) |
| `error` | TEXT | Last error message (if failed) |
| `run_at` | TIMESTAMPTZ | When to execute (default: now) |
| `started_at` | TIMESTAMPTZ | When execution started |
| `completed_at` | TIMESTAMPTZ | When execution finished |
| `created_at` | TIMESTAMPTZ | When the job was enqueued |

## Job Processing

The worker service polls `_jobs` for pending work using `FOR UPDATE SKIP LOCKED` to prevent multiple workers from claiming the same job:

```sql
SELECT * FROM _jobs
WHERE status = 'pending' AND run_at <= NOW()
ORDER BY run_at ASC
LIMIT 10
FOR UPDATE SKIP LOCKED
```

### Retry with Exponential Backoff

When a job fails:
- Attempt 1 → retry after **30 seconds**
- Attempt 2 → retry after **5 minutes**
- Attempt 3 → status set to `failed`, `error` column populated

The `max_attempts` and backoff schedule can be configured globally in `app.yaml` or per-function:

```yaml
jobs:
  max_attempts: 3       # default for all jobs
  timeout: 15s          # default execution timeout
  custom:
    _send_email:
      max_attempts: 5   # override for this function
    _generate_report:
      timeout: 1h       # long-running job
      max_attempts: 1   # no retry
```

## Cron Jobs

Schedule recurring jobs via `app.yaml`:

```yaml
cron:
  - schedule: "0 3 * * *"        # 3:00 AM daily
    function: _cleanup_expired
  - schedule: "*/10 * * * *"     # every 10 minutes
    function: _process_pending
    payload:                      # optional static payload
      mode: full
```

### Cron Expression Format

Standard 5-field format: `minute hour day month weekday`

| Field | Allowed Values |
|---|---|
| Minute | 0-59 |
| Hour | 0-23 |
| Day of month | 1-31 |
| Month | 1-12 |
| Day of week | 0-6 (Sunday=0) |

Special characters: `*` (any), `,` (list), `-` (range), `/` (step)

Examples:
- `0 3 * * *` — 3:00 AM daily
- `*/10 * * * *` — every 10 minutes
- `0 9 * * 1-5` — 9:00 AM weekdays
- `0 0 1 * *` — midnight on the 1st of each month

### How Cron Works

1. On startup, backd reads cron entries from `app.yaml`
2. An in-memory scheduler evaluates cron expressions each minute
3. When a schedule fires, a row is inserted into `_jobs` with `trigger='cron'`
4. The job worker picks it up and executes the function normally

Cron state is fully in-memory — rebuilt from `app.yaml` on every startup.

## Writing Job Handlers

Job handlers use the same function signature as HTTP handlers. The `req.body` contains the job payload:

```typescript
// functions/_send_email/index.ts

export default async function handler(req: Request, backd: BackdClient) {
  const { to, template, order_id } = await req.json()

  const sendgridKey = await backd.secrets.get('SENDGRID_KEY')

  // Fetch order data
  const order = await backd.from('orders').get(order_id)

  // Send email using external service
  const response = await fetch('https://api.sendgrid.com/v3/mail/send', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${sendgridKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      to, template, data: { order },
    }),
  })

  if (!response.ok) {
    // Throwing an error marks the job as failed and triggers retry
    throw new Error(`Email send failed: ${response.status}`)
  }

  return new Response(JSON.stringify({ sent: true }), {
    headers: { "Content-Type": "application/json" },
  })
}
```

If the function throws an error or returns a non-2xx status, the job is marked as failed and retried according to the backoff schedule.
