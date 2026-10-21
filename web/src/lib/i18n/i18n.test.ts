import { describe, expect, it } from 'vitest'

import { en } from './en'
import { zh } from './zh'
import { detectLanguage, translate } from './index'

const enKeys = Object.keys(en) as Array<keyof typeof en>
const zhKeys = Object.keys(zh) as Array<keyof typeof zh>

describe('dictionaries', () => {
  it('define exactly the same keys', () => {
    // A key present in English but missing in Chinese would silently render the
    // English string in the Chinese UI, which is worse than a visible gap.
    expect([...zhKeys].sort()).toEqual([...enKeys].sort())
  })

  it('have no empty translations', () => {
    for (const key of enKeys) {
      expect(en[key].trim(), `en:${key}`).not.toBe('')
      expect(zh[key].trim(), `zh:${key}`).not.toBe('')
    }
  })

  it('use the same interpolation placeholders in both languages', () => {
    // `{count}` dropped from a translation would render a sentence with a
    // missing number; a stray placeholder would render literal braces.
    const vars = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort()
    for (const key of enKeys) {
      expect(vars(zh[key]), `zh:${key}`).toEqual(vars(en[key]))
    }
  })
})

describe('translate', () => {
  it('returns the string for the requested language', () => {
    expect(translate('en', 'common.cancel')).toBe('Cancel')
    expect(translate('zh', 'common.cancel')).toBe('取消')
  })

  it('interpolates named variables', () => {
    expect(translate('en', 'confirm.deleteRequest.title', { name: 'Login' })).toBe(
      'Delete "Login"?',
    )
    expect(translate('zh', 'confirm.deleteRequest.title', { name: '登录' })).toBe(
      '删除“登录”？',
    )
  })

  it('leaves an unknown placeholder untouched rather than printing undefined', () => {
    expect(translate('en', 'confirm.deleteRequest.title', {})).toBe('Delete "{name}"?')
  })

  it('translates numbers as well as strings', () => {
    expect(translate('en', 'toast.imported', { count: 3 })).toBe('Imported 3 requests')
  })
})

describe('detectLanguage', () => {
  it('maps a zh prefix to Chinese', () => {
    const original = navigator.language
    Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
    expect(detectLanguage()).toBe('zh')
    Object.defineProperty(navigator, 'language', { value: 'zh-TW', configurable: true })
    expect(detectLanguage()).toBe('zh')
    Object.defineProperty(navigator, 'language', { value: original, configurable: true })
  })

  it('falls back to English for other languages', () => {
    const original = navigator.language
    Object.defineProperty(navigator, 'language', { value: 'de-DE', configurable: true })
    expect(detectLanguage()).toBe('en')
    Object.defineProperty(navigator, 'language', { value: original, configurable: true })
  })
})

/**
 * Every `t('key')` written in a component must exist in the dictionary. Without
 * this, a key that was renamed in `en.ts` leaves a raw identifier on screen, and
 * nothing else would catch it.
 *
 * Files are pulled in with Vite's raw glob rather than `node:fs`, so the suite
 * needs no Node type definitions and works the same in CI and locally.
 */
describe('component usage', () => {
  // Only the surfaces that render copy. Globbing the whole `src` would also
  // pick up the dictionaries themselves, whose English values are the source of
  // truth, not leftovers.
  const files = import.meta.glob(['../../components/**/*.tsx', '../../App.tsx'], {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>

  const relevant = Object.entries(files).filter(
    ([path]) => !path.includes('.test.') && !path.includes('/components/ui/'),
  )

  it('finds the component sources', () => {
    // A broken glob would make the assertions below vacuous.
    expect(relevant.length).toBeGreaterThan(10)
  })

  it('only references keys that exist', () => {
    const missing: string[] = []
    for (const [path, text] of relevant) {
      for (const m of text.matchAll(/\bt(?:Now)?\(\s*'([a-z][a-zA-Z0-9.]*)'/g)) {
        const key = m[1]
        if (!(key in en)) missing.push(`${path}: ${key}`)
      }
    }
    expect(missing).toEqual([])
  })

  it('has no leftover hardcoded English in the main surfaces', () => {
    // A cheap smoke test for the strings that were converted. It checks a few
    // distinctive phrases rather than every word, so it does not become a
    // maintenance burden that blocks unrelated copy changes.
    const suspicious = [
      'Publish as a mock endpoint',
      'No saved requests yet.',
      'Nothing matches that filter.',
      'Clear all history?',
      'Replace this workspace?',
      'Sent after variable substitution',
      'Waiting (TTFB)',
      'No credentials are attached',
    ]
    const offenders: string[] = []
    for (const [path, text] of relevant) {
      for (const phrase of suspicious) {
        if (text.includes(phrase)) offenders.push(`${path}: ${phrase}`)
      }
    }
    expect(offenders).toEqual([])
  })
})
