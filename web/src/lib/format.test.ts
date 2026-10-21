import { describe, expect, it } from 'vitest'
import {
  formatBytes,
  formatDuration,
  methodTint,
  pathOf,
  prettyJSON,
  slugify,
  statusTint,
  timeAgo,
} from '@/lib/format'

describe('formatBytes', () => {
  it('handles zero and sub-kilobyte values', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1)).toBe('1 B')
    expect(formatBytes(1023)).toBe('1023 B')
  })

  it('switches to KB at 1024 and MB at 1 MiB', () => {
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(1024 * 1024)).toBe('1.00 MB')
    expect(formatBytes(2.5 * 1024 * 1024)).toBe('2.50 MB')
  })
})

describe('formatDuration', () => {
  it('uses milliseconds below one second', () => {
    expect(formatDuration(0)).toBe('0 ms')
    expect(formatDuration(999)).toBe('999 ms')
  })

  it('uses seconds from 1000ms', () => {
    expect(formatDuration(1000)).toBe('1.00 s')
    expect(formatDuration(2500)).toBe('2.50 s')
  })
})

describe('statusTint', () => {
  it('maps each class of status to a distinct token', () => {
    expect(statusTint(0)).toBe('text-muted-foreground') // no response
    expect(statusTint(200)).toBe('text-ok')
    expect(statusTint(299)).toBe('text-ok')
    expect(statusTint(301)).toBe('text-warn')
    expect(statusTint(304)).toBe('text-warn')
    expect(statusTint(404)).toBe('text-bad')
    expect(statusTint(500)).toBe('text-bad')
  })
})

describe('methodTint', () => {
  it('is case-insensitive and falls back for unknown verbs', () => {
    expect(methodTint('get')).toBe(methodTint('GET'))
    expect(methodTint('GET')).toContain('m-get')
    expect(methodTint('TRACE')).toContain('m-other')
    expect(methodTint('')).toContain('m-other')
  })
})

describe('pathOf', () => {
  it('returns the path for a full URL', () => {
    expect(pathOf('https://api.test/v1/users?page=2')).toBe('/v1/users')
  })

  it('returns the host when the path is the root', () => {
    expect(pathOf('https://api.test/')).toBe('api.test')
    expect(pathOf('https://api.test')).toBe('api.test')
  })

  it('accepts a URL without a scheme', () => {
    expect(pathOf('api.test/v2/items')).toBe('/v2/items')
  })

  it('degrades gracefully', () => {
    expect(pathOf('')).toBe('no URL')
    expect(pathOf('   ')).toBe('no URL')
    expect(pathOf('{{baseUrl}}/x')).toBe('/x') // resolves as host-less path
  })
})

describe('prettyJSON', () => {
  it('indents valid JSON', () => {
    expect(prettyJSON('{"a":1}')).toBe('{\n  "a": 1\n}')
    expect(prettyJSON('[1,2]')).toBe('[\n  1,\n  2\n]')
  })

  it('returns the input untouched when it is not JSON', () => {
    expect(prettyJSON('not json')).toBe('not json')
    expect(prettyJSON('')).toBe('')
  })
})

describe('slugify', () => {
  it('lowercases and collapses separators', () => {
    expect(slugify('Daily Exchange Rates')).toBe('daily-exchange-rates')
    expect(slugify('  a   b  ')).toBe('a-b')
  })

  it('keeps dots, dashes and underscores', () => {
    expect(slugify('v1.2_beta-x')).toBe('v1.2_beta-x')
  })

  it('drops characters that are unsafe in a path', () => {
    expect(slugify('a/b?c=d&e')).toBe('a-b-c-d-e')
    expect(slugify('中文')).toBe('')
  })

  it('truncates to 80 characters', () => {
    expect(slugify('a'.repeat(200))).toHaveLength(80)
  })
})

describe('timeAgo', () => {
  const ago = (ms: number) => new Date(Date.now() - ms).toISOString()

  it('formats recent times', () => {
    expect(timeAgo(ago(5_000))).toBe('just now')
    expect(timeAgo(ago(5 * 60_000))).toBe('5m ago')
    expect(timeAgo(ago(3 * 3600_000))).toBe('3h ago')
    expect(timeAgo(ago(2 * 86400_000))).toBe('2d ago')
  })

  it('returns empty for missing or invalid input', () => {
    expect(timeAgo('')).toBe('')
    expect(timeAgo('not-a-date')).toBe('')
  })

  it('formats in Chinese when asked', () => {
    expect(timeAgo(ago(5_000), 'zh')).toBe('刚刚')
    expect(timeAgo(ago(5 * 60_000), 'zh')).toBe('5 分钟前')
    expect(timeAgo(ago(3 * 3600_000), 'zh')).toBe('3 小时前')
    expect(timeAgo(ago(2 * 86400_000), 'zh')).toBe('2 天前')
  })

  it('defaults to English so existing call sites are unaffected', () => {
    expect(timeAgo(ago(5 * 60_000))).toBe('5m ago')
  })
})
