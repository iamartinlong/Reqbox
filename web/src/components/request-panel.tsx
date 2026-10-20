import { GlobeIcon, KeyRoundIcon, LockIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CodeEditor } from '@/components/code-editor'
import { KVEditor } from '@/components/kv-editor'
import { cn } from 'cn'
import { useWorkspace } from '@/lib/store'
import type { KV } from '@/lib/types'

const BODY_TYPES = [
  { value: 'none', label: 'None' },
  { value: 'json', label: 'JSON' },
  { value: 'text', label: 'Text' },
  { value: 'xml', label: 'XML' },
  { value: 'form', label: 'Form' },
]

const AUTH_TYPES = [
  { value: 'none', label: 'None' },
  { value: 'bearer', label: 'Bearer token' },
  { value: 'basic', label: 'Basic auth' },
  { value: 'apikey', label: 'API key' },
]

function Row({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="eyebrow">{label}</span>
      {children}
      {hint && <span className="text-[10.5px] text-muted-foreground/80">{hint}</span>}
    </label>
  )
}

export function RequestPanel() {
  const { draft, editDraft, options, setOptions } = useWorkspace()
  if (!draft) return null

  const params = draft.params as KV[]
  const headers = draft.headers as KV[]
  const form = draft.form as KV[]

  const enabledParams = params.filter((p) => p.enabled && p.key).length
  const enabledHeaders = headers.filter((h) => h.enabled && h.key).length
  const hasBody = draft.bodyType !== 'none'

  const setAuth = (patch: Partial<typeof draft.auth>) =>
    editDraft({ auth: { ...draft.auth, ...patch } })

  return (
    <Tabs defaultValue="params" className="flex min-h-0 flex-col">
      <div className="px-4">
        <TabsList variant="line" className="h-8 w-full justify-start gap-4 border-b border-border/60 p-0 *:px-0">
          <TabsTrigger value="params" className="flex-none text-[12px]">
            Params
            {enabledParams > 0 && <Count n={enabledParams} />}
          </TabsTrigger>
          <TabsTrigger value="headers" className="flex-none text-[12px]">
            Headers
            {enabledHeaders > 0 && <Count n={enabledHeaders} />}
          </TabsTrigger>
          <TabsTrigger value="body" className="flex-none text-[12px]">
            Body
            {hasBody && <span className="ml-1 size-1.5 rounded-full bg-m-put" />}
          </TabsTrigger>
          <TabsTrigger value="auth" className="flex-none text-[12px]">
            Auth
            {draft.authType !== 'none' && <span className="ml-1 size-1.5 rounded-full bg-m-post" />}
          </TabsTrigger>
          <TabsTrigger value="publish" className="flex-none text-[12px]">
            Publish
            {draft.exposed && <span className="ml-1 size-1.5 rounded-full bg-ok" />}
          </TabsTrigger>
        </TabsList>
      </div>

      <div className="min-h-0 max-h-[46vh] flex-1 overflow-y-auto px-4 py-3 xl:max-h-none">
        <TabsContent value="params" className="mt-0">
          <KVEditor
            rows={params}
            onChange={(rows) => editDraft({ params: rows })}
            kind="query"
            keyLabel="Parameter"
            valueLabel="Value"
            emptyHint="Add a query parameter"
          />
          {enabledParams > 0 && (
            <p className="mono mt-3 text-[10.5px] text-muted-foreground/70">
              appended to the URL at send time
            </p>
          )}
        </TabsContent>

        <TabsContent value="headers" className="mt-0">
          <KVEditor
            rows={headers}
            onChange={(rows) => editDraft({ headers: rows })}
            kind="header"
            keyLabel="Header"
            valueLabel="Value"
            emptyHint="Add a header"
          />
          <p className="mt-3 text-[10.5px] text-muted-foreground/70">
            Content-Type, Accept and User-Agent are filled in automatically when left empty.
          </p>
        </TabsContent>

        <TabsContent value="body" className="mt-0 flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-0.5 rounded-lg bg-muted/60 p-0.5">
              {BODY_TYPES.map((t) => (
                <button
                  key={t.value}
                  type="button"
                  onClick={() => editDraft({ bodyType: t.value })}
                  className={cn(
                    'rounded-[6px] px-2 py-1 text-[11.5px] font-medium transition-colors',
                    draft.bodyType === t.value
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {t.label}
                </button>
              ))}
            </div>
            {draft.bodyType === 'json' && (
              <Button
                variant="ghost"
                size="xs"
                className="ml-auto text-muted-foreground"
                onClick={() => {
                  try {
                    editDraft({ body: JSON.stringify(JSON.parse(draft.body), null, 2) })
                  } catch {
                    /* already invalid; nothing to format */
                  }
                }}
              >
                Format
              </Button>
            )}
          </div>

          {draft.bodyType === 'none' && (
            <div className="flex h-32 items-center justify-center rounded-md border border-dashed border-border/70 text-xs text-muted-foreground">
              This request sends no body.
            </div>
          )}

          {(draft.bodyType === 'json' || draft.bodyType === 'text' || draft.bodyType === 'xml') && (
            <CodeEditor
              value={draft.body}
              onChange={(value) => editDraft({ body: value })}
              language={draft.bodyType === 'json' ? 'json' : 'text'}
              minHeight="11rem"
              maxHeight="22rem"
              ariaLabel="Request body"
              placeholder={draft.bodyType === 'json' ? '{\n  "key": "value"\n}' : 'Raw body'}
            />
          )}

          {draft.bodyType === 'form' && (
            <KVEditor
              rows={form}
              onChange={(rows) => editDraft({ form: rows })}
              kind="form"
              keyLabel="Field"
              valueLabel="Value"
              emptyHint="Add a form field"
            />
          )}
        </TabsContent>

        <TabsContent value="auth" className="mt-0">
          <div className="grid max-w-md gap-3">
            <Row label="Type">
              <Select
                value={draft.authType}
                onValueChange={(v) => editDraft({ authType: v })}
              >
                <SelectTrigger size="sm" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {AUTH_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Row>

            {draft.authType === 'bearer' && (
              <Row label="Token" hint="The Bearer prefix is added if you leave it off.">
                <Input
                  value={draft.auth.token ?? ''}
                  onChange={(e) => setAuth({ token: e.target.value })}
                  placeholder="eyJhbGciOi…"
                  className="mono text-xs"
                />
              </Row>
            )}

            {draft.authType === 'basic' && (
              <div className="grid grid-cols-2 gap-2">
                <Row label="Username">
                  <Input
                    value={draft.auth.username ?? ''}
                    onChange={(e) => setAuth({ username: e.target.value })}
                    className="mono text-xs"
                  />
                </Row>
                <Row label="Password">
                  <Input
                    type="password"
                    value={draft.auth.password ?? ''}
                    onChange={(e) => setAuth({ password: e.target.value })}
                    className="mono text-xs"
                  />
                </Row>
              </div>
            )}

            {draft.authType === 'apikey' && (
              <div className="grid gap-2">
                <div className="grid grid-cols-2 gap-2">
                  <Row label="Key">
                    <Input
                      value={draft.auth.key ?? ''}
                      onChange={(e) => setAuth({ key: e.target.value })}
                      placeholder="X-Api-Key"
                      className="mono text-xs"
                    />
                  </Row>
                  <Row label="Value">
                    <Input
                      value={draft.auth.value ?? ''}
                      onChange={(e) => setAuth({ value: e.target.value })}
                      className="mono text-xs"
                    />
                  </Row>
                </div>
                <Row label="Add to">
                  <Select
                    value={draft.auth.in || 'header'}
                    onValueChange={(v) => setAuth({ in: v })}
                  >
                    <SelectTrigger size="sm" className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="header">Header</SelectItem>
                      <SelectItem value="query">Query string</SelectItem>
                    </SelectContent>
                  </Select>
                </Row>
              </div>
            )}

            {draft.authType === 'none' && (
              <p className="flex items-center gap-2 rounded-md border border-dashed border-border/70 px-3 py-6 text-xs text-muted-foreground">
                <LockIcon className="size-3.5" />
                No credentials are attached to this request.
              </p>
            )}

            <div className="mt-1 flex items-start gap-2 rounded-md bg-muted/40 px-2.5 py-2">
              <KeyRoundIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
              <p className="text-[10.5px] leading-relaxed text-muted-foreground">
                Use <code className="mono text-foreground/80">{'{{variable}}'}</code> anywhere in
                the URL, headers or body to pull a value from the active environment.
              </p>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="publish" className="mt-0">
          <PublishPanel />
        </TabsContent>
      </div>

      <div className="hairline-t mt-auto flex items-center gap-3 px-4 py-1.5">
        <label className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
          Timeout
          <Input
            type="number"
            min={1}
            max={600}
            value={options.timeoutSeconds}
            onChange={(e) => setOptions({ timeoutSeconds: Number(e.target.value) || 30 })}
            className="mono h-6 w-14 px-1.5 text-[11px]"
          />
          s
        </label>
        <label className="flex cursor-pointer items-center gap-1.5 text-[11px] text-muted-foreground">
          <input
            type="checkbox"
            checked={options.followRedirects}
            onChange={(e) => setOptions({ followRedirects: e.target.checked })}
            className="size-3 accent-[var(--ring)]"
          />
          Follow redirects
        </label>
        <label className="flex cursor-pointer items-center gap-1.5 text-[11px] text-muted-foreground">
          <input
            type="checkbox"
            checked={options.verifyTls}
            onChange={(e) => setOptions({ verifyTls: e.target.checked })}
            className="size-3 accent-[var(--ring)]"
          />
          Verify TLS
        </label>
        <span className="mono ml-auto text-[10px] text-muted-foreground/60">
          {draft.method} · {draft.bodyType}
        </span>
      </div>
    </Tabs>
  )
}

function Count({ n }: { n: number }) {
  return (
    <span className="mono ml-1 rounded bg-muted px-1 text-[10px] text-muted-foreground">{n}</span>
  )
}

function PublishPanel() {
  const { draft, editDraft } = useWorkspace()
  if (!draft) return null

  const suggestSlug = () => {
    const base =
      draft.name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '') || 'mock'
    editDraft({ exposeSlug: base })
  }

  return (
    <div className="grid max-w-lg gap-3">
      <p className="text-xs leading-relaxed text-muted-foreground">
        Publishing makes this saved request reachable at a stable URL. Anyone who can reach this
        server can call it; the stored URL, headers and auth are applied on their behalf.
      </p>

      <label className="flex items-start gap-2.5 rounded-md border border-border/70 px-3 py-2.5">
        <input
          type="checkbox"
          checked={draft.exposed}
          onChange={(e) => editDraft({ exposed: e.target.checked })}
          className="mt-0.5 size-3.5 accent-[var(--ring)]"
        />
        <span>
          <span className="block text-[12.5px] font-medium">Publish as a mock endpoint</span>
          <span className="block text-[10.5px] text-muted-foreground">
            Requests are executed live, not replayed from cache.
          </span>
        </span>
      </label>

      <Row label="Slug" hint="Letters, numbers, dashes. Must be unique.">
        <div className="flex gap-1.5">
          <Input
            value={draft.exposeSlug}
            onChange={(e) => editDraft({ exposeSlug: e.target.value })}
            placeholder="my-endpoint"
            className="mono text-xs"
          />
          <Button variant="outline" size="sm" onClick={suggestSlug} type="button">
            Suggest
          </Button>
        </div>
      </Row>

      {draft.exposed && draft.exposeSlug && (
        <div className="flex items-center gap-2 rounded-md bg-muted/50 px-2.5 py-2">
          <GlobeIcon className="size-3.5 shrink-0 text-ok" />
          <code className="mono min-w-0 flex-1 truncate text-[11px]">
            {window.location.origin}/mock/{draft.exposeSlug}
          </code>
          <Button
            variant="ghost"
            size="xs"
            onClick={() => {
              void navigator.clipboard.writeText(
                `${window.location.origin}/mock/${draft.exposeSlug}`,
              )
            }}
          >
            Copy
          </Button>
        </div>
      )}

      <div className="mt-1 rounded-md border border-border/60 bg-muted/30 p-2.5">
        <p className="eyebrow mb-1.5">Caller overrides</p>
        <p className="text-[10.5px] leading-relaxed text-muted-foreground">
          A query parameter on the mock URL fills the matching{' '}
          <code className="mono">{'{{name}}'}</code> placeholder in this request. Callers can also
          use a method other than GET to exercise a saved POST.
        </p>
      </div>
    </div>
  )
}
