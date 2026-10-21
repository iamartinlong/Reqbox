/**
 * localStorage helpers that survive the apimock -> reqbox rename.
 *
 * The rename changed every stored key. Without a fallback, anyone upgrading
 * loses their theme, language and send settings, and would reasonably conclude
 * the app forgot them.
 *
 * Both helpers copy the old value forward on first read, so the fallback is
 * consulted once per machine rather than on every load, and a user who later
 * runs only the new binary keeps their settings.
 */

import { createJSONStorage, type PersistStorage, type StorageValue } from 'zustand/middleware'

/** Maps a current key to the name it had before the rename. */
function legacyKey(key: string): string | null {
  if (key.startsWith('reqbox-')) {
    return 'apimock-' + key.slice('reqbox-'.length)
  }
  return null
}

/** Reads a plain string key, falling back to its pre-rename name. */
export function readMigrated(key: string): string | null {
  if (typeof window === 'undefined') return null
  const current = window.localStorage.getItem(key)
  if (current !== null) return current

  const legacy = legacyKey(key)
  if (!legacy) return null
  const old = window.localStorage.getItem(legacy)
  if (old === null) return null

  window.localStorage.setItem(key, old)
  return old
}

/**
 * A zustand persist backend that reads the pre-rename key on first load.
 *
 * zustand stores a whole JSON value, so this cannot reuse readMigrated; it needs
 * the PersistStorage shape instead. Only `getItem` is special - writes always go
 * to the new key, so the fallback is consulted once and then never again.
 */
export function migratedStorage<S>(): PersistStorage<S> {
  const base = createJSONStorage<S>(() => localStorage)!
  return {
    getItem: (name) => {
      // localStorage is synchronous, so createJSONStorage returns plain values
      // rather than promises; the cast records that, since the type allows both.
      const current = base.getItem(name) as StorageValue<S> | null
      if (current !== null) return current

      const legacy = legacyKey(name)
      if (!legacy) return null
      const old = base.getItem(legacy) as StorageValue<S> | null
      if (old !== null) {
        // Copy forward so the fallback stops being consulted.
        base.setItem(name, old)
      }
      return old
    },
    setItem: (name, value) => base.setItem(name, value),
    removeItem: (name) => base.removeItem(name),
  }
}
