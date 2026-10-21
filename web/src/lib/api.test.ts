import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError } from '@/lib/api'

/** Build a fetch stub returning one response. */
function stubFetch(body: unknown, init: { status?: number; statusText?: string } = {}) {
  const status = init.status ?? 200
  const spy = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    statusText: init.statusText ?? '',
    json: async () => body,
  })
  vi.stubGlobal('fetch', spy)
  return spy
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('api', () => {
  it('prefixes requests with /api and sends JSON', async () => {
    const spy = stubFetch({ status: 'ok' })
    await api.health()

    expect(spy).toHaveBeenCalledOnce()
    const [url, init] = spy.mock.calls[0]
    expect(url).toBe('/api/health')
    expect(init.headers['Content-Type']).toBe('application/json')
  })

  it('serializes a body for writes', async () => {
    const spy = stubFetch({ id: '1', name: 'X' }, { status: 201 })
    await api.createCollection('X')

    const [url, init] = spy.mock.calls[0]
    expect(url).toBe('/api/collections')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ name: 'X' })
  })

  it('returns undefined for 204 instead of parsing an empty body', async () => {
    stubFetch(null, { status: 204 })
    await expect(api.deleteRequest('abc')).resolves.toBeUndefined()
  })

  it('surfaces the server error message', async () => {
    stubFetch({ error: 'mock slug is already in use' }, { status: 409, statusText: 'Conflict' })

    await expect(api.createRequest({ name: 'x' })).rejects.toThrow('mock slug is already in use')
    await expect(api.createRequest({ name: 'x' })).rejects.toBeInstanceOf(ApiError)
  })

  it('falls back to statusText when the error body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        statusText: 'Bad Gateway',
        json: async () => {
          throw new Error('not json')
        },
      }),
    )

    await expect(api.stats()).rejects.toThrow('Bad Gateway')
  })

  it('propagates transport failures', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network down')))
    await expect(api.health()).rejects.toThrow('network down')
  })

  it('encodes the history query string', async () => {
    const spy = stubFetch([])
    await api.history('a b&c', 20)

    const [url] = spy.mock.calls[0]
    expect(url).toBe('/api/history?q=a%20b%26c&limit=20')
  })

  it('exposes the status on ApiError', async () => {
    stubFetch({ error: 'nope' }, { status: 404 })
    await expect(api.collections()).rejects.toMatchObject({ status: 404 })
  })
})
