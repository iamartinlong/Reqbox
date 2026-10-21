import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { migratedStorage } from './storage'
import type { ApiRequest, Collection, Environment, ExecOptions, HistoryEntry } from './types'

export const blankRequest = (): ApiRequest => ({
  id: '',
  collectionId: null,
  name: 'Untitled request',
  method: 'GET',
  url: '',
  params: [],
  headers: [],
  bodyType: 'none',
  body: '',
  form: [],
  authType: 'none',
  auth: {},
  sortIndex: 0,
  exposeSlug: '',
  exposed: false,
  createdAt: '',
  updatedAt: '',
})

type WorkspaceState = {
  collections: Collection[]
  requests: ApiRequest[]
  environments: Environment[]
  activeEnvId: string
  history: HistoryEntry[]

  selectedId: string | null
  draft: ApiRequest | null
  dirty: boolean
  sidebarTab: 'requests' | 'history'

  options: ExecOptions
  setOptions: (patch: Partial<ExecOptions>) => void

  setWorkspace: (data: {
    collections: Collection[]
    requests: ApiRequest[]
    environments: Environment[]
    activeEnvId: string
  }) => void
  setHistory: (entries: HistoryEntry[]) => void

  select: (id: string | null) => void
  startDraft: (partial?: Partial<ApiRequest>) => void
  editDraft: (patch: Partial<ApiRequest>) => void
  setDraft: (request: ApiRequest) => void

  setSidebarTab: (tab: 'requests' | 'history') => void
}

const initialOptions: ExecOptions = {
  timeoutSeconds: 30,
  followRedirects: true,
  verifyTls: true,
}

export const useWorkspace = create<WorkspaceState>()(
  persist(
    (set, get) => ({
      collections: [],
      requests: [],
      environments: [],
      activeEnvId: '',
      history: [],

      selectedId: null,
      draft: null,
      dirty: false,
      sidebarTab: 'requests',

      options: initialOptions,
      setOptions: (patch) => set({ options: { ...get().options, ...patch } }),

      setWorkspace: ({ collections, requests, environments, activeEnvId }) =>
        set({ collections, requests, environments, activeEnvId }),

      setHistory: (entries) => set({ history: entries }),

      select: (id) => {
        const found = get().requests.find((r) => r.id === id) ?? null
        set({ selectedId: id, draft: found ? { ...found } : null, dirty: false })
      },

      startDraft: (partial) =>
        set({
          selectedId: null,
          draft: { ...blankRequest(), ...partial },
          dirty: true,
        }),

      editDraft: (patch) => {
        const draft = get().draft
        if (!draft) return
        set({ draft: { ...draft, ...patch }, dirty: true })
      },

      setDraft: (request) => set({ draft: request, dirty: false }),

      setSidebarTab: (tab) => set({ sidebarTab: tab }),
    }),
    {
      name: 'reqbox-session',
      storage: migratedStorage(),
      partialize: (state) => ({ options: state.options }),
    },
  ),
)
