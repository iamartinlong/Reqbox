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
import { useConfirm } from '@/components/confirm-dialog'
import { api } from '@/lib/api'
import { methodTint, pathOf, slugify, timeAgo } from '@/lib/format'
import { useWorkspace } from '@/lib/store'
import { tNow, useLanguageStore, useT } from '@/lib/i18n'
import type { ApiRequest } from '@/lib/types'
import { toast } from 'sonner'

type Props = {
  onOpenEnvironments: () => void
  onOpenSettings: () => void
  /** Mobile only: called after picking a request so the drawer can close. */
  onNavigate?: () => void
}

export function Sidebar({ onOpenEnvironments, onOpenSettings, onNavigate }: Props) {
  const t = useT()
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
  const { confirm, dialog: confirmDialog } = useConfirm()

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
      title: tNow('prompt.newCollection.title'),
      description: tNow('prompt.newCollection.desc'),
      label: tNow('prompt.newCollection.label'),
      value: '',
      placeholder: tNow('prompt.newCollection.placeholder'),
      confirmLabel: tNow('prompt.newCollection.confirm'),
      onConfirm: async (value) => {
        const name = value.trim()
        if (!name) return
        await api.createCollection(name)
        setReloadKey((k) => k + 1)
        toast.success(tNow('toast.collectionCreated'))
      },
    })
  }

  const deleteCollection = async (id: string, name: string) => {
    if (
      !(await confirm({
        title: tNow('confirm.deleteCollection.title', { name }),
        description: tNow('confirm.deleteCollection.desc'),
        confirmLabel: tNow('confirm.deleteCollection.confirm'),
      }))
    )
      return
    await api.deleteCollection(id)
    setReloadKey((k) => k + 1)
    toast.success(tNow('toast.collectionDeleted'))
  }

  const renameCollection = (id: string, current: string) => {
    setPrompt({
      title: tNow('prompt.renameCollection.title'),
      label: tNow('prompt.newCollection.label'),
      value: current,
      confirmLabel: tNow('prompt.renameCollection.confirm'),
      onConfirm: async (value) => {
        const name = value.trim()
        if (!name || name === current) return
        await api.updateCollection(id, { name })
        setReloadKey((k) => k + 1)
        toast.success(tNow('toast.collectionRenamed'))
      },
    })
  }

  const activateEnv = async (id: string) => {
    await api.activateEnvironment(id)
    setReloadKey((k) => k + 1)
  }

  const exportWorkspace = async () => {
    const res = await fetch('/api/export')
    // Without this check a 401 body (an error object) would be saved as the
    // export file, which looks like a corrupt backup rather than a lost session.
    if (!res.ok) {
      if (res.status === 401) {
        window.dispatchEvent(new CustomEvent('reqbox:unauthorized'))
      }
      toast.error(tNow('toast.exportFailed'))
      return
    }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'reqbox-export.json'
    a.click()
    URL.revokeObjectURL(url)
    toast.success(tNow('toast.workspaceExported'))
  }

  const importWorkspace = async (file: File) => {
    let data: unknown
    try {
      const text = await file.text()
      data = JSON.parse(text)
    } catch {
      toast.error(tNow('toast.invalidExport'))
      return
    }

    // Import replaces the whole workspace. Say so before touching anything.
    const counts = data as {
      requests?: unknown[]
      collections?: unknown[]
      environments?: unknown[]
    }
    const incoming =
      (counts.requests?.length ?? 0) +
      (counts.collections?.length ?? 0) +
      (counts.environments?.length ?? 0)
    if (
      !(await confirm({
        title: tNow('confirm.import.title'),
        description:
          incoming > 0
            ? tNow('confirm.import.desc', { count: incoming })
            : tNow('confirm.import.descEmpty'),
        confirmLabel: tNow('confirm.import.confirm'),
      }))
    )
      return

    try {
      const res = await api.importWorkspace(data)
      setReloadKey((k) => k + 1)
      toast.success(tNow('toast.imported', { count: res.requests }))
    } catch (e) {
      toast.error(e instanceof Error ? e.message : tNow('toast.importFailed'))
    }
  }

  const groupedCollections = collections.map((c) => ({
    ...c,
    items: grouped.get(c.id) ?? [],
  }))
  const loose = grouped.get('__none__') ?? []

  const deleteRequest = async (r: ApiRequest) => {
    if (
      !(await confirm({
        title: tNow('confirm.deleteRequest.title', { name: r.name }),
        description: tNow('confirm.deleteRequest.desc'),
        confirmLabel: tNow('confirm.deleteRequest.confirm'),
      }))
    )
      return
    await api.deleteRequest(r.id)
    setReloadKey((k) => k + 1)
    if (selectedId === r.id) select(null)
    toast.success(tNow('toast.requestDeleted'))
  }

  const duplicateRequest = async (r: ApiRequest) => {
    const copy = await api.duplicateRequest(r.id)
    setReloadKey((k) => k + 1)
    toast.success(tNow('toast.duplicated', { name: copy.name }))
  }

  const toggleExposed = async (r: ApiRequest) => {
    await api.updateRequest({ ...r, exposed: !r.exposed })
    setReloadKey((k) => k + 1)
    toast.success(r.exposed ? tNow('toast.mockUnpublished') : tNow('toast.mockPublished'))
  }

  const publishRequest = (r: ApiRequest) => {
    setPrompt({
      title: tNow('prompt.publish.title'),
      description: tNow('prompt.publish.desc'),
      label: tNow('prompt.publish.label'),
      value: r.exposeSlug || slugify(r.name),
      placeholder: tNow('prompt.publish.placeholder'),
      hint: tNow('prompt.publish.hint'),
      confirmLabel: tNow('prompt.publish.confirm'),
      onConfirm: async (value) => {
        const slug = slugify(value)
        if (!slug) {
          toast.error(tNow('toast.slugRequired'))
          return
        }
        try {
          await api.updateRequest({ ...r, exposeSlug: slug, exposed: true })
          setReloadKey((k) => k + 1)
          toast.success(tNow('toast.publishedAt', { slug }))
        } catch (e) {
          toast.error(e instanceof Error ? e.message : tNow('toast.cannotPublish'))
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
            aria-label={t('sidebar.publishedBadge')}
          />
        )}
      </button>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={t('sidebar.aria.actions', { name: r.name })}
            className="mr-1 opacity-0 group-hover:opacity-100 data-[state=open]:opacity-100"
          >
            <MoreHorizontalIcon className="size-3.5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={() => void duplicateRequest(r)}>
            <CopyIcon /> {t('sidebar.duplicate')}
          </DropdownMenuItem>
          {r.exposed ? (
            <DropdownMenuItem onClick={() => void toggleExposed(r)}>
              <GlobeIcon /> {t('sidebar.unpublishMock')}
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => publishRequest(r)}>
              <GlobeIcon /> {t('sidebar.publishMock')}
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => void deleteRequest(r)}>
            <Trash2Icon /> {t('common.delete')}
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
              <SelectValue placeholder={t('sidebar.noEnvironmentPlaceholder')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">{t('sidebar.noEnvironmentPlaceholder')}</SelectItem>
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
              aria-label={t('sidebar.manageEnvAria')}
              className="text-muted-foreground"
            >
              <Settings2Icon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('env.title')}</TooltipContent>
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
          {t('sidebar.requests')}
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
          {t('sidebar.history')}
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
                placeholder={t('sidebar.filterRequests')}
                className="h-7 border-transparent bg-muted/50 pl-7 text-xs"
              />
            </div>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => createCollection()}
                  aria-label={t('sidebar.newCollection')}
                  className="text-muted-foreground"
                >
                  <FolderPlusIcon className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t('sidebar.newCollection')}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => startDraft()}
                  aria-label={t('sidebar.newRequest')}
                  className="text-muted-foreground"
                >
                  <PlusIcon className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t('sidebar.newRequest')}</TooltipContent>
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
                          aria-label={t('sidebar.aria.actions', { name: col.name })}
                          className="opacity-0 group-hover:opacity-100"
                        >
                          <MoreHorizontalIcon className="size-3.5" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-40">
                        <DropdownMenuItem
                          onClick={() =>
                            startDraft({ collectionId: col.id, name: t('sidebar.untitledRequest') })
                          }
                        >
                          <PlusIcon /> {t('sidebar.newRequestHere')}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => renameCollection(col.id, col.name)}>
                          {t('common.rename')}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => void deleteCollection(col.id, col.name)}
                        >
                          <Trash2Icon /> {t('sidebar.deleteCollection')}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                  {!isCollapsed && (
                    <div className="ml-2 flex flex-col gap-px border-l border-border/60 pl-1.5">
                      {col.items.length === 0 ? (
                        <span className="py-1 pl-2 text-[11px] text-muted-foreground/70">
                          {t('sidebar.empty')}
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
                <div className="eyebrow px-2 py-1">{t('sidebar.unsorted')}</div>
                <div className="flex flex-col gap-px">{loose.map(renderRequest)}</div>
              </div>
            )}

            {filtered.length === 0 && (
              <div className="mt-8 px-3 text-center">
                <p className="text-xs text-muted-foreground">
                  {query ? t('sidebar.noMatch') : t('sidebar.noRequests')}
                </p>
                {!query && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2"
                    onClick={() => startDraft()}
                  >
                    <PlusIcon /> {t('sidebar.newRequest')}
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
          {activeEnv ? `${t('sidebar.envPrefix')} · ${activeEnv.name}` : t('sidebar.noEnvironment')}
        </span>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={t('sidebar.aria.import')}
              onClick={() => fileRef.current?.click()}
              className="text-muted-foreground"
            >
              <UploadIcon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('sidebar.import')}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={t('sidebar.aria.export')}
              onClick={() => void exportWorkspace()}
              className="text-muted-foreground"
            >
              <DownloadIcon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('sidebar.export')}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={t('sidebar.aria.settings')}
              onClick={onOpenSettings}
              className="text-muted-foreground"
            >
              <Settings2Icon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('sidebar.runSettings')}</TooltipContent>
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
        confirmLabel={prompt?.confirmLabel ?? t('common.save')}
        onCancel={() => setPrompt(null)}
        onSubmit={(values) => {
          const current = prompt
          setPrompt(null)
          void current?.onConfirm(values.value ?? '')
        }}
      />

      {confirmDialog}
    </aside>
  )
}

