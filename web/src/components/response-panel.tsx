import {
  AlertTriangleIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  CheckIcon,
  CopyIcon,
  DownloadIcon,
  ImageOffIcon,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { cn } from 'cn'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CodeEditor } from '@/components/code-editor'
import { formatBytes, formatDuration, prettyJSON, statusTint } from '@/lib/format'
import type { RunResponse, Timing } from '@/lib/types'
import { useT } from '@/lib/i18n'

type Props = {
  response: RunResponse | null
  running: boolean
}

export function ResponsePanel({ response, running }: Props) {
  const t = useT()
  const [tab, setTab] = useState('preview')
  const pretty = useMemo(() => {
    if (!response || response.encoding === 'base64') return response?.body ?? ''
    return isJson(response) ? prettyJSON(response.body) : response.body
  }, [response])

  if (running && !response) {
    return (
      <div className="flex flex-1 flex-col">
        <div className="flex h-9 items-center gap-2 border-b border-border px-4">
          <span className="mono text-[11px] text-muted-foreground">{t('resp.waiting')}</span>
        </div>
        <div className="flex-1 space-y-2 p-4">
          {[92, 78, 84, 60, 70].map((w, i) => (
            <div
              key={i}
              className="h-2.5 animate-pulse rounded bg-muted/70"
              style={{ width: `${w}%`, animationDelay: `${i * 70}ms` }}
            />
          ))}
        </div>
      </div>
    )
  }

  if (!response) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        <div className="mono text-[11px] tracking-[0.2em] text-muted-foreground/50">{t('resp.noResponse')}</div>
        <p className="max-w-[28ch] text-xs text-muted-foreground">
          {t('resp.noResponseHint')}
        </p>
        <p className="mono mt-2 text-[10px] text-muted-foreground/50">⌘ / Ctrl + Enter</p>
      </div>
    )
  }

  const headers = response.headers ?? {}
  const headerCount = Object.keys(headers).length
  const requestBodyCount = new Blob([response.requestBody ?? '']).size
  const isImage = response.contentType?.startsWith('image/')
  const isBinary = response.encoding === 'base64' && !isImage
  const bytes = response.sizeBytes

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* status strip */}
      <div className="flex h-9 shrink-0 items-center gap-2 border-b border-border px-4">
        {response.error ? (
          <>
            <AlertTriangleIcon className="size-3.5 text-bad" />
            <span className="mono text-[11.5px] font-semibold text-bad">{t('resp.requestFailed')}</span>
            <span className="truncate text-[11px] text-muted-foreground">{response.error}</span>
          </>
        ) : (
          <>
            <span className={cn('mono text-[12px] font-bold', statusTint(response.status))}>
              {response.status}
            </span>
            <span className="text-[11.5px] text-muted-foreground">{response.statusText}</span>
            <span className="text-border">·</span>
            <span className="mono text-[11px] text-muted-foreground">
              {formatDuration(response.durationMs)}
            </span>
            <span className="text-border">·</span>
            <span className="mono text-[11px] text-muted-foreground">{formatBytes(bytes)}</span>
            {response.envName && (
              <>
                <span className="text-border">·</span>
                <span className="text-[10.5px] text-muted-foreground">{t('sidebar.envPrefix')} {response.envName}</span>
              </>
            )}
            {response.truncated && (
              <span className="mono ml-1 rounded bg-warn/15 px-1 text-[10px] text-warn">
                {t('resp.truncated')}
              </span>
            )}
          </>
        )}
        <div className="ml-auto flex items-center gap-1">
          {response.historyId && (
            <Button variant="ghost" size="xs" asChild className="text-muted-foreground">
              <a href={`/api/history/${response.historyId}/body`} download>
                <DownloadIcon className="size-3.5" />
                {t('resp.download')}
              </a>
            </Button>
          )}
          <CopyButton value={pretty} />
        </div>
      </div>

      <Tabs value={tab} onValueChange={setTab} className="flex min-h-0 flex-1 flex-col">
        <div className="px-4">
          <TabsList
            variant="line"
            className="h-8 w-full justify-start gap-4 border-b border-border/60 p-0 *:px-0"
          >
            {/*
              Named after the browser's Network panel so each tab means exactly
              one thing: Headers is the overview (URL, status, both header
              sets), Request Body is what was actually sent, Preview is the
              formatted response, Response is the raw bytes as received.
            */}
            <TabsTrigger value="headers" className="flex-none text-[12px]">
              {t('tabs.headers')}
              <span className="mono ml-1 text-[10px] text-muted-foreground">{headerCount}</span>
            </TabsTrigger>
            <TabsTrigger value="request" className="flex-none text-[12px]">
              {t('tabs.requestBody')}
              {requestBodyCount > 0 && (
                <span className="mono ml-1 text-[10px] text-muted-foreground">
                  {formatBytes(requestBodyCount)}
                </span>
              )}
            </TabsTrigger>
            <TabsTrigger value="preview" className="flex-none text-[12px]">
              {t('tabs.preview')}
            </TabsTrigger>
            <TabsTrigger value="response" className="flex-none text-[12px]">
              {t('tabs.response')}
            </TabsTrigger>
          </TabsList>
        </div>

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <TabsContent value="preview" className="mt-0">
            {response.error ? (
              <div className="rounded-md border border-bad/30 bg-bad/5 px-3 py-2.5">
                <p className="mono text-[11px] text-bad">{response.error}</p>
                <p className="mt-1.5 text-[10.5px] text-muted-foreground">
                  {t('resp.checkUrl')}
                </p>
              </div>
            ) : isImage ? (
              <div className="flex flex-col items-start gap-2">
                <img
                  src={`data:${response.contentType};base64,${response.body}`}
                  alt={t('resp.imageAlt')}
                  className="max-h-[28rem] rounded border border-border"
                />
                <span className="mono text-[10px] text-muted-foreground">
                  {response.contentType} · {formatBytes(bytes)}
                </span>
              </div>
            ) : isBinary ? (
              <div className="flex flex-col items-center gap-2 rounded-md border border-dashed border-border/70 py-10 text-center">
                <ImageOffIcon className="size-5 text-muted-foreground/60" />
                <p className="text-xs text-muted-foreground">
                  {t('resp.binaryResponse', { type: response.contentType || t('resp.unknownType') })}
                </p>
                <Button variant="outline" size="sm" asChild>
                  <a href={`/api/history/${response.historyId}/body`} download>
                    <DownloadIcon /> {t('resp.download')} {formatBytes(bytes)}
                  </a>
                </Button>
              </div>
            ) : (
              <CodeEditor
                value={pretty}
                readOnly
                language={isJson(response) ? 'json' : 'text'}
                minHeight="14rem"
                ariaLabel={t('resp.aria.body')}
              />
            )}
          </TabsContent>

          <TabsContent value="headers" className="mt-0">
            <HeaderSections response={response} />
          </TabsContent>

          <TabsContent value="request" className="mt-0">
            {response.requestBody ? (
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <span className="eyebrow">{t('resp.sentAfterSubstitution')}</span>
                  <span className="mono text-[10px] text-muted-foreground">
                    {response.requestBodyType} · {formatBytes(requestBodyCount)}
                  </span>
                </div>
                <CodeEditor
                  value={formatRequestBody(response)}
                  readOnly
                  language={response.requestBodyType === 'json' ? 'json' : 'text'}
                  minHeight="12rem"
                  ariaLabel={t('resp.aria.requestSent')}
                />
              </div>
            ) : (
              <div className="flex flex-col items-center gap-1 rounded-md border border-dashed border-border/70 py-10 text-center">
                <p className="text-xs text-muted-foreground">
                  {t('resp.noRequestBody')}
                </p>
                <p className="mono text-[10px] text-muted-foreground/70">
                  {t('resp.bodyType')}: {response.requestBodyType || t('common.none')}
                </p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="response" className="mt-0">
            {response.error ? (
              <div className="rounded-md border border-bad/30 bg-bad/5 px-3 py-2.5">
                <p className="mono text-[11px] text-bad">{response.error}</p>
                <p className="mt-1.5 text-[10.5px] text-muted-foreground">
                  {t('resp.noResponseReceived')}
                </p>
              </div>
            ) : isBinary ? (
              <div className="flex flex-col items-start gap-2">
                <p className="text-[11px] text-muted-foreground">
                  {t('resp.binaryHint', { type: response.contentType || t('resp.unknownType') })}
                </p>
                <CodeEditor
                  value={response.body}
                  readOnly
                  language="text"
                  minHeight="12rem"
                  ariaLabel={t('resp.aria.raw')}
                />
              </div>
            ) : (
              <div className="flex flex-col gap-2">
                <span className="eyebrow">
                  {t('resp.rawHint')}
                </span>
                <CodeEditor
                  value={response.body}
                  readOnly
                  language="text"
                  minHeight="12rem"
                  ariaLabel={t('resp.aria.raw')}
                />
              </div>
            )}
          </TabsContent>
        </div>
      </Tabs>
    </div>
  )
}

