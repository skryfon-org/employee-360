const ROLE_PRIORITY = ['super_admin', 'admin', 'employee']

/** Highest-privilege role from a list (unknown roles rank last); undefined when empty. */
export function pickPrimaryRole(roles?: string[]): string | undefined {
  if (!roles?.length) return undefined
  const rank = (r: string) => {
    const i = ROLE_PRIORITY.indexOf(r)
    return i === -1 ? ROLE_PRIORITY.length : i
  }
  return [...roles].sort((a, b) => rank(a) - rank(b))[0]
}

/** "super_admin" -> "Super Admin". */
export function humanizeRole(role: string): string {
  return role
    .split(/[_\-\s]+/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join(' ')
}

export function UserBadge({
  email,
  name,
  role,
  variant = 'header',
  collapsed = false,
}: {
  email?: string
  /** Display name; falls back to the email when blank. */
  name?: string
  role?: string
  /** `sidebar` always shows the email; `collapsed` (sidebar only) hides the text from md up, leaving the avatar. */
  variant?: 'header' | 'sidebar'
  collapsed?: boolean
}) {
  if (!email) return null
  const display = name?.trim() || email
  const roleLabel = role ? humanizeRole(role) : ''
  const sidebar = variant === 'sidebar'
  const emailClass = sidebar ? 'block max-w-full' : 'hidden max-w-[10rem] sm:block lg:max-w-[16rem]'
  const textClass = sidebar && collapsed ? 'md:hidden' : ''
  return (
    <span className="flex min-w-0 items-center gap-2.5" title={roleLabel ? `${display} - ${email} (${roleLabel})` : display === email ? email : `${display} - ${email}`}>
      <span
        aria-hidden="true"
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-sm border border-line-strong bg-surface-subtle text-xs font-semibold uppercase text-ink"
      >
        {display.charAt(0)}
      </span>
      <span className={`flex min-w-0 flex-col items-start gap-0.5 ${textClass}`}>
        <span className={`truncate text-sm leading-4 text-ink ${emailClass}`}>{display}</span>
        {roleLabel && (
          <span className="inline-flex h-4 items-center rounded-sm border border-slate-300 bg-slate-100 px-1.5 text-[11px] font-medium uppercase leading-none tracking-wide text-slate-700">
            {roleLabel}
          </span>
        )}
      </span>
    </span>
  )
}
