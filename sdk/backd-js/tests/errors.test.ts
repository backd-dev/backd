import { describe, it, expect } from 'vitest'
import {
  BackdError,
  AuthError,
  QueryError,
  FunctionError,
  StorageError,
  NetworkError,
  parseError,
} from '../src/errors.ts'

describe('parseError', () => {
  it('maps UNAUTHORIZED to AuthError', () => {
    const err = parseError('UNAUTHORIZED', 'not authorized', 401)
    expect(err).toBeInstanceOf(AuthError)
    expect(err.code).toBe('UNAUTHORIZED')
    expect(err.status).toBe(401)
  })

  it('maps SESSION_EXPIRED to AuthError', () => {
    const err = parseError('SESSION_EXPIRED', 'expired', 401)
    expect(err).toBeInstanceOf(AuthError)
    expect(err.code).toBe('SESSION_EXPIRED')
  })

  it('maps TOO_MANY_REQUESTS to AuthError', () => {
    const err = parseError('TOO_MANY_REQUESTS', 'slow down', 429)
    expect(err).toBeInstanceOf(AuthError)
  })

  it('maps FUNCTION_NOT_FOUND to FunctionError', () => {
    const err = parseError('FUNCTION_NOT_FOUND', 'not found', 404)
    expect(err).toBeInstanceOf(FunctionError)
  })

  it('maps FUNCTION_TIMEOUT to FunctionError', () => {
    const err = parseError('FUNCTION_TIMEOUT', 'timeout', 504)
    expect(err).toBeInstanceOf(FunctionError)
  })

  it('maps FUNCTION_ERROR to FunctionError', () => {
    const err = parseError('FUNCTION_ERROR', 'failed', 500)
    expect(err).toBeInstanceOf(FunctionError)
  })

  it('maps STORAGE_DISABLED to StorageError', () => {
    const err = parseError('STORAGE_DISABLED', 'no storage', 501)
    expect(err).toBeInstanceOf(StorageError)
  })

  it('maps FORBIDDEN (403) to QueryError', () => {
    const err = parseError('FORBIDDEN', 'denied', 403)
    expect(err).toBeInstanceOf(QueryError)
  })

  it('maps NOT_FOUND (404) to QueryError', () => {
    const err = parseError('NOT_FOUND', 'missing', 404)
    expect(err).toBeInstanceOf(QueryError)
  })

  it('maps VALIDATION_ERROR (422) to QueryError', () => {
    const err = parseError('VALIDATION_ERROR', 'invalid', 422)
    expect(err).toBeInstanceOf(QueryError)
  })

  it('maps unknown codes to BackdError', () => {
    const err = parseError('INTERNAL_ERROR', 'oops', 500)
    expect(err).toBeInstanceOf(BackdError)
    expect(err).not.toBeInstanceOf(AuthError)
    expect(err).not.toBeInstanceOf(QueryError)
  })
})

describe('NetworkError', () => {
  it('has code NETWORK_ERROR and status 0', () => {
    const err = new NetworkError('connection failed')
    expect(err.code).toBe('NETWORK_ERROR')
    expect(err.status).toBe(0)
    expect(err.detail).toBe('connection failed')
  })
})

describe('error hierarchy', () => {
  it('AuthError extends BackdError', () => {
    const err = new AuthError('UNAUTHORIZED', 'test', 401)
    expect(err).toBeInstanceOf(BackdError)
    expect(err).toBeInstanceOf(Error)
  })

  it('QueryError has table property', () => {
    const err = new QueryError('NOT_FOUND', 'missing', 404, 'orders')
    expect(err.table).toBe('orders')
  })

  it('FunctionError has fn property', () => {
    const err = new FunctionError('FUNCTION_ERROR', 'failed', 500, 'send_invoice')
    expect(err.fn).toBe('send_invoice')
  })
})
