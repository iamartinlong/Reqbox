import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { session } from './api'

/**
 * A 401 has to announce itself, because the workspace cannot recover from it on
 * its own: without the event the user gets a toast and a dead UI.
 */
describe('unauthorized signalling', () => {
  const originalFetch = globalThis.fetch
  let events: string[] = []

  const onUnauthorized = () => events.push('unauthorized')

  beforeEach(() => {
    events = []
    window.addEventListener('reqbox:unauthorized', onUnauthorized)
  })

  afterEach(() => {
    window.removeEventListener('reqbox:unauthorized', onUnauthorized)
    globalThis.fetch = originalFetch
  })

  const respond = (status: number, body: unknown) => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      statusText: 'x',
      json: async () => body,
    }) as unknown as typeof fetch
  }

  it('emits the event on a 401 from a normal call', async () => {
    respond(401, { error: 'missing or invalid API token' })
    await expect(session.signOut()).rejects.toThrow()
    expect(events).toEqual(['unauthorized'])
  })

  it('does not emit on other failures', async () => {
    respond(500, { error: 'boom' })
    await expect(session.signOut()).rejects.toThrow()
    expect(events).toEqual([])
  })

  it('does not emit for the status endpoint, where 401 means "signed out"', async () => {
    // Otherwise loading the app while signed out would fire the event, which
    // is harmless but makes the flow impossible to reason about.
    respond(401, { ok: false })
    const status = await session.status()
    expect(status).toEqual({ ok: false, auth: true })
    expect(events).toEqual([])
  })

  it('does not emit for a failed sign-in attempt', async () => {
    // A wrong token is an expected answer on the unlock screen; treating it as
    // "session lost" would loop the screen against itself.
    respond(401, { error: 'invalid token' })
    await expect(session.signIn('wrong')).rejects.toThrow()
    expect(events).toEqual([])
  })

  it('reports signed-in status as ok', async () => {
    respond(200, { ok: true, auth: true })
    expect(await session.status()).toEqual({ ok: true, auth: true })
  })

  it('treats auth being off as signed in', async () => {
    // With -no-auth the frontend must not show the unlock screen at all.
    respond(200, { ok: true, auth: false })
    expect(await session.status()).toEqual({ ok: true, auth: false })
  })
})
