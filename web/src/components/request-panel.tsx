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
import { useT, type TranslationKey } from '@/lib/i18n'
import type { KV } from '@/lib/types'

const BODY_TYPES: Array<{ value: string; label: TranslationKey }> = [
  { value: 'none', label: 'body.typeNone' },
  { value: 'json', label: 'body.typeJson' },
  { value: 'text', label: 'body.typeText' },
  { value: 'xml', label: 'body.typeXml' },
  { value: 'form', label: 'body.typeForm' },
]

const AUTH_TYPES: Array<{ value: string; label: TranslationKey }> = [
  { value: 'none', label: 'common.none' },
  { value: 'bearer', label: 'auth.bearer' },
  { value: 'basic', label: 'auth.basic' },
  { value: 'apikey', label: 'auth.apikey' },
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
  const t = useT()
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
            {t('tabs.params')}
            {enabledParams > 0 && <Count n={enabledParams} />}
          </TabsTrigger>
          <TabsTrigger value="headers" className="flex-none text-[12px]">
            {t('tabs.headers')}
            {enabledHeaders > 0 && <Count n={enabledHeaders} />}
          </TabsTrigger>
          <TabsTrigger value="body" className="flex-none text-[12px]">
            {t('tabs.body')}
            {hasBody && <span className="ml-1 size-1.5 rounded-full bg-m-put" />}
          </TabsTrigger>
          <TabsTrigger value="auth" className="flex-none text-[12px]">
            {t('tabs.auth')}
            {draft.authType !== 'none' && <span className="ml-1 size-1.5 rounded-full bg-m-post" />}
          </TabsTrigger>
          <TabsTrigger value="publish" className="flex-none text-[12px]">
            {t('tabs.publish')}
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
            keyLabel={t('kv.parameter')}
            valueLabel={t('kv.value')}
            emptyHint={t('kv.addQueryParam')}
          />
          {enabledParams > 0 && (
            <p className="mono mt-3 text-[10.5px] text-muted-foreground/70">
              {t('kv.appendedOnSend')}
            </p>
          )}
        </TabsContent>

        <TabsContent value="headers" className="mt-0">
          <KVEditor
            rows={headers}
            onChange={(rows) => editDraft({ headers: rows })}
            kind="header"
            keyLabel={t('kv.header')}
            valueLabel={t('kv.value')}
            emptyHint={t('kv.addHeader')}
          />
          <p className="mt-3 text-[10.5px] text-muted-foreground/70">
            {t('kv.headerHint')}
          </p>
        </TabsContent>

        <TabsContent value="body" className="mt-0 flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-0.5 rounded-lg bg-muted/60 p-0.5">
              {BODY_TYPES.map((opt) => (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => editDraft({ bodyType: opt.value })}
                  className={cn(
                    'rounded-[6px] px-2 py-1 text-[11.5px] font-medium transition-colors',
                    draft.bodyType === opt.value
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {t(opt.label)}
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
                {t('body.format')}
              </Button>
            )}
          </div>

          {draft.bodyType === 'none' && (
            <div className="flex h-32 items-center justify-center rounded-md border border-dashed border-border/70 text-xs text-muted-foreground">
              {t('body.none')}
            </div>
          )}

          {(draft.bodyType === 'json' || draft.bodyType === 'text' || draft.bodyType === 'xml') && (
            <CodeEditor
              value={draft.body}
              onChange={(value) => editDraft({ body: value })}
              language={draft.bodyType === 'json' ? 'json' : 'text'}
              minHeight="11rem"
              maxHeight="22rem"
              ariaLabel={t('body.aria')}
              placeholder={t(draft.bodyType === 'json' ? 'body.placeholder.json' : 'body.placeholder.raw')}
            />
          )}

          {draft.bodyType === 'form' && (
            <KVEditor
              rows={form}
              onChange={(rows) => editDraft({ form: rows })}
              kind="form"
              keyLabel={t('kv.field')}
              valueLabel={t('kv.value')}
              emptyHint={t('kv.addFormField')}
            />
          )}
        </TabsContent>

        <TabsContent value="auth" className="mt-0">
          <div className="grid max-w-md gap-3">
            <Row label={t('auth.type')}>
              <Select
                value={draft.authType}
                onValueChange={(v) => editDraft({ authType: v })}
              >
                <SelectTrigger size="sm" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {AUTH_TYPES.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {t(opt.label)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Row>

            {draft.authType === 'bearer' && (
              <Row label={t('auth.token')} hint={t('auth.tokenHint')}>
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
                <Row label={t('auth.username')}>
                  <Input
                    value={draft.auth.username ?? ''}
                    onChange={(e) => setAuth({ username: e.target.value })}
                    className="mono text-xs"
                  />
                </Row>
                <Row label={t('auth.password')}>
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
                  <Row label={t('auth.key')}>
                    <Input
                      value={draft.auth.key ?? ''}
                      onChange={(e) => setAuth({ key: e.target.value })}
                      placeholder="X-Api-Key"
                      className="mono text-xs"
                    />
                  </Row>
                  <Row label={t('kv.value')}>
                    <Input
                      value={draft.auth.value ?? ''}
                      onChange={(e) => setAuth({ value: e.target.value })}
                      className="mono text-xs"
                    />
                  </Row>
                </div>
                <Row label={t('auth.addTo')}>
                  <Select
                    value={draft.auth.in || 'header'}
                    onValueChange={(v) => setAuth({ in: v })}
                  >
                    <SelectTrigger size="sm" className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="header">{t('auth.inHeader')}</SelectItem>
                      <SelectItem value="query">{t('auth.inQuery')}</SelectItem>
                    </SelectContent>
                  </Select>
                </Row>
              </div>
            )}

            {draft.authType === 'none' && (
              <p className="flex items-center gap-2 rounded-md border border-dashed border-border/70 px-3 py-6 text-xs text-muted-foreground">
                <LockIcon className="size-3.5" />
                {t('auth.noCredentials')}
              </p>
            )}

            <div className="mt-1 flex items-start gap-2 rounded-md bg-muted/40 px-2.5 py-2">
              <KeyRoundIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
              <p className="text-[10.5px] leading-relaxed text-muted-foreground">
                {t('auth.variableHint')}
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
          {t('run.timeout')}
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
          {t('run.followRedirects')}
        </label>
        <label className="flex cursor-pointer items-center gap-1.5 text-[11px] text-muted-foreground">
          <input
            type="checkbox"
            checked={options.verifyTls}
            onChange={(e) => setOptions({ verifyTls: e.target.checked })}
            className="size-3 accent-[var(--ring)]"
          />
          {t('run.verifyTls')}
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
  const t = useT()
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
        {t('publish.intro')}
      </p>

      <label className="flex items-start gap-2.5 rounded-md border border-border/70 px-3 py-2.5">
        <input
          type="checkbox"
          checked={draft.exposed}
          onChange={(e) => editDraft({ exposed: e.target.checked })}
          className="mt-0.5 size-3.5 accent-[var(--ring)]"
        />
        <span>
          <span className="block text-[12.5px] font-medium">{t('publish.enable')}</span>
          <span className="block text-[10.5px] text-muted-foreground">
            {t('publish.enableHint')}
          </span>
        </span>
      </label>

      <Row label={t('publish.slug')} hint={t('publish.slugHint')}>
        <div className="flex gap-1.5">
          <Input
            value={draft.exposeSlug}
            onChange={(e) => editDraft({ exposeSlug: e.target.value })}
            placeholder="my-endpoint"
            className="mono text-xs"
          />
          <Button variant="outline" size="sm" onClick={suggestSlug} type="button">
            {t('publish.suggest')}
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
            {t('common.copy')}
          </Button>
        </div>
      )}

      <div className="mt-1 rounded-md border border-border/60 bg-muted/30 p-2.5">
        <p className="eyebrow mb-1.5">{t('publish.overrides')}</p>
        <p className="text-[10.5px] leading-relaxed text-muted-foreground">
          {t('publish.overridesHint')}
        </p>
      </div>
    </div>
  )
}
