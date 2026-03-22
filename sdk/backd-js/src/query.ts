import type { WhereFilter, ListResult } from './types.ts'
import { HttpClient } from './http.ts'
import { QueryError } from './errors.ts'

export class QueryBuilder<T = Record<string, unknown>> {
  private http: HttpClient
  private apiUrl: string
  private table: string
  private _select: string[] = []
  private _where?: WhereFilter
  private _order: string[] = []
  private _limit = 50
  private _offset = 0

  constructor(http: HttpClient, apiUrl: string, table: string) {
    this.http = http
    this.apiUrl = apiUrl
    this.table = table
  }

  select(...columns: string[]): QueryBuilder<T> {
    this._select = columns
    return this
  }

  where(filter: WhereFilter): QueryBuilder<T> {
    this._where = filter
    return this
  }

  order(column: string, direction: 'asc' | 'desc' = 'asc'): QueryBuilder<T> {
    this._order.push(`${column}:${direction}`)
    return this
  }

  limit(n: number): QueryBuilder<T> {
    this._limit = n
    return this
  }

  offset(n: number): QueryBuilder<T> {
    this._offset = n
    return this
  }

  private buildParams(): Record<string, string> {
    const params: Record<string, string> = {}
    if (this._select.length > 0) {
      params.select = this._select.join(',')
    }
    if (this._where) {
      params.where = JSON.stringify(this._where)
    }
    if (this._order.length > 0) {
      params.order = this._order.join(',')
    }
    params.limit = String(this._limit)
    params.offset = String(this._offset)
    return params
  }

  async get(id: string): Promise<T> {
    return this.http.request<T>({
      method: 'GET',
      baseUrl: this.apiUrl,
      path: `/${this.table}/${id}`,
    })
  }

  async single(): Promise<T> {
    const result = await this.http.requestWithMeta<T[]>({
      method: 'GET',
      baseUrl: this.apiUrl,
      path: `/${this.table}`,
      params: { ...this.buildParams(), limit: '1' },
    })
    if (!result.data || result.data.length === 0) {
      throw new QueryError('NOT_FOUND', 'No matching record found', 404, this.table)
    }
    return result.data[0]
  }

  async insert(row: Partial<T>): Promise<T> {
    return this.http.request<T>({
      method: 'POST',
      baseUrl: this.apiUrl,
      path: `/${this.table}`,
      body: row,
    })
  }

  async insertMany(rows: Partial<T>[]): Promise<T[]> {
    const results: T[] = []
    for (const row of rows) {
      const result = await this.insert(row)
      results.push(result)
    }
    return results
  }

  async update(id: string, row: Partial<T>): Promise<T> {
    return this.http.request<T>({
      method: 'PUT',
      baseUrl: this.apiUrl,
      path: `/${this.table}/${id}`,
      body: row,
    })
  }

  async patch(id: string, partial: Partial<T>): Promise<T> {
    return this.http.request<T>({
      method: 'PATCH',
      baseUrl: this.apiUrl,
      path: `/${this.table}/${id}`,
      body: partial,
    })
  }

  async delete(id: string): Promise<void> {
    await this.http.request<void>({
      method: 'DELETE',
      baseUrl: this.apiUrl,
      path: `/${this.table}/${id}`,
    })
  }

  then<TResult1 = ListResult<T>, TResult2 = never>(
    onfulfilled?: ((value: ListResult<T>) => TResult1 | PromiseLike<TResult1>) | null,
    onrejected?: ((reason: unknown) => TResult2 | PromiseLike<TResult2>) | null,
  ): Promise<TResult1 | TResult2> {
    const promise = this.http.requestWithMeta<T[]>({
      method: 'GET',
      baseUrl: this.apiUrl,
      path: `/${this.table}`,
      params: this.buildParams(),
    }).then(result => ({
      data: result.data,
      count: result.count,
    }))

    return promise.then(onfulfilled, onrejected)
  }
}
