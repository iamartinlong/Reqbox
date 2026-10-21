import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConfirmDialog } from '@/components/confirm-dialog'

describe('ConfirmDialog', () => {
  it('renders nothing without options', () => {
    const { container } = render(
      <ConfirmDialog open={false} options={null} onResolve={() => {}} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('shows the title and description', () => {
    render(
      <ConfirmDialog
        open
        options={{ title: 'Delete "abc"?', description: 'Requests are kept.' }}
        onResolve={() => {}}
      />,
    )
    expect(screen.getByText('Delete "abc"?')).toBeInTheDocument()
    expect(screen.getByText('Requests are kept.')).toBeInTheDocument()
  })

  it('resolves true on confirm', async () => {
    let result: boolean | null = null
    render(
      <ConfirmDialog
        open
        options={{ title: 'Delete?', confirmLabel: 'Delete collection' }}
        onResolve={(ok) => (result = ok)}
      />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'Delete collection' }))
    expect(result).toBe(true)
  })

  it('resolves false on cancel', async () => {
    let result: boolean | null = null
    render(
      <ConfirmDialog open options={{ title: 'Delete?' }} onResolve={(ok) => (result = ok)} />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(result).toBe(false)
  })

  it('focuses Cancel so a stray Enter cannot delete', () => {
    render(<ConfirmDialog open options={{ title: 'Delete?' }} onResolve={() => {}} />)
    // The dialog sets focus explicitly on mount.
    expect(document.activeElement?.textContent).toBe('Cancel')
  })

  it('uses the destructive style by default and a plain one when asked', () => {
    const { rerender } = render(
      <ConfirmDialog open options={{ title: 'A' }} onResolve={() => {}} />,
    )
    expect(screen.getByRole('button', { name: 'Delete' })).toHaveAttribute(
      'data-variant',
      'destructive',
    )

    rerender(
      <ConfirmDialog
        open
        options={{ title: 'B', destructive: false, confirmLabel: 'Continue' }}
        onResolve={() => {}}
      />,
    )
    expect(screen.getByRole('button', { name: 'Continue' })).toHaveAttribute(
      'data-variant',
      'default',
    )
  })
})
