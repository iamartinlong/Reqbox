import {
  AlertTriangleIcon,
  ChevronDownIcon,
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
import type { RunResponse } from '@/lib/types'

type Props = {
  response: RunResponse | null
  running: boolean
}

export function ResponsePanel({ response, running }: Props) {
  const [tab, setTab] = useState('body')
  const pretty = useMemo(() => {
    if (!response || response.encoding === 'base64') return response?.body ?? ''
    return isJson(response) ? prettyJSON(response.body) : response.body
  }, [response])

  if (running && !response) {
    return (
      <div className="flex flex-1 flex-col">
        <div className="flex h-9 items-center gap-2 border-b border-border px-4">
          <span className="mono text-[11px] text-muted-foreground">waiting for response…</span>
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
        <div className="mono text-[11px] tracking-[0.2em] text-muted-foreground/50">NO RESPONSE</div>
        <p className="max-w-[28ch] text-xs text-muted-foreground">
          Send the request to see status, headers and body here.
        </p>
        <p className="mono mt-2 text-[10px] text-muted-foreground/50">⌘ / Ctrl + Enter</p>
      </div>
    )
  }

  const headers = response.headers ?? {}
  const headerCount = Object.keys(headers).length
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
            <span className="mono text-[11.5px] font-semibold text-bad">REQUEST FAILED</span>
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
                <span className="text-[10.5px] text-muted-foreground">env {response.envName}</span>
              </>
            )}
            {response.truncated && (
              <span className="mono ml-1 rounded bg-warn/15 px-1 text-[10px] text-warn">
                truncated
              </span>
            )}
          </>
        )}
        <div className="ml-auto flex items-center gap-1">
          {response.historyId && (
            <Button variant="ghost" size="xs" asChild className="text-muted-foreground">
              <a href={`/api/history/${response.historyId}/body`} download>
                <DownloadIcon className="size-3.5" />
                Download
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
            <TabsTrigger value="body" className="flex-none text-[12px]">
              Body
            </TabsTrigger>
            <TabsTrigger value="headers" className="flex-none text-[12px]">
              Headers
              <span className="mono ml-1 text-[10px] text-muted-foreground">{headerCount}</span>
            </TabsTrigger>
            <TabsTrigger value="raw" className="flex-none text-[12px]">
              Raw
            </TabsTrigger>
          </TabsList>
        </div>

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <TabsContent value="body" className="mt-0">
            {response.error ? (
              <div className="rounded-md border border-bad/30 bg-bad/5 px-3 py-2.5">
                <p className="mono text-[11px] text-bad">{response.error}</p>
                <p className="mt-1.5 text-[10.5px] text-muted-foreground">
                  Check the URL, DNS, or the timeout setting for this request.
                </p>
              </div>
            ) : isImage ? (
              <div className="flex flex-col items-start gap-2">
                <img
                  src={`data:${response.contentType};base64,${response.body}`}
                  alt="Response preview"
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
                  Binary response ({response.contentType || 'unknown type'})
                </p>
                <Button variant="outline" size="sm" asChild>
                  <a href={`/api/history/${response.historyId}/body`} download>
                    <DownloadIcon /> Download {formatBytes(bytes)}
                  </a>
                </Button>
              </div>
            ) : (
              <CodeEditor
                value={pretty}
                readOnly
                language={isJson(response) ? 'json' : 'text'}
                minHeight="14rem"
                ariaLabel="Response body"
              />
            )}
          </TabsContent>

          <TabsContent value="headers" className="mt-0">
            <div className="divide-y divide-border/50">
              {Object.entries(headers).map(([key, values]) => (
                <div key={key} className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.6fr)] gap-3 py-1">
                  <span className="mono truncate text-[11px] text-muted-foreground">{key}</span>
                  <span className="mono break-all text-[11px]">{values.join(', ')}</span>
                </div>
              ))}
              {headerCount === 0 && (
                <p className="py-4 text-xs text-muted-foreground">No headers were returned.</p>
              )}
            </div>
          </TabsContent>

          <TabsContent value="raw" className="mt-0 flex flex-col gap-3">
            {response.resolvedUrl && (
              <div>
                <p className="eyebrow mb-1">Resolved URL</p>
                <code className="mono block break-all rounded bg-muted/50 px-2 py-1.5 text-[11px]">
                  {response.resolvedUrl}
                </code>
              </div>
            )}
            {response.cookies?.length > 0 && (
              <div>
                <p className="eyebrow mb-1">Cookies set</p>
                <div className="flex flex-col gap-0.5">
                  {response.cookies.map((c) => (
                    <code key={c} className="mono break-all text-[11px] text-muted-foreground">
                      {c}
                    </code>
                  ))}
                </div>
              </div>
            )}
            <div>
              <p className="eyebrow mb-1">Content-Type</p>
              <code className="mono text-[11px] text-muted-foreground">
                {response.contentType || '—'}
              </code>
            </div>
            <div>
              <p className="eyebrow mb-1">Received</p>
              <code className="mono text-[11px] text-muted-foreground">
                {new Date(response.receivedAt).toLocaleString()}
              </code>
            </div>
          </TabsContent>
        </div>
      </Tabs>
    </div>
  )
}

function isJson(response: RunResponse): boolean {
  if (response.encoding === 'base64') return false
  const ct = response.contentType ?? ''
  if (ct.includes('json')) return true
  const t = response.body.trim()
  return (t.startsWith('{') && t.endsWith('}')) || (t.startsWith('[') && t.endsWith(']'))
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="xs" className="text-muted-foreground">
          <CopyIcon className="size-3.5" />
          Copy
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
          Body
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
