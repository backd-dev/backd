import { describe, it, expect, vi, beforeEach } from 'vitest'
import { HttpClient } from '../src/http.ts'
import { memoryStorage } from '../src/storage_adapter.ts'
import { NetworkError, AuthError } from '../src/errors.ts'

describe('HttpClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('attaches X-Publishable-Key header', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: { id: '1' } }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const http = new HttpClient('pk_test_123', memoryStorage())
    await http.request({ method: 'GET', baseUrl: 'http://api', path: '/test' })

    expect(mockFetch).toHaveBeenCalledOnce()
    const [, init] = mockFetch.mock.calls[0]
    expect(init.headers['X-Publishable-Key']).toBe('pk_test_123')
  })

  it('attaches X-Session when token is stored', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: {} }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const storage = memoryStorage()
    storage.set('backd_session', 'tok_abc')
    const http = new HttpClient('pk_test', storage)
    await http.request({ method: 'GET', baseUrl: 'http://api', path: '/test' })

    const [, init] = mockFetch.mock.calls[0]
    expect(init.headers['X-Session']).toBe('tok_abc')
  })

  it('does not attach X-Session when noAuth is true', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: {} }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const storage = memoryStorage()
    storage.set('backd_session', 'tok_abc')
    const http = new HttpClient('pk_test', storage)
    await http.request({ method: 'GET', baseUrl: 'http://api', path: '/test', noAuth: true })

    const [, init] = mockFetch.mock.calls[0]
    expect(init.headers['X-Session']).toBeUndefined()
  })

  it('attaches X-Secret-Key when provided', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: {} }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const http = new HttpClient('pk_test', memoryStorage(), 'sk_secret')
    await http.request({ method: 'GET', baseUrl: 'http://api', path: '/test' })

    const [, init] = mockFetch.mock.calls[0]
    expect(init.headers['X-Secret-Key']).toBe('sk_secret')
  })

  it('URL-encodes query params', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const http = new HttpClient('pk_test', memoryStorage())
    await http.request({
      method: 'GET',
      baseUrl: 'http://api',
      path: '/orders',
      params: { where: '{"status":{"$eq":"pending"}}' },
    })

    const [url] = mockFetch.mock.calls[0]
    expect(url).toContain('where=')
    expect(url).toContain(encodeURIComponent('{"status":{"$eq":"pending"}}'))
  })

  it('parses error response and throws typed error', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
      json: () => Promise.resolve({ error: 'UNAUTHORIZED', error_detail: 'invalid token' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const http = new HttpClient('pk_test', memoryStorage())
    await expect(
      http.request({ method: 'GET', baseUrl: 'http://api', path: '/test' }),
    ).rejects.toBeInstanceOf(AuthError)
  })

  it('returns undefined for 204 No Content', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
    })
    vi.stubGlobal('fetch', mockFetch)

    const http = new HttpClient('pk_test', memoryStorage())
    const result = await http.request({ method: 'DELETE', baseUrl: 'http://api', path: '/test' })
    expect(result).toBeUndefined()
  })

  it('retries once on network error', async () => {
    let callCount = 0
    const mockFetch = vi.fn().mockImplementation(() => {
      callCount++
      if (callCount === 1) throw new Error('connection refused')
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: { ok: true } }),
      })
    })
    vi.stubGlobal('fetch', mockFetch)

    const http = new HttpClient('pk_test', memoryStorage())
    const result = await http.request({ method: 'GET', baseUrl: 'http://api', path: '/test' })
    expect(result).toEqual({ ok: true })
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })

  it('throws NetworkError when both attempts fail', async () => {
    const mockFetch = vi.fn().mockRejectedValue(new Error('connection refused'))
    vi.stubGlobal('fetch', mockFetch)

    const http = new HttpClient('pk_test', memoryStorage())
    await expect(
      http.request({ method: 'GET', baseUrl: 'http://api', path: '/test' }),
    ).rejects.toBeInstanceOf(NetworkError)
  })
})
