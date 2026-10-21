import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ResponsePanel } from '@/components/response-panel'
import type { RunResponse } from '@/lib/types'

/** A completed response with the shapes the panel cares about. */
function makeResponse(overrides: Partial<RunResponse> = {}): RunResponse {
  return {
    status: 200,
    statusText: 'OK',
    headers: { 'Content-Type': ['application/json'], 'X-Custom': ['a'] },
    body: '{"ok":true}',
    encoding: 'text',
    contentType: 'application/json; charset=utf-8',
    durationMs: 12,
    sizeBytes: 11,
    truncated: false,
    finalUrl: 'https://api.test/x',
    cookies: [],
    receivedAt: new Date('2026-01-01T00:00:00Z').toISOString(),
    historyId: 'h1',
    envName: '',
    resolvedUrl: 'https://api.test/x?q=1',
    requestHeaders: { 'Content-Type': ['application/json'], Authorization: ['Bearer t'] },
    requestUrl: 'https://api.test/x?q=1',
    requestBodyType: 'json',
    requestBody: '{"a":1}',
    remoteAddr: '10.0.0.1:443',
    httpVersion: 'HTTP/1.1',
    ...overrides,
  }
}

describe('ResponsePanel — empty and loading', () => {
  it('prompts to send when there is no response', () => {
    render(<ResponsePanel response={null} running={false} />)
    expect(screen.getByText('NO RESPONSE')).toBeInTheDocument()
    expect(screen.getByText(/Send the request/)).toBeInTheDocument()
  })

  it('shows a waiting state while running with no prior response', () => {
    render(<ResponsePanel response={null} running />)
    expect(screen.getByText(/waiting for response/)).toBeInTheDocument()
    expect(screen.queryByText('NO RESPONSE')).not.toBeInTheDocument()
  })
})

describe('ResponsePanel — failure', () => {
  it('reports the failure and its message', () => {
    render(
      <ResponsePanel
        response={makeResponse({
          status: 0,
          error: 'request timed out',
          body: '',
          headers: null,
          encoding: 'text',
        })}
        running={false}
      />,
    )
    expect(screen.getByText('REQUEST FAILED')).toBeInTheDocument()
    expect(screen.getAllByText('request timed out').length).toBeGreaterThan(0)
  })
})

describe('ResponsePanel — success', () => {
  it('shows status, duration and size', () => {
    render(<ResponsePanel response={makeResponse()} running={false} />)
    expect(screen.getByText('200')).toBeInTheDocument()
    expect(screen.getByText('OK')).toBeInTheDocument()
    expect(screen.getByText('12 ms')).toBeInTheDocument()
    expect(screen.getByText('11 B')).toBeInTheDocument()
  })

  it('marks a truncated response', () => {
    render(<ResponsePanel response={makeResponse({ truncated: true })} running={false} />)
    expect(screen.getByText('truncated')).toBeInTheDocument()
  })

  it('does not show a failure badge on success', () => {
    render(<ResponsePanel response={makeResponse()} running={false} />)
    expect(screen.queryByText('REQUEST FAILED')).not.toBeInTheDocument()
  })
})

