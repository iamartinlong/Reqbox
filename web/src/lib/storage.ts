/**
 * localStorage helper that survives the apimock -> reqbox rename.
 *
 * Kept minimal on this branch: it predates the shared storage module on main,
 * and only the theme and session keys need migrating here.
 */

/** Reads a plain string key, falling back to its pre-rename name. */
export function readMigrated(key: string): string | null {
  if (typeof window === 'undefined') return null
  const current = window.localStorage.getItem(key)
  if (current !== null) return current
  if (!key.startsWith('reqbox-')) return null

  const legacy = 'apimock-' + key.slice('reqbox-'.length)
  const old = window.localStorage.getItem(legacy)
  if (old === null) return null

  // Copy forward so the fallback is consulted once, not on every load.
  window.localStorage.setItem(key, old)
  return old
}
