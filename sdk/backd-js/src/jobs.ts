import type { EnqueueOptions } from './types.ts'
import { HttpClient } from './http.ts'

export class JobsModule {
  private http: HttpClient
  private internalUrl: string
  private app: string

  constructor(http: HttpClient, internalUrl: string, app: string) {
    this.http = http
    this.internalUrl = internalUrl
    this.app = app
  }

  async enqueue(fn: string, payload?: unknown, opts?: EnqueueOptions): Promise<{ id: string }> {
    const body: Record<string, unknown> = {
      app: this.app,
      function: fn,
      input: payload ? JSON.stringify(payload) : '{}',
      trigger: 'sdk',
    }

    if (opts?.delay) {
      const now = new Date()
      const ms = parseDuration(opts.delay)
      body.run_at = new Date(now.getTime() + ms).toISOString()
    }

    if (opts?.max_attempts !== undefined) {
      body.max_attempts = opts.max_attempts
    }

    return this.http.request<{ id: string }>({
      method: 'POST',
      baseUrl: this.internalUrl,
      path: '/internal/jobs',
      body,
      noAuth: true,
    })
  }
}

function parseDuration(s: string): number {
  const match = s.match(/^(\d+)(ms|s|m|h|d)$/)
  if (!match) return 0
  const n = parseInt(match[1], 10)
  switch (match[2]) {
    case 'ms': return n
    case 's': return n * 1000
    case 'm': return n * 60_000
    case 'h': return n * 3_600_000
    case 'd': return n * 86_400_000
    default: return 0
  }
}
