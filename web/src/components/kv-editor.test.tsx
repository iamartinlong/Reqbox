import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { KVEditor } from '@/components/kv-editor'
import type { KV } from '@/lib/types'

const row = (overrides: Partial<KV> = {}): KV => ({
  id: crypto.randomUUID(),
  key: 'a',
  value: '1',
  enabled: true,
  description: '',
  ...overrides,
})

describe('KVEditor', () => {
  it('invites adding the first row when empty', () => {
    render(<KVEditor rows={[]} onChange={() => {}} kind="query" emptyHint="Add one" />)
    expect(screen.getByText('Add one')).toBeInTheDocument()
  })

  it('appends a row when the empty-state button is clicked', async () => {
    const onChange = vi.fn()
    render(<KVEditor rows={[]} onChange={onChange} kind="query" emptyHint="Add one" />)

    await userEvent.click(screen.getByText('Add one'))

    expect(onChange).toHaveBeenCalledOnce()
    const next = onChange.mock.calls[0][0] as KV[]
    expect(next).toHaveLength(1)
    expect(next[0]).toMatchObject({ key: '', value: '', enabled: true })
  })

  it('edits a key without disturbing the other rows', async () => {
    const rows = [row({ key: 'first' }), row({ key: 'second' })]
    const onChange = vi.fn()
    render(<KVEditor rows={rows} onChange={onChange} kind="query" emptyHint="x" />)

    const inputs = screen.getAllByPlaceholderText('limit')
    await userEvent.type(inputs[1], 'X')

    const next = onChange.mock.calls.at(-1)?.[0] as KV[]
    expect(next[0].key).toBe('first')
    expect(next[1].key).toBe('secondX')
  })

  it('toggles inclusion without removing the row', async () => {
    const rows = [row()]
    const onChange = vi.fn()
    render(<KVEditor rows={rows} onChange={onChange} kind="query" emptyHint="x" />)

    await userEvent.click(screen.getByLabelText('Include a'))

    const next = onChange.mock.calls.at(-1)?.[0] as KV[]
    expect(next).toHaveLength(1)
    expect(next[0].enabled).toBe(false)
  })

  it('removes a row', async () => {
    const rows = [row({ key: 'keep' }), row({ key: 'drop' })]
    const onChange = vi.fn()
    render(<KVEditor rows={rows} onChange={onChange} kind="query" emptyHint="x" />)

    const removeButtons = screen.getAllByLabelText('Remove row')
    await userEvent.click(removeButtons[1])

    const next = onChange.mock.calls.at(-1)?.[0] as KV[]
    expect(next).toHaveLength(1)
    expect(next[0].key).toBe('keep')
  })

  it('uses header-appropriate placeholders', () => {
    render(<KVEditor rows={[row({ key: '', value: '' })]} onChange={() => {}} kind="header" emptyHint="x" />)
    expect(screen.getByPlaceholderText('Authorization')).toBeInTheDocument()
  })
})