/** Pretty-print a JSON request body; leave anything else untouched. */
function formatRequestBody(response: RunResponse): string {
  const raw = response.requestBody ?? ''
  if (response.requestBodyType !== 'json') return raw
  return prettyJSON(raw)
}

function isJson(response: RunResponse): boolean {
  if (response.encoding === 'base64') return false
  const ct = response.contentType ?? ''
  if (ct.includes('json')) return true
  const t = response.body.trim()
  return (t.startsWith('{') && t.endsWith('}')) || (t.startsWith('[') && t.endsWith(']'))
}

/** One collapsible group of name/value rows, like a DevTools header section. */
function HeaderGroup({
  title,
  count,
  defaultOpen = true,
  children,
}: {
  title: string
  count: number
  defaultOpen?: boolean
  children: React.ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="mb-1">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center gap-1.5 rounded px-1 py-1 text-left hover:bg-accent/50"
      >
        {open ? (
          <ChevronDownIcon className="size-3 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronRightIcon className="size-3 shrink-0 text-muted-foreground" />
        )}
        <span className="text-[11.5px] font-medium">{title}</span>
        <span className="mono text-[10px] text-muted-foreground">{count}</span>
      </button>
      {open && <div className="mt-0.5 border-l border-border/60 pl-3.5">{children}</div>}
    </div>
  )
}

