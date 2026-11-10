import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Card } from '@employee360/ui'

/** Titled dashboard section with an optional "view all" link. */
export function DashboardCard({
  title,
  to,
  linkLabel,
  children,
}: {
  title: string
  to?: string
  linkLabel?: string
  children: ReactNode
}) {
  return (
    <Card className="flex min-w-0 flex-col gap-4" aria-label={title} role="region">
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-base font-semibold text-ink">{title}</h2>
        {to && (
          <Link
            to={to}
            className="text-sm font-medium text-ink-muted hover:text-ink focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            {linkLabel ?? 'View all'}
          </Link>
        )}
      </div>
      {children}
    </Card>
  )
}
