import { beforeEach, describe, expect, it } from 'vitest'
import { act, renderHook } from '@testing-library/react'

import { en } from './en'
import { zh } from './zh'
import { useLanguageStore, useT, translate, tNow } from './index'

beforeEach(() => {
  // The store persists to localStorage; reset both so tests do not depend on
  // the order they run in.
  localStorage.clear()
  useLanguageStore.setState({ language: 'en' })
})

describe('language store', () => {
  it('starts in English', () => {
    expect(useLanguageStore.getState().language).toBe('en')
  })

  it('switches and toggles', () => {
    act(() => useLanguageStore.getState().setLanguage('zh'))
    expect(useLanguageStore.getState().language).toBe('zh')
    act(() => useLanguageStore.getState().toggleLanguage())
    expect(useLanguageStore.getState().language).toBe('en')
  })

  it('persists the choice under an reqbox key', () => {
    act(() => useLanguageStore.getState().setLanguage('zh'))
    expect(localStorage.getItem('reqbox-language')).toContain('zh')
  })
})

describe('useT', () => {
  it('re-renders consumers with the new language', () => {
    const { result } = renderHook(() => useT())
    expect(result.current('common.cancel')).toBe('Cancel')

    act(() => useLanguageStore.getState().setLanguage('zh'))
    expect(result.current('common.cancel')).toBe('取消')
  })
})

describe('tNow', () => {
  it('reads the current language outside React', () => {
    act(() => useLanguageStore.getState().setLanguage('zh'))
    expect(tNow('common.save')).toBe('保存')
    act(() => useLanguageStore.getState().setLanguage('en'))
    expect(tNow('common.save')).toBe('Save')
  })
})

describe('dictionary shape', () => {
  it('keeps Chinese and English key-for-key aligned', () => {
    expect(Object.keys(zh).sort()).toEqual(Object.keys(en).sort())
  })

  it('translates the shell chrome', () => {
    expect(translate('zh', 'app.online')).toBe('服务在线')
    expect(translate('zh', 'sidebar.requests')).toBe('请求')
  })
})
