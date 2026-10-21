/**
 * Minimal i18n runtime.
 *
 * Kept separate from the workspace store on purpose: workspace state is server
 * data, whereas the UI language is a device preference. Mixing them would mean
 * a language change dirties a request and triggers a save.
 *
 * The language is persisted under `reqbox-language` so it survives a restart,
 * and defaults to the browser's own preference so a Chinese user does not have
 * to switch it on first load.
 */

import { useCallback } from 'react'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { en, type TranslationKey } from './en'
import { zh } from './zh'
import { migratedStorage } from '../storage'

export type Language = 'en' | 'zh'

const dictionaries: Record<Language, Record<TranslationKey, string>> = { en, zh }

/**
 * Strings that are interpolated, e.g. `Delete "{name}"?`. Values are inserted
 * verbatim: no escaping, because they are UI labels rather than markup.
 */
export type Vars = Record<string, string | number>

export function translate(lang: Language, key: TranslationKey, vars?: Vars): string {
  const dict = dictionaries[lang] ?? en
  // Falling back to English keeps an untranslated key readable instead of
  // rendering a raw `sidebar.foo`. The parity test should stop that happening
  // in the first place, but production should degrade gracefully.
  const template = dict[key] ?? en[key] ?? key
  if (!vars) return template
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in vars ? String(vars[name]) : match,
  )
}

/** Only a language prefix is needed; a region such as `zh-TW` still maps to `zh`. */
export function detectLanguage(): Language {
  if (typeof navigator === 'undefined') return 'en'
  return navigator.language?.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

type LanguageState = {
  language: Language
  setLanguage: (lang: Language) => void
  toggleLanguage: () => void
}

export const useLanguageStore = create<LanguageState>()(
  persist(
    (set, get) => ({
      language: detectLanguage(),
      setLanguage: (language) => set({ language }),
      toggleLanguage: () => set({ language: get().language === 'en' ? 'zh' : 'en' }),
    }),
    { name: 'reqbox-language', storage: migratedStorage() },
  ),
)

/**
 * The translator for the current language. Components call this once and pass
 * `t` down, or call it locally; both read the same store, so a language change
 * re-renders every consumer.
 */
export function useT() {
  const language = useLanguageStore((s) => s.language)
  return useCallback(
    (key: TranslationKey, vars?: Vars) => translate(language, key, vars),
    [language],
  )
}

/** Non-hook access for async handlers and toasts, where a hook is unusable. */
export function tNow(key: TranslationKey, vars?: Vars): string {
  return translate(useLanguageStore.getState().language, key, vars)
}

export type { TranslationKey }
