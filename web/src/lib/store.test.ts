import { beforeEach, describe, expect, it } from 'vitest'
import { blankRequest, useWorkspace } from '@/lib/store'
import type { ApiRequest } from '@/lib/types'

export function makeRequest(overrides: Partial<ApiRequest> = {}): ApiRequest {
  return {
    ...blankRequest(),
    id: 'req-1',
    name: 'Saved request',
    method: 'GET',
    url: 'https://api.test/a',
    ...overrides,
  }
}

function reset() {
  useWorkspace.setState({
    collections: [],
    requests: [],
    environments: [],
    activeEnvId: '',
    history: [],
    selectedId: null,
    draft: null,
    dirty: false,
    sidebarTab: 'requests',
  })
}

beforeEach(reset)

describe('select', () => {
  it('copies the saved request into the draft and clears dirty', () => {
    const saved = makeRequest()
    useWorkspace.setState({ requests: [saved] })

    useWorkspace.getState().select(saved.id)

    const state = useWorkspace.getState()
    expect(state.selectedId).toBe(saved.id)
    expect(state.draft?.name).toBe('Saved request')
    expect(state.dirty).toBe(false)
    // The draft must be a copy: editing it must not mutate the saved list.
    expect(state.draft).not.toBe(saved)
  })

  it('clears the draft when the id is unknown', () => {
    useWorkspace.setState({ requests: [makeRequest()] })
    useWorkspace.getState().select('missing')

    const state = useWorkspace.getState()
    expect(state.draft).toBeNull()
    expect(state.selectedId).toBe('missing')
  })
})

describe('startDraft', () => {
  it('creates an unsaved draft and marks it dirty', () => {
    useWorkspace.getState().startDraft()

    const state = useWorkspace.getState()
    expect(state.draft?.id).toBe('')
    expect(state.dirty).toBe(true)
    expect(state.selectedId).toBeNull()
  })

  it('applies partial overrides', () => {
    useWorkspace.getState().startDraft({ method: 'POST', url: 'https://api.test/x' })

    const draft = useWorkspace.getState().draft
    expect(draft?.method).toBe('POST')
    expect(draft?.url).toBe('https://api.test/x')
    // Untouched fields keep their blank defaults.
    expect(draft?.bodyType).toBe('none')
  })
})

describe('editDraft', () => {
  it('patches the draft and marks it dirty', () => {
    useWorkspace.getState().startDraft()
    useWorkspace.getState().editDraft({ name: 'Renamed' })

    const state = useWorkspace.getState()
    expect(state.draft?.name).toBe('Renamed')
    expect(state.dirty).toBe(true)
  })

  it('does nothing when there is no draft', () => {
    useWorkspace.getState().editDraft({ name: 'ignored' })
    expect(useWorkspace.getState().draft).toBeNull()
  })

  it('does not touch the saved list', () => {
    const saved = makeRequest()
    useWorkspace.setState({ requests: [saved] })
    useWorkspace.getState().select(saved.id)
    useWorkspace.getState().editDraft({ name: 'changed locally' })

    expect(useWorkspace.getState().requests[0].name).toBe('Saved request')
  })
})

describe('setDraft', () => {
  it('adopts a server copy without marking dirty', () => {
    useWorkspace.getState().startDraft()
    useWorkspace.getState().setDraft(makeRequest({ name: 'from server' }))

    const state = useWorkspace.getState()
    expect(state.draft?.name).toBe('from server')
    expect(state.dirty).toBe(false)
  })
})

describe('options', () => {
  it('merges a partial patch', () => {
    useWorkspace.getState().setOptions({ timeoutSeconds: 90 })
    const o = useWorkspace.getState().options
    expect(o.timeoutSeconds).toBe(90)
    // Other options keep their defaults.
    expect(o.followRedirects).toBe(true)
    expect(o.verifyTls).toBe(true)
  })
})

describe('sidebarTab', () => {
  it('switches between requests and history', () => {
    useWorkspace.getState().setSidebarTab('history')
    expect(useWorkspace.getState().sidebarTab).toBe('history')
    useWorkspace.getState().setSidebarTab('requests')
    expect(useWorkspace.getState().sidebarTab).toBe('requests')
  })
})

describe('blankRequest', () => {
  it('produces an unsaved request with inert defaults', () => {
    const r = blankRequest()
    expect(r.id).toBe('')
    expect(r.method).toBe('GET')
    expect(r.bodyType).toBe('none')
    expect(r.authType).toBe('none')
    expect(r.exposed).toBe(false)
    expect(r.params).toEqual([])
  })

  it('returns a new object each time', () => {
    const a = blankRequest()
    const b = blankRequest()
    expect(a).not.toBe(b)
    a.name = 'mutated'
    expect(b.name).toBe('Untitled request')
  })
})
