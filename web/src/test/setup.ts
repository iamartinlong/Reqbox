import '@testing-library/jest-dom/vitest'
import { afterEach, beforeEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'

// jsdom ships no matchMedia. The theme hook reads it to pick a default, so
// without this every component that renders a CodeEditor throws on mount.
if (!window.matchMedia) {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

// Components read localStorage (theme, run options) at module scope in places,
// so starting every test from a clean slate keeps them independent.
beforeEach(() => {
  window.localStorage.clear()
})

afterEach(() => {
  cleanup()
})
