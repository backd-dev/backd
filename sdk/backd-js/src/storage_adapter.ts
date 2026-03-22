import type { StorageAdapter } from './types.ts'

export type { StorageAdapter }

export function memoryStorage(): StorageAdapter {
  const store = new Map<string, string>()
  return {
    get(key: string): string | null {
      return store.get(key) ?? null
    },
    set(key: string, value: string): void {
      store.set(key, value)
    },
    delete(key: string): void {
      store.delete(key)
    },
  }
}

export function localStorageAdapter(): StorageAdapter {
  return {
    get(key: string): string | null {
      return localStorage.getItem(key)
    },
    set(key: string, value: string): void {
      localStorage.setItem(key, value)
    },
    delete(key: string): void {
      localStorage.removeItem(key)
    },
  }
}
