import { useEffect, useState } from 'react'
import { MoonIcon, SunIcon } from 'lucide-react'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/lib/theme'
import { Workspace } from '@/components/workspace'

/**
 * ApiMock is one screen: a rail of saved work on the left, and on the right the
 * request you are shaping above the answer you got back. The fixed split is the
 * point — a request and its response are never more than a glance apart.
 */
export default function App() {
  const { theme, toggle } = useTheme()
  const [online, setOnline] = useState(true)

  useEffect(() => {
    let cancelled = false
    const ping = async () => {
      try {
        // A hung server would otherwise leave the indicator green forever.
        const res = await fetch('/api/health', {
          cache: 'no-store',
          signal: AbortSignal.timeout(5000),
        })
        if (!cancelled) setOnline(res.ok)
      } catch {
        if (!cancelled) setOnline(false)
      }
    }
    void ping()
    const id = window.setInterval(ping, 15000)
    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [])

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex h-full flex-col overflow-hidden bg-background">
        <header className="hairline-b flex h-10 shrink-0 items-center gap-3 bg-chrome px-3">
          <div className="flex items-center gap-2">
            <Mark />
            <span className="text-[13px] font-semibold tracking-tight">ApiMock</span>
            <span className="mono rounded border border-border/70 px-1 text-[9.5px] tracking-wider text-muted-foreground">
              LOCAL
            </span>
          </div>
          <div className="ml-auto flex items-center gap-2">
            <span className="flex items-center gap-1.5 text-[10.5px] text-muted-foreground">
              <span
                className={cn(
                  'size-1.5 rounded-full',
                  online ? 'bg-ok' : 'bg-bad',
                )}
              />
              {online ? 'server online' : 'server unreachable'}
            </span>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={toggle}
              aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
            >
              {theme === 'dark' ? <SunIcon className="size-3.5" /> : <MoonIcon className="size-3.5" />}
            </Button>
          </div>
        </header>
        <Workspace />
        <Toaster position="bottom-right" />
      </div>
    </TooltipProvider>
  )
}

function cn(...args: Array<string | false | undefined>) {
  return args.filter(Boolean).join(' ')
}

/** The brand mark: a request arrow leaving a bracket. Drawn, not iconographic. */
function Mark() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" aria-hidden className="text-foreground">
      <rect x="0.75" y="0.75" width="16.5" height="16.5" rx="4" fill="none" stroke="currentColor" strokeOpacity="0.25" />
      <path d="M4.5 9h6.2" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
      <path d="M8.6 6.2 11.6 9l-3 2.8" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="13.4" cy="9" r="1.15" fill="var(--m-get)" />
    </svg>
  )
}
