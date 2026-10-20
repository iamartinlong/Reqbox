import {
  ChevronDownIcon,
  ChevronRightIcon,
  ClockIcon,
  CopyIcon,
  DownloadIcon,
  FolderIcon,
  FolderPlusIcon,
  GlobeIcon,
  MoreHorizontalIcon,
  PlusIcon,
  SearchIcon,
  Settings2Icon,
  Trash2Icon,
  UploadIcon,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { cn } from 'cn'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { PromptDialog } from '@/components/prompt-dialog'
import { api } from '@/lib/api'
import { methodTint, pathOf, slugify, timeAgo } from '@/lib/format'
import { useWorkspace } from '@/lib/store'
import type { ApiRequest } from '@/lib/types'
import { toast } from 'sonner'

type Props = {
  onOpenEnvironments: () => void
  onOpenSettings: () => void
  /** Mobile only: called after picking a request so the drawer can close. */
  onNavigate?: () => void
}

export function Sidebar({ onOpenEnvironments, onOpenSettings, onNavigate }: Props) {
  const {
    collections,
    requests,
    history,
    activeEnvId,
    environments,
    selectedId,
    sidebarTab,
    setSidebarTab,
    select,
    startDraft,
    setWorkspace,
  } = useWorkspace()

  const [query, setQuery] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [reloadKey, setReloadKey] = useState(0)
  const fileRef = useRef<HTMLInputElement>(null)

  // One prompt dialog serves collection creation, renaming and mock publishing.
  // window.prompt is not available in every browser context, so it is avoided.
  const [prompt, setPrompt] = useState<null | {
    title: string
    description?: string
    label: string
    value: string
    placeholder?: string
    hint?: string
    confirmLabel: string
    onConfirm: (value: string) => void | Promise<void>
  }>(null)

  const activeEnv = environments.find((e) => e.id === activeEnvId)

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return requests
    return requests.filter(
      (r) =>
        r.name.toLowerCase().includes(q) ||
        r.url.toLowerCase().includes(q) ||
        r.method.toLowerCase().includes(q),
    )
  }, [requests, query])

  const grouped = useMemo(() => {
    const map = new Map<string, ApiRequest[]>()
    for (const r of filtered) {
      const key = r.collectionId ?? '__none__'
      const list = map.get(key) ?? []
      list.push(r)
      map.set(key, list)
    }
    return map
  }, [filtered])

  const refresh = useCallback(async () => {
    const [cols, reqs, envRes] = await Promise.all([
      api.collections(),
      api.requests(),
      api.environments(),
    ])
    setWorkspace({
      collections: cols,
      requests: reqs,
      environments: envRes.items,
      activeEnvId: envRes.active,
    })
  }, [setWorkspace])

  useEffect(() => {
    if (reloadKey > 0) void refresh()
  }, [reloadKey, refresh])

  const createCollection = () => {
    setPrompt({
      title: 'New collection',
      description: 'Collections group saved requests in the sidebar.',
      label: 'Name',
      value: '',
      placeholder: 'Payments',
      confirmLabel: 'Create',
      onConfirm: async (value) => {
        const name = value.trim()
        if (!name) return
        await api.createCollection(name)
        setReloadKey((k) => k + 1)
        toast.success('Collection created')
      },
    })
  }

  const deleteCollection = async (id: string, name: string) => {
    if (!window.confirm(`Delete "${name}"? Requests inside it are kept.`)) return
    await api.deleteCollection(id)
    setReloadKey((k) => k + 1)
    toast.success('Collection deleted')
  }

  const renameCollection = (id: string, current: string) => {
    setPrompt({
      title: 'Rename collection',
      label: 'Name',
      value: current,
      confirmLabel: 'Rename',
      onConfirm: async (value) => {
        const name = value.trim()
        if (!name || name === current) return
        await api.updateCollection(id, { name })
        setReloadKey((k) => k + 1)
        toast.success('Collection renamed')
      },
    })
  }

  const activateEnv = async (id: string) => {
    await api.activateEnvironment(id)
    setReloadKey((k) => k + 1)
  }

  const exportWorkspace = async () => {
    const res = await fetch('/api/export')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'apimock-export.json'
    a.click()
    URL.revokeObjectURL(url)
    toast.success('Workspace exported')
  }

  const importWorkspace = async (file: File) => {
    try {
      const text = await file.text()
      const data = JSON.parse(text)
      const res = await api.importWorkspace(data)
      setReloadKey((k) => k + 1)
      toast.success(`Imported ${res.requests} requests`)
    } catch {
      toast.error('That file is not a valid ApiMock export')
    }
  }

  const groupedCollections = collections.map((c) => ({
    ...c,
    items: grouped.get(c.id) ?? [],
  }))
  const loose = grouped.get('__none__') ?? []

  const deleteRequest = async (r: ApiRequest) => {
    if (!window.confirm(`Delete "${r.name}"? Runs of it stay in history.`)) return
    await api.deleteRequest(r.id)
    setReloadKey((k) => k + 1)
    if (selectedId === r.id) select(null)
    toast.success('Request deleted')
  }

  const duplicateRequest = async (r: ApiRequest) => {
    const copy = await api.duplicateRequest(r.id)
    setReloadKey((k) => k + 1)
    toast.success(`Duplicated as "${copy.name}"`)
  }

  const toggleExposed = async (r: ApiRequest) => {
    await api.updateRequest({ ...r, exposed: !r.exposed })
    setReloadKey((k) => k + 1)
    toast.success(r.exposed ? 'Mock unpublished' : 'Mock published')
  }

  const publishRequest = (r: ApiRequest) => {
    setPrompt({
      title: 'Publish as a mock',
      description:
        'Anyone who can reach this server can call this endpoint. The stored URL, headers and auth are applied for them.',
      label: 'Slug',
      value: r.exposeSlug || slugify(r.name),
      placeholder: 'my-endpoint',
      hint: 'Letters, numbers and dashes. Must be unique.',
      confirmLabel: 'Publish',
      onConfirm: async (value) => {
        const slug = slugify(value)
        if (!slug) {
          toast.error('A slug is required')
          return
        }
        try {
          await api.updateRequest({ ...r, exposeSlug: slug, exposed: true })
          setReloadKey((k) => k + 1)
          toast.success(`Published at /mock/${slug}`)
        } catch (e) {
          toast.error(e instanceof Error ? e.message : 'Could not publish')
        }
      },
    })
  }

  const renderRequest = (r: ApiRequest) => (
    <div
      key={r.id}
      className={cn(
        'group flex items-center rounded-md transition-colors',
        selectedId === r.id ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/60',
      )}
    >
      <button
        type="button"
        onClick={() => {
          select(r.id)
          onNavigate?.()
        }}
        className="flex min-w-0 flex-1 items-center gap-2 py-1.5 pr-1 pl-2 text-left"
      >
        <span
          className={cn(
            'mono shrink-0 rounded px-1 py-[1px] text-[9.5px] font-bold tracking-wide',
            methodTint(r.method),
          )}
        >
          {r.method.slice(0, 4)}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[12.5px] leading-tight text-foreground/90">
            {r.name}
          </span>
          <span className="mono block truncate text-[10.5px] leading-tight text-muted-foreground/80">
            {pathOf(r.url)}
          </span>
        </span>
        {r.exposed && (
          <span
            className="size-1.5 shrink-0 rounded-full bg-ok"
            aria-label="Published as a mock"
          />
        )}
      </button>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={`Actions for ${r.name}`}
            className="mr-1 opacity-0 group-hover:opacity-100 data-[state=open]:opacity-100"
          >
            <MoreHorizontalIcon className="size-3.5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={() => void duplicateRequest(r)}>
            <CopyIcon /> Duplicate
          </DropdownMenuItem>
          {r.exposed ? (
            <DropdownMenuItem onClick={() => void toggleExposed(r)}>
              <GlobeIcon /> Unpublish mock
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => publishRequest(r)}>
              <GlobeIcon /> Publish as mock
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => void deleteRequest(r)}>
            <Trash2Icon /> Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )

  return (
    <aside className="flex h-full w-[290px] shrink-0 flex-col border-r border-border bg-rail">
      {/* workspace switcher */}
      <div className="flex items-center gap-2 px-3 pt-3 pb-2">
        <div className="min-w-0 flex-1">
          <Select value={activeEnvId || '__none__'} onValueChange={(v) => void activateEnv(v === '__none__' ? '' : v)}>
            <SelectTrigger
              size="sm"
              className="h-7 w-full border-transparent bg-transparent px-1.5 text-xs font-medium hover:bg-accent/60"
            >
              <SelectValue placeholder="No environment" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">No environment</SelectItem>
              {environments.map((env) => (
                <SelectItem key={env.id} value={env.id}>
                  {env.name}
                  <span className="mono ml-auto text-[10px] text-muted-foreground">
                    {env.variables.filter((v) => v.enabled).length}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={onOpenEnvironments}
              aria-label="Manage environments"
              className="text-muted-foreground"
            >
              <Settings2Icon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Environments</TooltipContent>
        </Tooltip>
      </div>

      {/* tabs */}
      <div className="mx-3 flex items-center gap-0.5 rounded-lg bg-muted/60 p-0.5">
        <button
          type="button"
          onClick={() => setSidebarTab('requests')}
          className={cn(
            'flex-1 rounded-[6px] py-1 text-[11.5px] font-medium transition-colors',
            sidebarTab === 'requests'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          Requests
          <span className="mono ml-1.5 text-[10px] text-muted-foreground">{requests.length}</span>
        </button>
        <button
          type="button"
          onClick={() => setSidebarTab('history')}
          className={cn(
            'flex-1 rounded-[6px] py-1 text-[11.5px] font-medium transition-colors',
            sidebarTab === 'history'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          History
          <span className="mono ml-1.5 text-[10px] text-muted-foreground">{history.length}</span>
        </button>
      </div>

      {sidebarTab === 'requests' ? (
        <>
          <div className="flex items-center gap-1.5 px-3 pt-2.5 pb-1">
            <div className="relative flex-1">
              <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-3 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Filter requests"
                className="h-7 border-transparent bg-muted/50 pl-7 text-xs"
              />
            </div>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => createCollection()}
                  aria-label="New collection"
                  className="text-muted-foreground"
                >
                  <FolderPlusIcon className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>New collection</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => startDraft()}
                  aria-label="New request"
                  className="text-muted-foreground"
                >
                  <PlusIcon className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>New request</TooltipContent>
            </Tooltip>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto px-2 pt-1 pb-3">
            {groupedCollections.map((col) => {
              const isCollapsed = collapsed[col.id]
              return (
                <div key={col.id} className="mb-0.5">
                  <div className="group flex items-center gap-1 rounded-md py-1 pr-1 pl-1 hover:bg-accent/40">
                    <button
                      type="button"
                      onClick={() =>
                        setCollapsed((prev) => ({ ...prev, [col.id]: !prev[col.id] }))
                      }
                      className="flex min-w-0 flex-1 items-center gap-1 text-left"
                      aria-expanded={!isCollapsed}
                    >
                      {isCollapsed ? (
                        <ChevronRightIcon className="size-3 shrink-0 text-muted-foreground" />
                      ) : (
                        <ChevronDownIcon className="size-3 shrink-0 text-muted-foreground" />
                      )}
                      <FolderIcon className="size-3.5 shrink-0 text-muted-foreground" />
                      <span className="truncate text-[12px] font-medium text-foreground/85">
                        {col.name}
                      </span>
                      <span className="mono text-[10px] text-muted-foreground/70">
                        {col.items.length}
                      </span>
                    </button>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          aria-label={`Actions for ${col.name}`}
                          className="opacity-0 group-hover:opacity-100"
                        >
                          <MoreHorizontalIcon className="size-3.5" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-40">
                        <DropdownMenuItem
                          onClick={() =>
                            startDraft({ collectionId: col.id, name: 'Untitled request' })
                          }
                        >
                          <PlusIcon /> New request here
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => renameCollection(col.id, col.name)}>
                          Rename
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => void deleteCollection(col.id, col.name)}
                        >
                          <Trash2Icon /> Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                  {!isCollapsed && (
                    <div className="ml-2 flex flex-col gap-px border-l border-border/60 pl-1.5">
                      {col.items.length === 0 ? (
                        <span className="py-1 pl-2 text-[11px] text-muted-foreground/70">
                          Empty
                        </span>
                      ) : (
                        col.items.map(renderRequest)
                      )}
                    </div>
                  )}
                </div>
              )
            })}

            {loose.length > 0 && (
              <div className="mt-1">
                <div className="eyebrow px-2 py-1">Unsorted</div>
                <div className="flex flex-col gap-px">{loose.map(renderRequest)}</div>
              </div>
            )}

            {filtered.length === 0 && (
              <div className="mt-8 px-3 text-center">
                <p className="text-xs text-muted-foreground">
                  {query ? 'Nothing matches that filter.' : 'No saved requests yet.'}
                </p>
                {!query && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2"
                    onClick={() => startDraft()}
                  >
                    <PlusIcon /> New request
                  </Button>
                )}
              </div>
            )}
          </div>
        </>
      ) : (
        <HistoryList />
      )}

      {/* footer */}
      <div className="hairline-t flex items-center gap-1 px-2 py-1.5">
        <span className="mono flex-1 truncate px-1 text-[10.5px] text-muted-foreground">
          {activeEnv ? `env · ${activeEnv.name}` : 'no environment'}
        </span>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Import workspace"
              onClick={() => fileRef.current?.click()}
              className="text-muted-foreground"
            >
              <UploadIcon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Import</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Export workspace"
              onClick={() => void exportWorkspace()}
              className="text-muted-foreground"
            >
              <DownloadIcon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Export</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Request settings"
              onClick={onOpenSettings}
              className="text-muted-foreground"
            >
              <Settings2Icon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Run settings</TooltipContent>
        </Tooltip>
        <input
          ref={fileRef}
          type="file"
          accept="application/json"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) void importWorkspace(file)
            e.target.value = ''
          }}
        />
      </div>

      <PromptDialog
        open={prompt !== null}
        title={prompt?.title ?? ''}
        description={prompt?.description}
        fields={[
          {
            name: 'value',
            label: prompt?.label ?? '',
            value: prompt?.value ?? '',
            placeholder: prompt?.placeholder,
            hint: prompt?.hint,
          },
        ]}
        confirmLabel={prompt?.confirmLabel ?? 'Save'}
        onCancel={() => setPrompt(null)}
        onSubmit={(values) => {
          const current = prompt
          setPrompt(null)
          void current?.onConfirm(values.value ?? '')
        }}
      />
    </aside>
  )
}

