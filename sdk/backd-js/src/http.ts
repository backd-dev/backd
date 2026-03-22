import type { StorageAdapter, RequestOptions } from './types.ts'
import { SESSION_KEY } from './types.ts'
import { NetworkError, parseError } from './errors.ts'

export class HttpClient {
  private publishableKey: string
  private secretKey?: string
  private storage: StorageAdapter

  constructor(publishableKey: string, storage: StorageAdapter, secretKey?: string) {
    this.publishableKey = publishableKey
    this.storage = storage
    this.secretKey = secretKey
  }

  async request<T>(opts: RequestOptions): Promise<T> {
    const url = this.buildUrl(opts)
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Publishable-Key': this.publishableKey,
      ...opts.headers,
    }

    if (this.secretKey) {
      headers['X-Secret-Key'] = this.secretKey
    }

    if (!opts.noAuth) {
      const token = await this.storage.get(SESSION_KEY)
      if (token) {
        headers['X-Session'] = token
      }
    }

    const init: RequestInit = {
      method: opts.method,
      headers,
    }

    if (opts.body !== undefined) {
      init.body = JSON.stringify(opts.body)
    }

    let response: Response
    try {
      response = await fetch(url, init)
    } catch (err) {
      // Retry once on network error
      try {
        await new Promise(resolve => setTimeout(resolve, 500))
        response = await fetch(url, init)
      } catch {
        throw new NetworkError(err instanceof Error ? err.message : 'Network request failed')
      }
    }

    if (!response.ok) {
      const body = await response.json().catch(() => ({}))
      const code = body.error ?? 'UNKNOWN_ERROR'
      const detail = body.error_detail ?? response.statusText
      throw parseError(code, detail, response.status)
    }

    if (response.status === 204) {
      return undefined as T
    }

    const body = await response.json()
    return body.data !== undefined ? body.data : body
  }

  async requestWithMeta<T>(opts: RequestOptions): Promise<{ data: T; count: number }> {
    const url = this.buildUrl(opts)
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Publishable-Key': this.publishableKey,
      ...opts.headers,
    }

    if (this.secretKey) {
      headers['X-Secret-Key'] = this.secretKey
    }

    if (!opts.noAuth) {
      const token = await this.storage.get(SESSION_KEY)
      if (token) {
        headers['X-Session'] = token
      }
    }

    let response: Response
    try {
      response = await fetch(url, { method: opts.method, headers })
    } catch (err) {
      try {
        await new Promise(resolve => setTimeout(resolve, 500))
        response = await fetch(url, { method: opts.method, headers })
      } catch {
        throw new NetworkError(err instanceof Error ? err.message : 'Network request failed')
      }
    }

    if (!response.ok) {
      const body = await response.json().catch(() => ({}))
      const code = body.error ?? 'UNKNOWN_ERROR'
      const detail = body.error_detail ?? response.statusText
      throw parseError(code, detail, response.status)
    }

    const body = await response.json()
    return {
      data: body.data ?? [],
      count: body.count ?? 0,
    }
  }

  private buildUrl(opts: RequestOptions): string {
    let url = `${opts.baseUrl}${opts.path}`
    const params = new URLSearchParams()

    if (opts.params) {
      for (const [key, value] of Object.entries(opts.params)) {
        params.set(key, value)
      }
    }

    const qs = params.toString()
    if (qs) {
      url += `?${qs}`
    }
    return url
  }
}
