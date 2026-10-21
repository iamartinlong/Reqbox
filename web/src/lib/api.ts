import type {
  ApiRequest,
  Collection,
  Environment,
  ExecOptions,
  HistoryEntry,
  RunResponse,
  Stats,
} from './types'

const BASE = '/api'

class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })
  if (!res.ok) {
    let message = res.statusText
    try {
      const body = (await res.json()) as { error?: string }
      if (body?.error) message = body.error
    } catch {
      /* response had no JSON body */
    }
    // A 401 means the session is gone or was never established. Announce it so
    // the app can show the unlock screen instead of a toast the user cannot act
    // on. The status endpoint is excluded because it uses 401 as its normal
    // "signed out" answer.
    if (res.status === 401 && !path.startsWith('/session')) {
      window.dispatchEvent(new CustomEvent('reqbox:unauthorized'))
    }
    throw new ApiError(res.status, message)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export type SessionStatus = { ok: boolean; auth: boolean }

export const session = {
  /** Signed-out is a normal answer here, not an error to surface. */
  status: async (): Promise<SessionStatus> => {
    const res = await fetch(`${BASE}/session/status`, { headers: { 'Content-Type': 'application/json' } })
    if (res.status === 401) return { ok: false, auth: true }
    if (!res.ok) throw new ApiError(res.status, res.statusText)
    return (await res.json()) as SessionStatus
  },
  signIn: (token: string) =>
    request<{ ok: boolean }>('/session', { method: 'POST', body: JSON.stringify({ token }) }),
  signOut: () => request<{ ok: boolean }>('/logout', { method: 'POST' }),
}

export const api = {
  health: () => request<{ status: string; version: string }>('/health'),
  stats: () => request<Stats>('/stats'),

  collections: () => request<Collection[]>('/collections'),
  createCollection: (name: string) =>
    request<Collection>('/collections', { method: 'POST', body: JSON.stringify({ name }) }),
  updateCollection: (id: string, patch: { name?: string; sortIndex?: number }) =>
    request<Collection>(`/collections/${id}`, { method: 'PUT', body: JSON.stringify(patch) }),
  deleteCollection: (id: string) => request<void>(`/collections/${id}`, { method: 'DELETE' }),

  requests: () => request<ApiRequest[]>('/requests'),
  createRequest: (req: Partial<ApiRequest>) =>
    request<ApiRequest>('/requests', { method: 'POST', body: JSON.stringify(req) }),
  updateRequest: (req: ApiRequest) =>
    request<ApiRequest>(`/requests/${req.id}`, { method: 'PUT', body: JSON.stringify(req) }),
  deleteRequest: (id: string) => request<void>(`/requests/${id}`, { method: 'DELETE' }),
  duplicateRequest: (id: string) =>
    request<ApiRequest>(`/requests/${id}/duplicate`, { method: 'POST' }),

  execute: (payload: {
    request: Partial<ApiRequest>
    environmentId?: string
    variables?: Record<string, string>
    options?: ExecOptions
  }) => request<RunResponse>('/execute', { method: 'POST', body: JSON.stringify(payload) }),

  curl: (payload: { request: Partial<ApiRequest>; environmentId?: string }) =>
    request<{ curl: string }>('/curl', { method: 'POST', body: JSON.stringify(payload) }),

  history: (q = '', limit = 100) =>
    request<HistoryEntry[]>(`/history?q=${encodeURIComponent(q)}&limit=${limit}`),
  historyEntry: (id: string) => request<HistoryEntry>(`/history/${id}`),
  deleteHistory: (id: string) => request<void>(`/history/${id}`, { method: 'DELETE' }),
  clearHistory: () => request<{ deleted: number }>('/history', { method: 'DELETE' }),
  replayHistory: (id: string) =>
    request<RunResponse>(`/history/${id}/replay`, { method: 'POST' }),

  environments: () =>
    request<{ items: Environment[]; active: string }>('/environments'),
  createEnvironment: (name: string, variables: Environment['variables']) =>
    request<Environment>('/environments', {
      method: 'POST',
      body: JSON.stringify({ name, variables }),
    }),
  updateEnvironment: (env: Environment) =>
    request<Environment>(`/environments/${env.id}`, {
      method: 'PUT',
      body: JSON.stringify({ name: env.name, variables: env.variables }),
    }),
  deleteEnvironment: (id: string) => request<void>(`/environments/${id}`, { method: 'DELETE' }),
  activateEnvironment: (id: string) =>
    request<{ active: string }>('/environments/active', {
      method: 'PUT',
      body: JSON.stringify({ id }),
    }),

  importWorkspace: (data: unknown) =>
    request<{ collections: number; requests: number; environments: number }>('/import', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
}

export { ApiError }
