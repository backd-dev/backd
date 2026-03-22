export type XID = string

export interface ClientOptions {
  api: string
  auth: string
  functions: string
  publishableKey: string
  storage?: StorageAdapter
}

export interface DenoClientOptions {
  internalUrl?: string
  app?: string
  secretKey?: string
}

export interface StorageAdapter {
  get(key: string): string | null | Promise<string | null>
  set(key: string, value: string): void | Promise<void>
  delete(key: string): void | Promise<void>
}

export interface RequestOptions {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  baseUrl: string
  path: string
  body?: unknown
  params?: Record<string, string>
  headers?: Record<string, string>
  noAuth?: boolean
}

export interface BackdUser {
  id: XID
  username: string
  meta: Record<string, unknown>
  meta_app: Record<string, unknown>
  created_at: string
}

export interface BackdSession {
  token: string
  expires_at: string
  user: Pick<BackdUser, 'id' | 'username'>
}

export interface ListResult<T> {
  data: T[]
  count: number
}

export interface FileDescriptor {
  id: XID
  filename: string
  mime_type: string
  size: number
  secure: boolean
  url: string
  expires_in?: number
}

export type ScalarValue = string | number | boolean | null
export type FilterValue = { [op: string]: ScalarValue | ScalarValue[] }
export type WhereFilter = { [field: string]: FilterValue } | LogicalFilter

export interface LogicalFilter {
  $and?: WhereFilter[]
  $or?: WhereFilter[]
  $not?: WhereFilter
}

export interface SignInParams {
  username: string
  password: string
  domain?: string
}

export interface SignUpParams {
  username: string
  password: string
  domain?: string
}

export interface UpdateProfileParams {
  username?: string
  password?: string
  oldPassword?: string
}

export interface EnqueueOptions {
  delay?: string
  max_attempts?: number
}

export interface InvokeOptions {
  headers?: Record<string, string>
}

export const SESSION_KEY = 'backd_session'
