import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { UnlockScreen } from './unlock-screen'
import { useLanguageStore } from '@/lib/i18n'

// The component calls the real session API; stub it so the test can drive the
// success and failure paths without a server.
const signIn = vi.fn()
vi.mock('@/lib/api', () => ({
  session: {
    signIn: (...args: unknown[]) => signIn(...args),
  },
}))

beforeEach(() => {
  signIn.mockReset()
  useLanguageStore.setState({ language: 'en' })
})

describe('UnlockScreen', () => {
  it('explains where the token comes from', () => {
    render(<UnlockScreen onUnlocked={() => {}} />)
    expect(screen.getByText('This workspace is locked')).toBeInTheDocument()
    // The hint matters: without it the screen is a dead end for anyone who
    // closed the terminal.
    expect(screen.getByText(/printed once in the terminal/i)).toBeInTheDocument()
  })

  it('keeps the submit button disabled until something is typed', async () => {
    const user = userEvent.setup()
    render(<UnlockScreen onUnlocked={() => {}} />)

    const submit = screen.getByRole('button', { name: 'Unlock' })
    expect(submit).toBeDisabled()

    await user.type(screen.getByLabelText('API token'), 'abc')
    expect(submit).toBeEnabled()
  })

  it('calls onUnlocked after a successful sign-in', async () => {
    signIn.mockResolvedValue({ ok: true })
    const onUnlocked = vi.fn()
    const user = userEvent.setup()

    render(<UnlockScreen onUnlocked={onUnlocked} />)
    await user.type(screen.getByLabelText('API token'), 'good-token')
    await user.click(screen.getByRole('button', { name: 'Unlock' }))

    await waitFor(() => expect(onUnlocked).toHaveBeenCalled())
    expect(signIn).toHaveBeenCalledWith('good-token')
  })

  it('shows an inline error rather than a toast when the token is wrong', async () => {
    // A toast would vanish and leave the user staring at a form with no idea
    // what happened; the message belongs next to the field.
    signIn.mockRejectedValue(new Error('invalid token'))
    const onUnlocked = vi.fn()
    const user = userEvent.setup()

    render(<UnlockScreen onUnlocked={onUnlocked} />)
    await user.type(screen.getByLabelText('API token'), 'bad-token')
    await user.click(screen.getByRole('button', { name: 'Unlock' }))

    expect(await screen.findByText('That token was not accepted.')).toBeInTheDocument()
    expect(onUnlocked).not.toHaveBeenCalled()
  })

  it('clears the error when the user edits the field again', async () => {
    signIn.mockRejectedValue(new Error('invalid token'))
    const user = userEvent.setup()

    render(<UnlockScreen onUnlocked={() => {}} />)
    const field = screen.getByLabelText('API token')
    await user.type(field, 'bad')
    await user.click(screen.getByRole('button', { name: 'Unlock' }))
    await screen.findByText('That token was not accepted.')

    await user.type(field, 'x')
    expect(screen.queryByText('That token was not accepted.')).not.toBeInTheDocument()
  })

  it('trims whitespace so a pasted token with a trailing newline works', async () => {
    signIn.mockResolvedValue({ ok: true })
    const user = userEvent.setup()

    render(<UnlockScreen onUnlocked={() => {}} />)
    await user.type(screen.getByLabelText('API token'), '  token-with-spaces  ')
    await user.click(screen.getByRole('button', { name: 'Unlock' }))

    await waitFor(() => expect(signIn).toHaveBeenCalledWith('token-with-spaces'))
  })

  it('renders in Chinese when that is the chosen language', () => {
    useLanguageStore.setState({ language: 'zh' })
    render(<UnlockScreen onUnlocked={() => {}} />)
    expect(screen.getByText('工作区已锁定')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '解锁' })).toBeInTheDocument()
  })
})
