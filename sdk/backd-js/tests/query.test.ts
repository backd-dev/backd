import { describe, it, expect, vi, beforeEach } from 'vitest'
import { QueryBuilder } from '../src/query.ts'
import { HttpClient } from '../src/http.ts'
import { memoryStorage } from '../src/storage_adapter.ts'

function mockHttp(): HttpClient {
  const http = new HttpClient('pk_test', memoryStorage())
  return http
}

describe('QueryBuilder', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('builds params from chain methods', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: [], count: 0 }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    await qb
      .select('id', 'status')
      .where({ status: { $eq: 'pending' } })
      .order('created_at', 'desc')
      .limit(20)
      .offset(10)

    const [url] = mockFetch.mock.calls[0]
    expect(url).toContain('/orders')
    expect(url).toContain('select=id%2Cstatus')
    expect(url).toContain('limit=20')
    expect(url).toContain('offset=10')
    expect(url).toContain('order=created_at%3Adesc')
    expect(url).toContain('where=')
  })

  it('get() calls GET /{table}/{id}', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: { id: 'abc', name: 'test' } }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    const result = await qb.get('abc')

    const [url, init] = mockFetch.mock.calls[0]
    expect(url).toBe('http://api/orders/abc')
    expect(init.method).toBe('GET')
    expect(result).toEqual({ id: 'abc', name: 'test' })
  })

  it('insert() calls POST /{table}', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: { id: 'new1' } }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    const result = await qb.insert({ total: 99.99 })

    const [url, init] = mockFetch.mock.calls[0]
    expect(url).toBe('http://api/orders')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ total: 99.99 })
    expect(result).toEqual({ id: 'new1' })
  })

  it('update() calls PUT /{table}/{id}', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: { id: 'abc' } }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    await qb.update('abc', { status: 'shipped' })

    const [url, init] = mockFetch.mock.calls[0]
    expect(url).toBe('http://api/orders/abc')
    expect(init.method).toBe('PUT')
  })

  it('patch() calls PATCH /{table}/{id}', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: { id: 'abc' } }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    await qb.patch('abc', { status: 'shipped' })

    const [url, init] = mockFetch.mock.calls[0]
    expect(url).toBe('http://api/orders/abc')
    expect(init.method).toBe('PATCH')
  })

  it('delete() calls DELETE /{table}/{id}', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    await qb.delete('abc')

    const [url, init] = mockFetch.mock.calls[0]
    expect(url).toBe('http://api/orders/abc')
    expect(init.method).toBe('DELETE')
  })

  it('insertMany() calls insert sequentially', async () => {
    let callCount = 0
    const mockFetch = vi.fn().mockImplementation(() => {
      callCount++
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: { id: `item${callCount}` } }),
      })
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    const results = await qb.insertMany([{ total: 10 }, { total: 20 }])

    expect(results).toHaveLength(2)
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })

  it('is thenable — await returns ListResult', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: [{ id: '1' }], count: 42 }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const qb = new QueryBuilder(mockHttp(), 'http://api', 'orders')
    const result = await qb.limit(10)

    expect(result).toEqual({ data: [{ id: '1' }], count: 42 })
  })
})
