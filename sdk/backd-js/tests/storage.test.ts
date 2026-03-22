import { describe, it, expect } from 'vitest'
import { memoryStorage } from '../src/storage_adapter.ts'

describe('memoryStorage', () => {
  it('returns null for missing keys', () => {
    const s = memoryStorage()
    expect(s.get('missing')).toBeNull()
  })

  it('stores and retrieves values', () => {
    const s = memoryStorage()
    s.set('key', 'value')
    expect(s.get('key')).toBe('value')
  })

  it('deletes values', () => {
    const s = memoryStorage()
    s.set('key', 'value')
    s.delete('key')
    expect(s.get('key')).toBeNull()
  })

  it('overwrites existing values', () => {
    const s = memoryStorage()
    s.set('key', 'first')
    s.set('key', 'second')
    expect(s.get('key')).toBe('second')
  })

  it('isolates between instances', () => {
    const s1 = memoryStorage()
    const s2 = memoryStorage()
    s1.set('key', 'from-s1')
    expect(s2.get('key')).toBeNull()
  })
})
