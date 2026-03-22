import type { ClientOptions, DenoClientOptions, BackdUser, StorageAdapter } from './types.ts'
import { HttpClient } from './http.ts'
import { BackdClient } from './client.ts'
import { AuthModule } from './auth.ts'
import { JobsModule } from './jobs.ts'
import { SecretsModule } from './secrets.ts'
import { memoryStorage } from './storage_adapter.ts'

class DenoAuthModule extends AuthModule {
  private denoHttp: HttpClient
  private denoInternalUrl: string
  private denoApp: string

  constructor(http: HttpClient, authUrl: string, storage: StorageAdapter, internalUrl: string, app: string) {
    super(http, authUrl, storage)
    this.denoHttp = http
    this.denoInternalUrl = internalUrl
    this.denoApp = app
  }

  async setAppMeta(userId: string, key: string, value: unknown): Promise<void> {
    await this.denoHttp.request<void>({
      method: 'POST',
      baseUrl: this.denoInternalUrl,
      path: '/internal/auth',
      body: { app: this.denoApp, action: 'set_app_meta', user_id: userId, key, value },
      noAuth: true,
    })
  }

  async setGlobalMeta(userId: string, key: string, value: unknown): Promise<void> {
    await this.denoHttp.request<void>({
      method: 'POST',
      baseUrl: this.denoInternalUrl,
      path: '/internal/auth',
      body: { app: this.denoApp, action: 'set_global_meta', user_id: userId, key, value },
      noAuth: true,
    })
  }

  async setPassword(userId: string, password: string): Promise<void> {
    await this.denoHttp.request<void>({
      method: 'POST',
      baseUrl: this.denoInternalUrl,
      path: '/internal/auth',
      body: { app: this.denoApp, action: 'set_password', user_id: userId, password },
      noAuth: true,
    })
  }

  async setUsername(userId: string, username: string): Promise<void> {
    await this.denoHttp.request<void>({
      method: 'POST',
      baseUrl: this.denoInternalUrl,
      path: '/internal/auth',
      body: { app: this.denoApp, action: 'set_username', user_id: userId, username },
      noAuth: true,
    })
  }

  async getUser(userId: string): Promise<BackdUser> {
    return this.denoHttp.request<BackdUser>({
      method: 'POST',
      baseUrl: this.denoInternalUrl,
      path: '/internal/auth',
      body: { app: this.denoApp, action: 'get_user', user_id: userId },
      noAuth: true,
    })
  }
}

export class DenoBackdClient extends BackdClient {
  readonly jobs: JobsModule
  readonly secrets: SecretsModule
  override readonly auth: DenoAuthModule

  constructor(opts: ClientOptions, denoOpts: DenoClientOptions) {
    super(opts)
    const internalUrl = denoOpts.internalUrl ?? 'http://127.0.0.1:9191'
    const app = denoOpts.app ?? ''
    const storage = opts.storage ?? memoryStorage()

    if (denoOpts.secretKey) {
      // Recreate HttpClient with secret key for elevated access
      this.http = new HttpClient(opts.publishableKey, storage, denoOpts.secretKey)
    }

    this.auth = new DenoAuthModule(this.http, opts.auth, storage, internalUrl, app)
    this.jobs = new JobsModule(this.http, internalUrl, app)
    this.secrets = new SecretsModule(this.http, internalUrl, app)
  }
}

declare const Deno: {
  env: {
    get(key: string): string | undefined
  }
}

export function createClient(opts?: Partial<ClientOptions> & DenoClientOptions): DenoBackdClient {
  if (!opts || Object.keys(opts).length === 0) {
    // Auto-configure from Deno environment variables
    const app = Deno.env.get('BACKD_APP') ?? ''
    const internalUrl = Deno.env.get('BACKD_INTERNAL_URL') ?? 'http://127.0.0.1:9191'
    const secretKey = Deno.env.get('BACKD_SECRET_KEY') ?? ''

    const clientOpts: ClientOptions = {
      api: `${internalUrl}/v1/${app}`,
      auth: `${internalUrl}/v1/${app}/auth`,
      functions: `${internalUrl}/v1/${app}/functions`,
      publishableKey: secretKey,
    }

    return new DenoBackdClient(clientOpts, { internalUrl, app, secretKey })
  }

  const clientOpts: ClientOptions = {
    api: opts.api ?? '',
    auth: opts.auth ?? '',
    functions: opts.functions ?? '',
    publishableKey: opts.publishableKey ?? '',
    storage: opts.storage,
  }

  return new DenoBackdClient(clientOpts, {
    internalUrl: opts.internalUrl,
    app: opts.app,
    secretKey: opts.secretKey,
  })
}

// Re-export everything from the browser entry
export { BackdError, AuthError, QueryError, FunctionError, StorageError, NetworkError, parseError } from './errors.ts'
export { memoryStorage, localStorageAdapter } from './storage_adapter.ts'
export { QueryBuilder } from './query.ts'

export type {
  ClientOptions,
  DenoClientOptions,
  StorageAdapter,
  BackdUser,
  BackdSession,
  ListResult,
  FileDescriptor,
  WhereFilter,
  LogicalFilter,
  ScalarValue,
  FilterValue,
  SignInParams,
  SignUpParams,
  UpdateProfileParams,
  InvokeOptions,
  EnqueueOptions,
  XID,
} from './types.ts'
