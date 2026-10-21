export const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'] as const

export const methodColor: Record<string, string> = {
  GET: 'text-m-get',
  POST: 'text-m-post',
  PUT: 'text-m-put',
  PATCH: 'text-m-patch',
  DELETE: 'text-m-delete',
}

export const methodBg: Record<string, string> = {
  GET: 'bg-m-get/10 text-m-get',
  POST: 'bg-m-post/12 text-m-post',
  PUT: 'bg-m-put/12 text-m-put',
  PATCH: 'bg-m-patch/12 text-m-patch',
  DELETE: 'bg-m-delete/12 text-m-delete',
}

export function methodTint(method: string): string {
  return methodBg[method?.toUpperCase()] ?? 'bg-m-other/10 text-m-other'
}

export function statusTint(status: number): string {
  if (status === 0) return 'text-muted-foreground'
  if (status >= 200 && status < 300) return 'text-ok'
  if (status >= 300 && status < 400) return 'text-warn'
  return 'text-bad'
}

export function formatBytes(n: number): string {
  if (!n) return '0 B'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

/**
 * Relative time, in the caller's language. Defaults to English so existing
 * call sites and tests keep working.
 */
export function timeAgo(iso: string, lang: 'en' | 'zh' = 'en'): string {
  if (!iso) return ''
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const diff = Date.now() - then
  const sec = Math.floor(diff / 1000)
  const zh = lang === 'zh'
  if (sec < 60) return zh ? '刚刚' : 'just now'
  const min = Math.floor(sec / 60)
  if (min < 60) return zh ? `${min} 分钟前` : `${min}m ago`
  const hr = Math.floor(min / 60)
  if (hr < 24) return zh ? `${hr} 小时前` : `${hr}h ago`
  const day = Math.floor(hr / 24)
  if (day < 30) return zh ? `${day} 天前` : `${day}d ago`
  return new Date(iso).toLocaleDateString()
}

/** Turn a URL into a short, human label for the request list. */
export function pathOf(url: string): string {
  const raw = url.trim()
  if (!raw) return 'no URL'
  try {
    const u = new URL(raw.includes('://') ? raw : `http://${raw}`)
    return u.pathname === '/' ? u.host : u.pathname
  } catch {
    return raw
  }
}

export function prettyJSON(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

export function slugify(input: string): string {
  return input
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9-_.]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 80)
}
