import { useCallback, useEffect, useRef, useState } from 'react'
import { FileTextIcon, MenuIcon, TerminalIcon } from 'lucide-react'
import { cn } from 'cn'
import { api } from '@/lib/api'
import { useWorkspace } from '@/lib/store'
import { toast } from 'sonner'
import { Sidebar } from '@/components/sidebar'
import { RequestBar } from '@/components/request-bar'
import { RequestPanel } from '@/components/request-panel'
import { ResponsePanel } from '@/components/response-panel'
import { EnvironmentDialog } from '@/components/environment-dialog'
import { CurlDialog } from '@/components/curl-dialog'
import { useConfirm } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { RunSettingsDialog } from '@/components/run-settings-dialog'
import type { RunResponse } from '@/lib/types'

export function Workspace() {
  const {
    collections,
    requests,
    activeEnvId,
    draft,
    dirty,
    options,
    setWorkspace,
    setHistory,
    startDraft,
    select,
    setDraft,
  } = useWorkspace()

  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)
  const [response, setResponse] = useState<RunResponse | null>(null)
  const [envOpen, setEnvOpen] = useState(false)
  const [curlOpen, setCurlOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [navOpen, setNavOpen] = useState(false)
  const { confirm, dialog: confirmDialog } = useConfirm()
  const responseRef = useRef<HTMLDivElement>(null)
  const responseOwner = useRef<string>('')

  const reload = useCallback(async () => {
    const [cols, reqs, envRes, hist] = await Promise.all([
      api.collections(),
      api.requests(),
      api.environments(),
      api.history('', 100),
    ])
    setWorkspace({
      collections: cols,
      requests: reqs,
      environments: envRes.items,
      activeEnvId: envRes.active,
    })
    setHistory(hist)
    return reqs
  }, [setWorkspace, setHistory])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        await reload()
      } catch {
        if (!cancelled) toast.error('Could not load the workspace')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [reload])

  // Open the most recent request on first load so the app never starts blank.
  useEffect(() => {
    if (!loading && !draft && requests.length > 0) {
      select(requests[0].id)
    }
  }, [loading, draft, requests, select])

  // A response belongs to the request that produced it. Switching requests
  // clears the pane rather than showing another request's answer.
  const draftKey = draft?.id || (draft ? '__draft__' : '')
  useEffect(() => {
    if (response && responseOwner.current !== draftKey) {
      setResponse(null)
    }
  }, [draftKey, response])

  // The sidebar can change the selected request behind the editor's back (for
  // example publishing it, or duplicating another one). Pull the saved version
  // in whenever the open request changed on the server and has no local edits,
  // so the editor never shows stale state.
  useEffect(() => {
    if (!draft?.id || dirty) return
    const fresh = requests.find((r) => r.id === draft.id)
    if (!fresh) return
    if (fresh.updatedAt !== draft.updatedAt) {
      setDraft(fresh)
    }
  }, [requests, draft?.id, draft?.updatedAt, dirty, setDraft])

  const save = useCallback(async () => {
    if (!draft || !draft.url.trim()) {
      toast.error('Add a URL before saving')
      return
    }
    try {
      if (draft.id) {
        await api.updateRequest(draft)
        toast.success('Request saved')
      } else {
        const created = await api.createRequest(draft)
        toast.success('Request created')
        const list = await reload()
        // Keep the response that prompted the save: it belongs to this request
        // before and after the id is assigned.
        responseOwner.current = created.id
        select(list.find((r) => r.id === created.id)?.id ?? created.id)
        return
      }
      const list = await reload()
      const found = list.find((r) => r.id === draft.id)
      if (found) setDraft(found)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Could not save')
    }
  }, [draft, reload, select, setDraft])

  const send = useCallback(async () => {
    if (!draft || !draft.url.trim() || running) return
    setRunning(true)
    // Remember which request this run belongs to. The user may switch requests
    // while it is in flight; the result must not land on the new one.
    const runFor = draft.id || '__draft__'
    responseOwner.current = runFor
    try {
      const res = await api.execute({
        request: draft,
        environmentId: activeEnvId,
        options: {
          timeoutSeconds: options.timeoutSeconds,
          followRedirects: options.followRedirects,
          verifyTls: options.verifyTls,
        },
      })
      if (useWorkspace.getState().draft?.id === draft.id || runFor === '__draft__') {
        setResponse(res)
      }
      if (res.error) {
        toast.error(res.error)
      } else {
        toast.success(`${res.status} ${res.statusText} · ${res.durationMs} ms`)
      }
      const hist = await api.history('', 100)
      setHistory(hist)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Request failed')
    } finally {
      setRunning(false)
      requestAnimationFrame(() => {
        responseRef.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
      })
    }
  }, [draft, running, activeEnvId, options, setHistory])

  // Keyboard: send on mod+enter, save on mod+s, new on mod+n.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey
      if (!mod) return
      if (e.key === 'Enter') {
        e.preventDefault()
        void send()
      } else if (e.key.toLowerCase() === 's') {
        e.preventDefault()
        void save()
      } else if (e.key.toLowerCase() === 'n') {
        e.preventDefault()
        startDraft()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [send, save, startDraft])

  const newRequest = () => {
    startDraft()
    setResponse(null)
  }

  return (
    <div className="relative flex min-h-0 flex-1">
      {/* On phones the rail slides over the work area instead of squeezing it. */}
      {navOpen && (
        <button
          type="button"
          aria-label="Close request list"
          onClick={() => setNavOpen(false)}
          className="absolute inset-0 z-30 bg-black/40 backdrop-blur-[2px] md:hidden"
        />
      )}
      <div
        className={cn(
          'z-40 h-full max-md:absolute max-md:top-0 max-md:left-0 max-md:transition-transform',
          navOpen ? 'max-md:translate-x-0' : 'max-md:-translate-x-full',
        )}
      >
        <Sidebar
          onOpenEnvironments={() => setEnvOpen(true)}
          onOpenSettings={() => setSettingsOpen(true)}
          onNavigate={() => setNavOpen(false)}
        />
      </div>

      <main className="flex min-w-0 flex-1 flex-col">
        <div className="hairline-b flex h-8 shrink-0 items-center gap-2 px-3 md:hidden">
          <Button variant="ghost" size="icon-xs" onClick={() => setNavOpen(true)} aria-label="Open request list">
            <MenuIcon className="size-4" />
          </Button>
          <span className="mono truncate text-[11px] text-muted-foreground">
            {draft?.name || 'Reqbox'}
          </span>
        </div>
        {loading ? (
          <div className="flex flex-1 items-center justify-center">
            <span className="mono text-[11px] tracking-widest text-muted-foreground">
              LOADING WORKSPACE…
            </span>
          </div>
        ) : !draft ? (
          <EmptyState onNew={newRequest} />
        ) : (
          <>
            <RequestBar onSend={() => void send()} onSave={() => void save()} running={running} canSave={!!draft.url.trim()} />

            <div className="hairline-b flex items-center gap-1 px-4 pb-1.5">
              <Button variant="ghost" size="xs" onClick={() => setCurlOpen(true)} className="text-muted-foreground">
                <TerminalIcon className="size-3.5" /> cURL
              </Button>
              {draft.id && (
                <Button
                  variant="ghost"
                  size="xs"
                  onClick={async () => {
                    if (
                      !(await confirm({
                        title: `Delete "${draft.name}"?`,
                        description: 'Past runs of this request stay in History.',
                        confirmLabel: 'Delete request',
                      }))
                    )
                      return
                    await api.deleteRequest(draft.id)
                    const list = await reload()
                    startDraft()
                    setResponse(null)
                    if (list.length > 0) select(list[0].id)
                    toast.success('Request deleted')
                  }}
                  className="text-muted-foreground hover:text-bad"
                >
                  Delete
                </Button>
              )}
              <span className="mono ml-auto text-[10px] text-muted-foreground/60">
                {collections.find((c) => c.id === draft.collectionId)?.name ?? 'unsorted'}
                {dirty && draft.id ? ' · edited' : ''}
              </span>
            </div>

            <div className="flex min-h-0 flex-1 flex-col xl:flex-row">
              <section className="flex min-h-0 min-w-0 shrink-0 flex-col border-border xl:flex-1 xl:shrink xl:border-r">
                <RequestPanel />
              </section>
              <section
                ref={responseRef}
                className="flex min-h-0 min-w-0 flex-1 flex-col border-t border-border bg-panel/40 xl:border-t-0"
              >
                <ResponsePanel response={response} running={running} />
              </section>
            </div>
          </>
        )}
      </main>

      <EnvironmentDialog open={envOpen} onOpenChange={setEnvOpen} />
      <CurlDialog open={curlOpen} onOpenChange={setCurlOpen} />
      <RunSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      {confirmDialog}
    </div>
  )
}

