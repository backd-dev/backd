export class BackdError extends Error {
  code: string
  detail: string
  status: number

  constructor(code: string, detail: string, status: number) {
    super(`${code}: ${detail}`)
    this.name = 'BackdError'
    this.code = code
    this.detail = detail
    this.status = status
  }
}

export class AuthError extends BackdError {
  constructor(code: string, detail: string, status: number) {
    super(code, detail, status)
    this.name = 'AuthError'
  }
}

export class QueryError extends BackdError {
  table?: string

  constructor(code: string, detail: string, status: number, table?: string) {
    super(code, detail, status)
    this.name = 'QueryError'
    this.table = table
  }
}

export class FunctionError extends BackdError {
  fn?: string

  constructor(code: string, detail: string, status: number, fn?: string) {
    super(code, detail, status)
    this.name = 'FunctionError'
    this.fn = fn
  }
}

export class StorageError extends BackdError {
  constructor(code: string, detail: string, status: number) {
    super(code, detail, status)
    this.name = 'StorageError'
  }
}

export class NetworkError extends BackdError {
  constructor(detail: string) {
    super('NETWORK_ERROR', detail, 0)
    this.name = 'NetworkError'
  }
}

const AUTH_CODES = new Set([
  'UNAUTHORIZED',
  'SESSION_EXPIRED',
  'TOO_MANY_REQUESTS',
])

const FUNCTION_CODES = new Set([
  'FUNCTION_NOT_FOUND',
  'FUNCTION_TIMEOUT',
  'FUNCTION_ERROR',
])

const STORAGE_CODES = new Set([
  'STORAGE_DISABLED',
])

export function parseError(code: string, detail: string, status: number): BackdError {
  if (AUTH_CODES.has(code)) {
    return new AuthError(code, detail, status)
  }
  if (FUNCTION_CODES.has(code)) {
    return new FunctionError(code, detail, status)
  }
  if (STORAGE_CODES.has(code)) {
    return new StorageError(code, detail, status)
  }
  if (status === 403) {
    return new QueryError(code, detail, status)
  }
  if (status === 404) {
    return new QueryError(code, detail, status)
  }
  if (status === 422) {
    return new QueryError(code, detail, status)
  }
  return new BackdError(code, detail, status)
}
