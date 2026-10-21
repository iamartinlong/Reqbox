import { useCallback, useEffect, useState } from 'react'
import { readMigrated } from './storage'

type Theme = 'dark' | 'light'

const STORAGE_KEY = 'reqbox-theme'

function readInitial(): Theme {
  if (typeof window === 'undefined') return 'dark'
  const stored = readMigrated(STORAGE_KEY)
  if (stored === 'dark' || stored === 'light') return stored
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(readInitial)

  useEffect(() => {
    const root = document.documentElement
    root.classList.toggle('dark', theme === 'dark')
    root.style.colorScheme = theme
    window.localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  const toggle = useCallback(() => {
    setTheme((t) => (t === 'dark' ? 'light' : 'dark'))
  }, [])

  return { theme, setTheme, toggle }
}
