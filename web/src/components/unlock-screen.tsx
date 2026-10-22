import { useState } from 'react'
import { KeyRoundIcon, Loader2Icon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { session } from '@/lib/api'

type Props = {
  /** Called after a successful sign-in so the app can load the workspace. */
  onUnlocked: () => void
}

/**
 * Shown when the API answers 401.
 *
 * This branch has no i18n layer, so the strings are inline English. That is a
 * deliberate limit, not an oversight: adding the dictionary here would mean
 * porting the whole translation system, and this branch only accepts changes
 * needed for the app to work on Windows Server 2008 R2.
 *
 * Full screen rather than a dialog: with no token there is no workspace to
 * show, and an empty shell behind a modal reads as a broken app rather than a
 * locked one.
 */
export function UnlockScreen({ onUnlocked }: Props) {
  const [token, setToken] = useState('')
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)

  const submit = async () => {
    const value = token.trim()
    if (!value || busy) return
    setBusy(true)
    setFailed(false)
    try {
      await session.signIn(value)
      onUnlocked()
    } catch {
      // Any failure here is "not accepted": the server deliberately does not
      // distinguish a malformed token from a wrong one.
      setFailed(true)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex h-full items-center justify-center px-6">
      <form
        className="w-full max-w-sm"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        <div className="mb-4 flex items-center gap-2">
          <KeyRoundIcon className="size-4 text-muted-foreground" />
          <h1 className="text-sm font-semibold tracking-tight">This workspace is locked</h1>
        </div>

        <p className="mb-4 text-[12px] leading-relaxed text-muted-foreground">
          The server requires an API token. It was printed once in the terminal when Reqbox
          started, as a URL you can open directly.
        </p>

        <label className="mb-1.5 block eyebrow" htmlFor="unlock-token">
          API token
        </label>
        <Input
          id="unlock-token"
          autoFocus
          autoComplete="off"
          spellCheck={false}
          value={token}
          onChange={(e) => {
            setToken(e.target.value)
            setFailed(false)
          }}
          placeholder="Paste the token"
          className="mono h-8 text-xs"
          aria-invalid={failed}
          aria-describedby={failed ? 'unlock-error' : undefined}
        />

        {failed && (
          <p id="unlock-error" className="mt-1.5 text-[11px] text-bad">
            That token was not accepted.
          </p>
        )}

        <Button type="submit" size="sm" className="mt-3 w-full" disabled={busy || !token.trim()}>
          {busy ? (
            <>
              <Loader2Icon className="size-3.5 animate-spin" />
              Checking…
            </>
          ) : (
            'Unlock'
          )}
        </Button>

        <p className="mt-4 text-[10.5px] leading-relaxed text-muted-foreground/70">
          Lost it? Restart the server with <code className="mono">-token</code> to set a token you
          choose, or with <code className="mono">-no-auth</code> to turn the check off. Both print
          a warning.
        </p>
      </form>
    </div>
  )
}
