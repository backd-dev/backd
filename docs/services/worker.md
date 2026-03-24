---
title: Worker Service
weight: 3
---

The worker service processes background jobs from the `_jobs` table. It shares the same binary as the functions service but runs in worker-only mode.

## Starting

```bash
# Worker only (no HTTP function routes)
api functions --config-dir ./apps --mode worker

# Combined (HTTP + worker) — default
api functions --config-dir ./apps --mode both
```

## How the Worker Operates

The worker continuously polls the `_jobs` table for pending work:

```
┌──────────┐     poll      ┌──────────┐     execute    ┌──────────┐
│  _jobs   │ ◄──────────── │  Worker  │ ──────────────► │   Deno   │
│  table   │ ──────────────►│  loop   │ ◄──────────────│  process  │
│          │   claim (SKIP  │          │    result      │          │
│          │    LOCKED)     │          │                │          │
└──────────┘               └──────────┘                └──────────┘
```

1. Worker polls `_jobs` every `BACKD_JOB_POLL_INTERVAL` (default: 1 second)
2. Claims pending jobs using `FOR UPDATE SKIP LOCKED` — prevents double-processing across replicas
3. Updates job status to `running`
4. Executes the function via the Deno process pool
5. On success: marks job as `completed`
6. On failure: increments `attempts`, applies backoff, or marks as `failed`

## Concurrency

Multiple worker replicas can run safely against the same database. The `FOR UPDATE SKIP LOCKED` query ensures each job is claimed by exactly one worker:

```sql
SELECT * FROM _jobs
WHERE status = 'pending' AND run_at <= NOW()
ORDER BY run_at ASC
LIMIT 10
FOR UPDATE SKIP LOCKED
```

## Retry and Backoff

When a job fails, the worker applies exponential backoff:

| Attempt | Wait before retry |
|---|---|
| 1 → 2 | 30 seconds |
| 2 → 3 | 5 minutes |
| 3 → failed | No more retries — `status = 'failed'`, `error` column populated |

The `max_attempts` and schedule are configurable per app or per function in `app.yaml`:

```yaml
jobs:
  max_attempts: 3
  timeout: 15s
  custom:
    _send_email:
      max_attempts: 5
    _generate_report:
      timeout: 1h
      max_attempts: 1
```

## Cron Scheduling

The worker also runs the cron scheduler:

1. On startup, reads cron entries from `app.yaml` for all apps
2. An in-memory scheduler (using `robfig/cron/v3`) evaluates expressions every minute
3. When a schedule fires, a `_jobs` row is inserted with `trigger='cron'`
4. The job is then picked up by the normal worker polling loop

Cron state is fully in-memory — rebuilt from config on every startup. No persistent cron registry exists.

## Scaling

| Strategy | How |
|---|---|
| **Vertical** | Increase `BACKD_DENO_MAX_WORKERS` for more concurrent job execution |
| **Horizontal** | Run multiple worker replicas — `SKIP LOCKED` prevents conflicts |
| **Separation** | Run `--mode worker` separately from `--mode functions` for independent scaling |

## Monitoring

Worker activity is tracked via Prometheus metrics on the metrics server (`:9090`):

| Metric | Description |
|---|---|
| `backd_jobs_enqueued_total` | Jobs added to the queue |
| `backd_jobs_completed_total` | Jobs finished successfully |
| `backd_jobs_failed_total` | Jobs that exhausted all retries |
| `backd_job_duration_seconds` | Job execution time histogram |

## Docker Deployment

The worker uses the same Docker image as the functions service:

```yaml
# docker-compose.yaml
worker:
  build:
    context: .
    dockerfile: Dockerfile.functions
  entrypoint: ["/app/api"]
  command: ["functions", "--config-dir", "/apps", "--mode", "worker"]
```
