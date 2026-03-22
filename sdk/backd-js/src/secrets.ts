import { HttpClient } from './http.ts'

export class SecretsModule {
  private http: HttpClient
  private internalUrl: string
  private app: string

  constructor(http: HttpClient, internalUrl: string, app: string) {
    this.http = http
    this.internalUrl = internalUrl
    this.app = app
  }

  async get(name: string): Promise<string | null> {
    try {
      const result = await this.http.request<{ secret: string }>({
        method: 'POST',
        baseUrl: this.internalUrl,
        path: '/internal/secret',
        body: { app: this.app, name },
        noAuth: true,
      })
      return result.secret ?? null
    } catch {
      return null
    }
  }
}
