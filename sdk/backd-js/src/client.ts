import type { ClientOptions, StorageAdapter } from './types.ts'
import { HttpClient } from './http.ts'
import { AuthModule } from './auth.ts'
import { QueryBuilder } from './query.ts'
import { FunctionsModule } from './functions.ts'
import { memoryStorage } from './storage_adapter.ts'

export class BackdClient {
  readonly auth: AuthModule
  readonly functions: FunctionsModule

  protected http: HttpClient
  protected apiUrl: string

  constructor(opts: ClientOptions) {
    const storage: StorageAdapter = opts.storage ?? memoryStorage()
    this.http = new HttpClient(opts.publishableKey, storage)
    this.apiUrl = opts.api
    this.auth = new AuthModule(this.http, opts.auth, storage)
    this.functions = new FunctionsModule(this.http, opts.functions)
  }

  from<T = Record<string, unknown>>(table: string): QueryBuilder<T> {
    return new QueryBuilder<T>(this.http, this.apiUrl, table)
  }
}

export function createClient(opts: ClientOptions): BackdClient {
  return new BackdClient(opts)
}
