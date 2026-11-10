export interface Crumb {
  /** Link target; the last crumb is rendered as plain text regardless. */
  to: string
  label: string
}

/** A label, or a function of the matched `:params` (for dynamic segments, e.g. a looked-up name). */
export type CrumbLabel = string | ((params: Record<string, string>) => string)

/** Route pattern (`/invitations/:id`) -> label. Patterns match segment by segment. */
export type CrumbConfig = Record<string, CrumbLabel>

const segmentsOf = (path: string) => path.split('/').filter(Boolean)

function safeDecode(s: string) {
  try {
    return decodeURIComponent(s)
  } catch {
    return s
  }
}

/** "invitation-details" / "invitation_details" -> "Invitation details". */
export function humanise(segment: string): string {
  const text = safeDecode(segment).replace(/[-_]+/g, ' ').trim()
  return text ? text.charAt(0).toUpperCase() + text.slice(1) : segment
}

function match(pattern: string, parts: string[]): Record<string, string> | null {
  const pat = segmentsOf(pattern)
  if (pat.length !== parts.length) return null
  const params: Record<string, string> = {}
  for (let i = 0; i < pat.length; i++) {
    if (pat[i].startsWith(':')) params[pat[i].slice(1)] = safeDecode(parts[i])
    else if (pat[i] !== parts[i]) return null
  }
  return params
}

function labelFor(prefix: string[], config: CrumbConfig): string {
  for (const [pattern, label] of Object.entries(config)) {
    const params = match(pattern, prefix)
    if (params) return typeof label === 'function' ? label(params) : label
  }
  return humanise(prefix[prefix.length - 1])
}

/**
 * Builds the trail for `pathname`: the root crumb (`config['/']`, "Dashboard")
 * followed by one crumb per path segment, each labelled via `config` with a
 * humanised-segment fallback. The root itself (and any `rootAliases`) yields
 * just the single root crumb.
 */
export function buildBreadcrumbs(pathname: string, config: CrumbConfig, rootAliases: string[] = []): Crumb[] {
  const root: Crumb = { to: '/', label: labelFor([], { '/': 'Home', ...config }) }
  const parts = segmentsOf(pathname)
  if (parts.length === 0 || rootAliases.includes('/' + parts.join('/'))) return [root]
  const trail = parts.map((_, i) => {
    const prefix = parts.slice(0, i + 1)
    return { to: '/' + prefix.join('/'), label: labelFor(prefix, config) }
  })
  return [root, ...trail]
}