function HeaderRows({ headers }: { headers: Record<string, string[]> }) {
  const t = useT()
  const entries = Object.entries(headers).sort(([a], [b]) => a.localeCompare(b))
  if (entries.length === 0) {
    return <p className="py-2 text-[11px] text-muted-foreground">{t('resp.noneInGroup')}</p>
  }
  return (
    <div className="divide-y divide-border/40">
      {entries.map(([key, values]) => (
        <div
          key={key}
          className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.7fr)] gap-3 py-[3px]"
        >
          <span className="mono truncate text-[11px] text-muted-foreground" title={key}>
            {key}
          </span>
          <span className="mono break-all text-[11px]" title={values.join(', ')}>
            {values.join(', ')}
          </span>
        </div>
      ))}
    </div>
  )
}

/**
 * Request/response details laid out like a browser's Network panel: a summary
 * block, then the response headers and the request headers as separate groups.
 *
 * The request side matters most: Reqbox fills in Content-Type, Accept,
 * User-Agent and Authorization itself, so until now there was no way to confirm
 * what the target actually received.
 */
function HeaderSections({ response }: { response: RunResponse }) {
  const t = useT()
  const responseHeaders = response.headers ?? {}
  const requestHeaders = response.requestHeaders ?? {}

  const url = response.requestUrl || response.resolvedUrl || ''
  const redirected = response.finalUrl && url && response.finalUrl !== url

  // Method is not on the response object (it lives on the request editor), so
  // the summary block covers only what this payload can prove.
  const summary: Array<{ label: string; value: React.ReactNode }> = [
    {
      label: t('resp.requestUrl'),
      value: url ? <span className="mono break-all">{url}</span> : '—',
    },
  ]
  if (response.remoteAddr) {
    summary.push({
      label: t('resp.remoteAddress'),
      value: <span className="mono">{response.remoteAddr}</span>,
    })
  }
  if (response.httpVersion) {
    summary.push({
      label: t('resp.httpVersion'),
      value: <span className="mono">{response.httpVersion}</span>,
    })
  }
  summary.push({
    label: t('resp.status'),
    value: response.error ? (
      <span className="text-bad">{t('common.failed')}</span>
    ) : (
      <span className={cn('mono font-semibold', statusTint(response.status))}>
        {response.status} {response.statusText}
      </span>
    ),
  })
  summary.push({
    label: t('resp.duration'),
    value: <span className="mono">{formatDuration(response.durationMs)}</span>,
  })
  summary.push({
    label: t('resp.size'),
    value: <span className="mono">{formatBytes(response.sizeBytes)}</span>,
  })
  if (redirected) {
    summary.push({
      label: t('resp.finalUrl'),
      value: <span className="mono break-all text-muted-foreground">{response.finalUrl}</span>,
    })
  }
  if (response.receivedAt) {
    summary.push({
      label: t('resp.received'),
      value: (
        <span className="mono">{new Date(response.receivedAt).toLocaleString()}</span>
      ),
    })
  }

  return (
    <div className="flex flex-col">
      <HeaderGroup title={t('resp.general')} count={0}>
        <div className="divide-y divide-border/40">
          {summary.map(({ label, value }) => (
            <div
              key={label}
              className="grid grid-cols-[minmax(0,0.8fr)_minmax(0,2fr)] gap-3 py-[3px]"
            >
              <span className="text-[11px] text-muted-foreground">{label}</span>
              <span className="text-[11px]">{value}</span>
            </div>
          ))}
        </div>
      </HeaderGroup>

      <HeaderGroup title={t('resp.responseHeaders')} count={Object.keys(responseHeaders).length}>
        <HeaderRows headers={responseHeaders} />
      </HeaderGroup>

      <HeaderGroup title={t('resp.requestHeaders')} count={Object.keys(requestHeaders).length}>
        <HeaderRows headers={requestHeaders} />
      </HeaderGroup>

      {response.cookies?.length > 0 && (
        <HeaderGroup title={t('resp.cookiesSet')} count={response.cookies.length}>
          <div className="flex flex-col gap-0.5">
            {response.cookies.map((c) => (
              <code key={c} className="mono break-all text-[11px] text-muted-foreground">
                {c}
              </code>
            ))}
          </div>
        </HeaderGroup>
      )}

      {response.timing && (
        <HeaderGroup title={t('resp.timing')} count={0}>
          <TimingBreakdown timing={response.timing} total={response.durationMs} />
        </HeaderGroup>
      )}
    </div>
  )
}

