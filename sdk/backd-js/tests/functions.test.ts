import { describe, it, expect } from 'vitest'
import { FunctionError } from '../src/errors.ts'
import { FunctionsModule } from '../src/functions.ts'
import { HttpClient } from '../src/http.ts'
import { memoryStorage } from '../src/storage_adapter.ts'

function makeHttp(): HttpClient {
  return new HttpClient('pk_test', memoryStorage())
}

describe('FunctionsModule', () => {
  it('rejects _-prefixed function names before HTTP', async () => {
    const fns = new FunctionsModule(makeHttp(), 'http://localhost:8081/v1/app')
    await expect(fns.call('_send_email', {})).rejects.toBeInstanceOf(FunctionError)
    await expect(fns.call('_send_email', {})).rejects.toMatchObject({
      code: 'FUNCTION_NOT_FOUND',
      status: 404,
    })
  })

  it('rejects any underscore prefix', async () => {
    const fns = new FunctionsModule(makeHttp(), 'http://localhost:8081/v1/app')
    await expect(fns.call('__double', {})).rejects.toBeInstanceOf(FunctionError)
  })

  it('allows non-prefixed function names', async () => {
    // This will fail with NetworkError since no server is running,
    // but it should NOT throw FunctionError
    const fns = new FunctionsModule(makeHttp(), 'http://localhost:1/v1/app')
    await expect(fns.call('send_invoice', {})).rejects.not.toBeInstanceOf(FunctionError)
  })
})
