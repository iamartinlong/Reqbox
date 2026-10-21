import { beforeEach, describe, expect, it } from 'vitest'

import { migratedStorage, readMigrated } from './storage'

beforeEach(() => {
  localStorage.clear()
})

/**
 * The rename from apimock to reqbox changed every storage key. These tests exist
 * because the failure is invisible in a fresh install and only shows up as a
 * returning user's settings silently resetting.
 */
describe('readMigrated', () => {
  it('returns the current key when present', () => {
    localStorage.setItem('reqbox-theme', 'light')
    expect(readMigrated('reqbox-theme')).toBe('light')
  })

  it('falls back to the pre-rename key', () => {
    localStorage.setItem('apimock-theme', 'light')
    expect(readMigrated('reqbox-theme')).toBe('light')
  })

  it('copies the old value forward so the fallback is used once', () => {
    localStorage.setItem('apimock-theme', 'light')
    readMigrated('reqbox-theme')

    expect(localStorage.getItem('reqbox-theme')).toBe('light')
    // Removing the old key must not lose the setting.
    localStorage.removeItem('apimock-theme')
    expect(readMigrated('reqbox-theme')).toBe('light')
  })

  it('prefers the current key when both exist', () => {
    // A user who ran both builds: the new value is the one they last chose.
    localStorage.setItem('apimock-theme', 'light')
    localStorage.setItem('reqbox-theme', 'dark')
    expect(readMigrated('reqbox-theme')).toBe('dark')
  })

  it('returns null when neither key exists', () => {
    expect(readMigrated('reqbox-theme')).toBeNull()
  })

  it('does not invent a legacy name for unrelated keys', () => {
    localStorage.setItem('some-other-key', 'x')
    expect(readMigrated('apimock-theme')).toBeNull()
  })
})

describe('migratedStorage', () => {
  type S = { language: string }

  it('reads state stored under the pre-rename key', () => {
    const legacyValue = JSON.stringify({ state: { language: 'zh' }, version: 0 })
    localStorage.setItem('apimock-language', legacyValue)

    const storage = migratedStorage<S>()
    const got = storage.getItem('reqbox-language') as { state: S } | null

    expect(got?.state.language).toBe('zh')
  })

  it('migrates the value to the new key', () => {
    localStorage.setItem(
      'apimock-language',
      JSON.stringify({ state: { language: 'zh' }, version: 0 }),
    )

    const storage = migratedStorage<S>()
    storage.getItem('reqbox-language')

    expect(localStorage.getItem('reqbox-language')).toContain('zh')
  })

  it('prefers the current key when both exist', () => {
    localStorage.setItem(
      'apimock-language',
      JSON.stringify({ state: { language: 'zh' }, version: 0 }),
    )
    localStorage.setItem(
      'reqbox-language',
      JSON.stringify({ state: { language: 'en' }, version: 0 }),
    )

    const storage = migratedStorage<S>()
    const got = storage.getItem('reqbox-language') as { state: S } | null
    expect(got?.state.language).toBe('en')
  })

  it('returns null when nothing is stored', () => {
    const storage = migratedStorage<S>()
    expect(storage.getItem('reqbox-language')).toBeNull()
  })

  it('writes to the new key only', () => {
    const storage = migratedStorage<S>()
    storage.setItem('reqbox-language', { state: { language: 'zh' }, version: 0 })

    expect(localStorage.getItem('reqbox-language')).toContain('zh')
    expect(localStorage.getItem('apimock-language')).toBeNull()
  })
})