function HistoryList() {
  const t = useT()
  const language = useLanguageStore((s) => s.language)
  const { history, setHistory, select, startDraft } = useWorkspace()
  const [query, setQuery] = useState('')
  const { confirm, dialog: confirmDialog } = useConfirm()

  const refresh = async () => {
    const entries = await api.history(query)
    setHistory(entries)
  }

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query])

  const clearAll = async () => {
    if (
      !(await confirm({
        title: tNow('confirm.clearHistory.title'),
        description: tNow('confirm.clearHistory.desc'),
        confirmLabel: tNow('confirm.clearHistory.confirm'),
      }))
    )
      return
    await api.clearHistory()
    setHistory([])
    toast.success(tNow('toast.historyCleared'))
  }

  const removeOne = async (id: string) => {
    await api.deleteHistory(id)
    setHistory(history.filter((h) => h.id !== id))
  }

  const openEntry = async (id: string) => {
    const entry = await api.historyEntry(id)
    if (!entry.requestSnapshot) {
      toast.error(tNow('toast.noSavedRequest'))
      return
    }
    try {
      const snapshot = JSON.parse(entry.requestSnapshot) as { request: ApiRequest }
      select(null)
      startDraft(snapshot.request)
    } catch {
      toast.error(tNow('toast.cannotReadEntry'))
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
            placeholder={t('sidebar.searchHistory')}
            className="h-7 border-transparent bg-muted/50 pl-7 text-xs"
          />
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => void clearAll()}
              aria-label={t('sidebar.clearHistory')}
              className="text-muted-foreground"
            >
              <Trash2Icon className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t('sidebar.clearAll')}</TooltipContent>
        </Tooltip>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
        {history.length === 0 ? (
          <div className="mt-8 px-3 text-center">
            <ClockIcon className="mx-auto mb-2 size-5 text-muted-foreground/50" />
            <p className="text-xs text-muted-foreground">
              {t('sidebar.noHistory')}
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
                    {h.status || t('sidebar.err')}
                  </span>
                  <span className="mono text-[10px] text-muted-foreground">{h.method}</span>
                  <span className="ml-auto mono text-[10px] text-muted-foreground/70">
                    {timeAgo(h.createdAt, language)}
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
                aria-label={t('sidebar.deleteHistoryEntry')}
                onClick={() => void removeOne(h.id)}
                className="opacity-0 group-hover:opacity-100"
              >
                <Trash2Icon className="size-3" />
              </Button>
            </div>
          ))
        )}
      </div>

      {confirmDialog}
    </>
  )
}