function EmptyState({ onNew }: { onNew: () => void }) {
  return (
    <div className="flex flex-1 items-center justify-center px-6">
      <div className="max-w-md">
        <p className="eyebrow mb-2">Workspace empty</p>
        <h1 className="text-2xl leading-tight font-semibold tracking-tight">
          Point it at an API,
          <br />
          keep what answers.
        </h1>
        <p className="mt-3 text-[13px] leading-relaxed text-muted-foreground">
          Save a request once and it stays here: URL, headers, body, auth. Send it from the
          browser, read the response next to the definition, and publish any saved request as a
          shareable mock URL.
        </p>
        <div className="mt-5 flex items-center gap-2">
          <Button onClick={onNew}>
            <FileTextIcon /> New request
          </Button>
          <span className="mono text-[10.5px] text-muted-foreground">or press ⌘N</span>
        </div>
        <dl className="mt-8 grid grid-cols-3 gap-4 border-t border-border pt-4">
          {[
            ['Requests', 'saved in SQLite'],
            ['History', 'every run kept'],
            ['Mocks', 'public by slug'],
          ].map(([title, sub]) => (
            <div key={title}>
              <dt className="text-[12px] font-medium">{title}</dt>
              <dd className="text-[10.5px] text-muted-foreground">{sub}</dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  )
}
