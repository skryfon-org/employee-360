const DATE = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })

/** "2026-10-01T..." -> "1 Oct 2026"; "-" when missing or unparseable. */
export function formatCreated(iso?: string): string {
  if (!iso) return '-'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '-' : DATE.format(d)
}
