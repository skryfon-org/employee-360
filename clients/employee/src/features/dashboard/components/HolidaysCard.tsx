import { EmptyState } from '@employee360/ui'
import type { EmployeeDashboard } from '@employee360/api-client'
import { DashboardCard } from './DashboardCard'
import { shortDate } from './format'

function inDays(n: number): string {
  if (n <= 0) return 'Today'
  if (n === 1) return 'Tomorrow'
  return `in ${n} days`
}

export function HolidaysCard({ holidays }: { holidays?: EmployeeDashboard['holidays'] }) {
  const items = holidays ?? []
  return (
    <DashboardCard title="Upcoming holidays" to="/holidays" linkLabel="View calendar">
      {items.length === 0 ? (
        <EmptyState title="No upcoming holidays" description="Nothing is scheduled yet." />
      ) : (
        <ul className="divide-y divide-line">
          {items.map((h) => (
            <li key={h.id} className="flex items-center justify-between gap-3 py-2">
              <div className="flex min-w-0 items-center gap-3">
                <span aria-hidden="true" className="h-2.5 w-2.5 shrink-0 rounded-full border border-slate-300" style={{ backgroundColor: h.category_color }} />
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-ink">{h.name}</p>
                  <p className="text-xs text-ink-muted">{shortDate(h.date ?? '')}{h.category_name ? ` · ${h.category_name}` : ''}</p>
                </div>
              </div>
              <span className="shrink-0 text-xs text-ink-muted">{inDays(h.days_until ?? 0)}</span>
            </li>
          ))}
        </ul>
      )}
    </DashboardCard>
  )
}
