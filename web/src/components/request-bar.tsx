import { ChevronDownIcon, Loader2Icon, SendIcon, SaveIcon, CopyIcon, CheckIcon } from 'lucide-react'
import { useState } from 'react'
import { cn } from 'cn'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { METHODS, methodColor, pathOf } from '@/lib/format'
import { useWorkspace } from '@/lib/store'

type Props = {
  onSend: () => void
  onSave: () => void
  running: boolean
  canSave: boolean
}

/**
 * The request bar. It is the one element that stays put: method, URL, send.
 * Everything else below it is a detail view of the same object.
 */
export function RequestBar({ onSend, onSave, running, canSave }: Props) {
  const { draft, editDraft, dirty } = useWorkspace()
  const [copied, setCopied] = useState(false)

  if (!draft) return null
  const method = (draft.method || 'GET').toUpperCase()

  const copyURL = async () => {
    if (!draft.url) return
    await navigator.clipboard.writeText(draft.url)
    setCopied(true)
    setTimeout(() => setCopied(false), 1400)
  }

  return (
    <div className="flex flex-col gap-2 px-4 pt-3 pb-2.5">
      <div className="flex items-center gap-2">
        <input
          value={draft.name}
          onChange={(e) => editDraft({ name: e.target.value })}
          aria-label="Request name"
          spellCheck={false}
          className="-ml-1.5 h-7 max-w-[280px] min-w-[8ch] truncate rounded-md border border-transparent bg-transparent px-1.5 text-[13px] font-semibold outline-none hover:border-border/60 focus:border-border"
        />
        <span className="mono truncate text-[11px] text-muted-foreground">
          {dirty ? 'unsaved changes' : draft.id ? 'saved' : 'draft'}
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <Button
            variant="outline"
            size="sm"
            onClick={onSave}
            disabled={!canSave}
            title="Save request (Ctrl+S)"
          >
            <SaveIcon /> Save
          </Button>
        </div>
      </div>

      {/*
        The row is a fixed height rather than a stretch-to-tallest flex line.
        Previously the URL input's own content box (18.75px) drove the row
        height, leaving the bar visibly shorter than the flat-bordered SEND
        button beside it. Every child now fills one 34px track.
      */}
      <div className="flex h-[34px] items-stretch gap-0 overflow-hidden rounded-lg border border-border bg-background focus-within:border-ring/60">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              className={cn(
                'mono flex w-[92px] shrink-0 items-center justify-between gap-1 border-r border-border px-2.5 text-[12px] font-bold tracking-wide hover:bg-accent/50',
                methodColor[method] ?? 'text-m-other',
              )}
              aria-label="HTTP method"
            >
              {method}
              <ChevronDownIcon className="size-3 opacity-50" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-36">
            {METHODS.map((m) => (
              <DropdownMenuItem
                key={m}
                onClick={() => editDraft({ method: m })}
                className={cn('mono text-[12px] font-bold', methodColor[m])}
              >
                {m}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <input
          value={draft.url}
          onChange={(e) => editDraft({ url: e.target.value })}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !running) onSend()
          }}
          placeholder="https://api.example.com/v1/resource"
          spellCheck={false}
          aria-label="Request URL"
          className="mono h-full min-w-0 flex-1 bg-transparent px-3 text-[12.5px] leading-none outline-none placeholder:text-muted-foreground/60"
        />

        <button
          type="button"
          onClick={() => void copyURL()}
          className="flex w-8 shrink-0 items-center justify-center border-l border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          aria-label="Copy URL"
        >
          {copied ? <CheckIcon className="size-3.5 text-ok" /> : <CopyIcon className="size-3.5" />}
        </button>

        <Button
          onClick={onSend}
          disabled={running || !draft.url.trim()}
          className="mono h-full w-[104px] shrink-0 rounded-none border-0 border-l border-border text-[12px] font-semibold tracking-wide"
        >
          {running ? (
            <Loader2Icon className="size-3.5 animate-spin" />
          ) : (
            <SendIcon className="size-3.5" />
          )}
          {running ? 'SENDING' : 'SEND'}
        </Button>
      </div>

      {draft.exposed && draft.exposeSlug && (
        <p className="mono flex items-center gap-1.5 text-[10.5px] text-muted-foreground">
          <span className="size-1.5 rounded-full bg-ok" />
          public · {window.location.origin}/mock/{draft.exposeSlug}
        </p>
      )}
      {!draft.exposed && pathOf(draft.url) && (
        <p className="mono truncate text-[10.5px] text-muted-foreground/60">
          {pathOf(draft.url)}
        </p>
      )}
    </div>
  )
}