function HistoryList() {
  const { history, setHistory, select, startDraft } = useWorkspace()
  const [query, setQuery] = useState('')

  const refresh = async () => {
    const entries = await api.history(query)
    setHistory(entries)
  }

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query])

  const clearAll = async () => {
    if (!window.confirm('Clear all history? This cannot be undone.')) return
    await api.clearHistory()
    setHistory([])
    toast.success('History cleared')
  }

  const removeOne = async (id: string) => {
    await api.deleteHistory(id)
    setHistory(history.filter((h) => h.id !== id))
  }

  const openEntry = async (id: string) => {
    const entry = await api.historyEntry(id)
    if (!entry.requestSnapshot) {
      toast.error('This entry has no saved request')
      return
    }
    try {
      const snapshot = JSON.parse(entry.requestSnapshot) as { request: ApiRequest }
      select(null)
      startDraft(snapshot.request)
    } catch {
      toast.error('Could not read that entry')
    }
  }

  return (
    <>
      <div className="flex items-center gap-1.5 px-3 pt-2.5 pb-1">
        <div className="relative flex-1">
          <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-3 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search history"
            className="h-7 border-transparent bg-muted/50 pl-7 text-xs"
          />
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => void clearAll()}
              aria-label="Clear history"
              className="text-muted-foreground"
            >
              <Trash2Icon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Clear all</TooltipContent>
        </Tooltip>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
        {history.length === 0 ? (
          <div className="mt-8 px-3 text-center">
            <ClockIcon className="mx-auto mb-2 size-5 text-muted-foreground/50" />
            <p className="text-xs text-muted-foreground">
              Runs show up here, with the exact request that produced them.
            </p>
          </div>
        ) : (
          history.map((h) => (
            <div
              key={h.id}
              className="group flex items-start gap-2 rounded-md py-1.5 pr-1 pl-2 hover:bg-accent/50"
            >
              <button
                type="button"
                onClick={() => void openEntry(h.id)}
                className="min-w-0 flex-1 text-left"
              >
                <div className="flex items-center gap-1.5">
                  <span
                    className={cn(
                      'mono text-[10px] font-bold',
                      h.status >= 200 && h.status < 300
                        ? 'text-ok'
                        : h.status === 0
                          ? 'text-muted-foreground'
                          : 'text-bad',
                    )}
                  >
                    {h.status || 'ERR'}
                  </span>
                  <span className="mono text-[10px] text-muted-foreground">{h.method}</span>
                  <span className="ml-auto mono text-[10px] text-muted-foreground/70">
                    {timeAgo(h.createdAt)}
                  </span>
                </div>
                <div className="truncate text-[12px] text-foreground/85">{h.name || h.url}</div>
                <div className="mono truncate text-[10px] text-muted-foreground/70">
                  {h.durationMs} ms · {pathOf(h.url)}
                </div>
              </button>
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label="Delete history entry"
                onClick={() => void removeOne(h.id)}
                className="opacity-0 group-hover:opacity-100"
              >
                <Trash2Icon className="size-3" />
              </Button>
            </div>
          ))
        )}
      </div>
    </>
  )
}
