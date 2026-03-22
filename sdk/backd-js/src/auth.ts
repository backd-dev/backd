import type { StorageAdapter, SignInParams, SignUpParams, UpdateProfileParams, BackdUser } from './types.ts'
import { SESSION_KEY } from './types.ts'
import { HttpClient } from './http.ts'

export class AuthModule {
  private http: HttpClient
  private authUrl: string
  private storage: StorageAdapter

  constructor(http: HttpClient, authUrl: string, storage: StorageAdapter) {
    this.http = http
    this.authUrl = authUrl
    this.storage = storage
  }

  async signIn(params: SignInParams): Promise<void> {
    const result = await this.http.request<{ token: string }>({
      method: 'POST',
      baseUrl: this.authUrl,
      path: '/local/login',
      body: params,
      noAuth: true,
    })
    await this.storage.set(SESSION_KEY, result.token)
  }

  async signOut(): Promise<void> {
    const token = await this.storage.get(SESSION_KEY)
    if (token) {
      await this.http.request<void>({
        method: 'POST',
        baseUrl: this.authUrl,
        path: '/logout',
        body: { token },
      })
    }
    await this.storage.delete(SESSION_KEY)
  }

  async signUp(params: SignUpParams): Promise<BackdUser> {
    return this.http.request<BackdUser>({
      method: 'POST',
      baseUrl: this.authUrl,
      path: '/local/register',
      body: params,
      noAuth: true,
    })
  }

  async me(): Promise<BackdUser> {
    return this.http.request<BackdUser>({
      method: 'POST',
      baseUrl: this.authUrl,
      path: '/refresh',
      body: { token: await this.storage.get(SESSION_KEY) },
    })
  }

  async update(params: UpdateProfileParams): Promise<BackdUser> {
    return this.http.request<BackdUser>({
      method: 'PATCH',
      baseUrl: this.authUrl,
      path: '/profile',
      body: params,
    })
  }

  isAuthenticated(): boolean {
    // Sync check — only works with synchronous storage adapters
    const token = (this.storage as { get(key: string): string | null }).get(SESSION_KEY)
    return token !== null && token !== undefined
  }

  token(): string | null {
    // Sync check — only works with synchronous storage adapters
    return (this.storage as { get(key: string): string | null }).get(SESSION_KEY)
  }
}