describe('ResponsePanel — tabs', () => {
  it('defaults to Preview', () => {
    render(<ResponsePanel response={makeResponse()} running={false} />)
    const preview = screen.getByRole('tab', { name: 'Preview' })
    expect(preview).toHaveAttribute('data-state', 'active')
  })

  it('offers the four panes in a stable order', () => {
    render(<ResponsePanel response={makeResponse()} running={false} />)
    const names = screen.getAllByRole('tab').map((t) => t.textContent)
    // The request editor's tabs come first; the response tabs are the last four.
    expect(names.slice(-4).map((n) => n?.replace(/\d.*$/, '').trim())).toEqual([
      'Headers',
      'Request Body',
      'Preview',
      'Response',
    ])
  })

  it('switches panes when a tab is clicked', async () => {
    const user = userEvent.setup()
    render(<ResponsePanel response={makeResponse()} running={false} />)

    await user.click(screen.getByRole('tab', { name: /^Response$/ }))
    expect(screen.getByRole('tab', { name: /^Response$/ })).toHaveAttribute(
      'data-state',
      'active',
    )
    expect(screen.getByText(/Exactly as received/)).toBeInTheDocument()
  })

  it('lists request and response headers under Headers', async () => {
    const user = userEvent.setup()
    render(<ResponsePanel response={makeResponse()} running={false} />)

    await user.click(screen.getByRole('tab', { name: /^Headers/ }))

    expect(screen.getByText('General')).toBeInTheDocument()
    expect(screen.getByText('Response Headers')).toBeInTheDocument()
    expect(screen.getByText('Request Headers')).toBeInTheDocument()
    // The automatically added auth header must be visible.
    expect(screen.getByText('Authorization')).toBeInTheDocument()
    expect(screen.getByText('Bearer t')).toBeInTheDocument()
    // Summary facts.
    expect(screen.getByText('10.0.0.1:443')).toBeInTheDocument()
    expect(screen.getByText('HTTP/1.1')).toBeInTheDocument()
  })

  it('reports an empty request body rather than rendering nothing', async () => {
    const user = userEvent.setup()
    render(
      <ResponsePanel
        response={makeResponse({ requestBody: '', requestBodyType: 'none' })}
        running={false}
      />,
    )

    await user.click(screen.getByRole('tab', { name: /^Request Body/ }))
    expect(screen.getByText('This request sent no body.')).toBeInTheDocument()
  })

  it('lists cookies when the response set any', async () => {
    const user = userEvent.setup()
    render(
      <ResponsePanel
        response={makeResponse({ cookies: ['session=abc', 'theme=dark'] })}
        running={false}
      />,
    )

    await user.click(screen.getByRole('tab', { name: /^Headers/ }))
    expect(screen.getByText('Cookies Set')).toBeInTheDocument()
    expect(screen.getByText('session=abc')).toBeInTheDocument()
  })

  it('shows the received timestamp in the summary', async () => {
    const user = userEvent.setup()
    render(<ResponsePanel response={makeResponse()} running={false} />)

    await user.click(screen.getByRole('tab', { name: /^Headers/ }))
    expect(screen.getByText('Received')).toBeInTheDocument()
  })

  it('reveals the final URL when a redirect changed it', async () => {
    const user = userEvent.setup()
    render(
      <ResponsePanel
        response={makeResponse({
          requestUrl: 'https://api.test/old',
          finalUrl: 'https://api.test/new',
        })}
        running={false}
      />,
    )

    await user.click(screen.getByRole('tab', { name: /^Headers/ }))
    expect(screen.getByText('Final URL')).toBeInTheDocument()
    expect(screen.getByText('https://api.test/new')).toBeInTheDocument()
  })

  it('shows per-phase timing when available', async () => {
    const user = userEvent.setup()
    render(
      <ResponsePanel
        response={makeResponse({
          durationMs: 200,
          timing: { dnsMs: 5, connectMs: 12, tlsMs: 40, ttfbMs: 150, downloadMs: 50, reused: false },
        })}
        running={false}
      />,
    )

    await user.click(screen.getByRole('tab', { name: /^Headers/ }))

    expect(screen.getByText('Timing')).toBeInTheDocument()
    expect(screen.getByText('DNS lookup')).toBeInTheDocument()
    expect(screen.getByText('TLS handshake')).toBeInTheDocument()
    expect(screen.getByText('Waiting (TTFB)')).toBeInTheDocument()
    // "Download" also appears on the download button, so scope to the label cell.
    expect(
      screen.getAllByText('Download', { selector: 'span.truncate' }).length,
    ).toBeGreaterThan(0)
    // Values are rendered next to each phase.
    expect(screen.getByText('40 ms')).toBeInTheDocument()
    expect(screen.getByText('150 ms')).toBeInTheDocument()
  })

  it('marks a reused connection instead of implying zero-cost TLS', async () => {
    const user = userEvent.setup()
    render(
      <ResponsePanel
        response={makeResponse({
          timing: { dnsMs: 0, connectMs: 0, tlsMs: 0, ttfbMs: 3, downloadMs: 1, reused: true },
        })}
        running={false}
      />,
    )

    await user.click(screen.getByRole('tab', { name: /^Headers/ }))
    expect(screen.getByText(/connection reused/)).toBeInTheDocument()
  })

  it('omits the timing group when no timing was recorded', async () => {
    const user = userEvent.setup()
    render(<ResponsePanel response={makeResponse({ timing: null })} running={false} />)

    await user.click(screen.getByRole('tab', { name: /^Headers/ }))
    expect(screen.queryByText('Timing')).not.toBeInTheDocument()
  })
})
