export type KV = {
  id: string
  key: string
  value: string
  enabled: boolean
  description?: string
}

export type Auth = {
  token?: string
  username?: string
  password?: string
  key?: string
  value?: string
  in?: string
}

export type BodyType = 'none' | 'json' | 'text' | 'xml' | 'form'
export type AuthType = 'none' | 'bearer' | 'basic' | 'apikey'
export type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS'

export type ApiRequest = {
  id: string
  collectionId: string | null
  name: string
  method: string
  url: string
  params: KV[]
  headers: KV[]
  bodyType: BodyType | string
  body: string
  form: KV[]
  authType: AuthType | string
  auth: Auth
  sortIndex: number
  exposeSlug: string
  exposed: boolean
  createdAt: string
  updatedAt: string
}

export type Collection = {
  id: string
  name: string
  sortIndex: number
  createdAt: string
  updatedAt: string
}

export type EnvVar = {
  id: string
  key: string
  value: string
  enabled: boolean
  secret: boolean
}

export type Environment = {
  id: string
  name: string
  sortIndex: number
  variables: EnvVar[]
  createdAt: string
  updatedAt: string
}

export type HistoryEntry = {
  id: string
  requestId: string
  name: string
  method: string
  url: string
  status: number
  durationMs: number
  sizeBytes: number
  ok: boolean
  error: string
  requestSnapshot?: string
  responseHeaders?: string
  responseBody?: string
  bodyEncoding?: string
  createdAt: string
}

export type RunResponse = {
  status: number
  statusText: string
  headers: Record<string, string[]> | null
  body: string
  encoding: 'text' | 'base64'
  contentType: string
  durationMs: number
  sizeBytes: number
  truncated: boolean
  finalUrl: string
  cookies: string[]
  receivedAt: string
  historyId: string
  envName: string
  resolvedUrl: string
  error?: string
}

export type ExecOptions = {
  timeoutSeconds: number
  followRedirects: boolean
  verifyTls: boolean
}

export type Stats = {
  collections: number
  requests: number
  history: number
  environments: number
}