/**
 * Per-phase timing, drawn as proportional bars so the dominant phase is obvious
 * at a glance. TTFB already includes DNS/connect/TLS, so the phases are shown
 * in sequence rather than as additive slices of the total.
 */
function TimingBreakdown({ timing, total }: { timing: Timing; total: number }) {
  const t = useT()
  const rows: Array<{ label: string; ms: number; note?: string }> = [
    { label: t('resp.timing.dns'), ms: timing.dnsMs },
    {
      label: t('resp.timing.connect'),
      ms: timing.connectMs,
      note: timing.reused ? t('resp.timing.reused') : undefined,
    },
    { label: t('resp.timing.tls'), ms: timing.tlsMs },
    { label: t('resp.timing.ttfb'), ms: timing.ttfbMs, note: t('resp.timing.includes') },
    { label: t('resp.timing.download'), ms: timing.downloadMs },
  ]

  const scale = Math.max(total, 1)

  return (
    <div className="flex flex-col gap-1">
      {rows.map((row) => {
        const pct = Math.min(100, (row.ms / scale) * 100)
        const idle = row.ms === 0
        return (
          <div key={row.label} className="grid grid-cols-[7.5rem_minmax(0,1fr)_3.5rem] items-center gap-2">
            <span className="truncate text-[11px] text-muted-foreground" title={row.label}>
              {row.label}
            </span>
            <div className="h-1.5 overflow-hidden rounded-full bg-muted/60">
              {!idle && (
                <div
                  className="h-full rounded-full bg-foreground/40"
                  style={{ width: `${Math.max(pct, 1)}%` }}
                />
              )}
            </div>
            <span className="mono text-right text-[10.5px] text-muted-foreground">
              {idle ? '—' : `${row.ms} ms`}
            </span>
          </div>
        )
      })}
      {rows.some((r) => r.note) && (
        <p className="mono mt-0.5 text-[10px] text-muted-foreground/70">
          {[...new Set(rows.map((r) => r.note).filter(Boolean))].join(' · ')}
        </p>
      )}
    </div>
  )
}

function CopyButton({ value }: { value: string }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="xs" className="text-muted-foreground">
          <CopyIcon className="size-3.5" />
          {t('common.copy')}
          <ChevronDownIcon className="size-3" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuItem
          onClick={() => {
            void navigator.clipboard.writeText(value)
            setCopied(true)
            setTimeout(() => setCopied(false), 1200)
          }}
        >
          {copied ? <CheckIcon className="text-ok" /> : <CopyIcon />}
          {t('tabs.body')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
