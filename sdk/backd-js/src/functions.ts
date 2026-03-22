import type { InvokeOptions } from './types.ts'
import { HttpClient } from './http.ts'
import { FunctionError } from './errors.ts'

export class FunctionsModule {
  private http: HttpClient
  private functionsUrl: string

  constructor(http: HttpClient, functionsUrl: string) {
    this.http = http
    this.functionsUrl = functionsUrl
  }

  async call<T = unknown>(name: string, payload?: unknown, opts?: InvokeOptions): Promise<T> {
    if (name.startsWith('_')) {
      throw new FunctionError('FUNCTION_NOT_FOUND', 'function not found', 404, name)
    }

    return this.http.request<T>({
      method: 'POST',
      baseUrl: this.functionsUrl,
      path: `/${name}`,
      body: payload,
      headers: opts?.headers,
    })
  }
}
