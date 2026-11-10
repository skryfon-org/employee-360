import { EmptyState } from '@employee360/ui'
import type { EmployeeDashboard, EmployeeDashboardLeaveItem } from '@employee360/api-client'
import { LeaveStatusBadge } from '../../leave/components/LeaveStatusBadge'
import type { LeaveStatus } from '../../leave/queries/leaveQueries'
import { DashboardCard } from './DashboardCard'
import { dateRange, plural } from './format'

function LeaveRow({ item }: { item: EmployeeDashboardLeaveItem }) {
  return (
    <li className="flex items-center justify-between gap-3 py-2">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-ink">{item.leave_type_name}</p>
        <p className="text-xs text-ink-muted">{dateRange(item.start_date, item.end_date)} · {plural(item.days ?? 0, 'day')}</p>
      </div>
      <LeaveStatusBadge status={(item.status ?? 'pending') as LeaveStatus} />
    </li>
  )
}

export function LeaveOverviewCard({ leave }: { leave?: EmployeeDashboard['leave'] }) {
  const next = leave?.next_approved
  const recent = leave?.recent ?? []
  return (
    <DashboardCard title="Leave overview" to="/leave" linkLabel="View leave">
      {!next && recent.length === 0 ? (
        <EmptyState title="No leave requests yet" description="Your leave requests will appear here." />
      ) : (
        <>
          {next && (
            <div data-testid="next-leave" className="rounded-sm border border-line bg-slate-100 p-3">
              <p className="text-xs font-medium text-ink-muted">Next approved leave</p>
              <p className="mt-1 text-sm font-medium text-ink">{next.leave_type_name}</p>
              <p className="text-xs text-ink-muted">{dateRange(next.start_date, next.end_date)} · {plural(next.days ?? 0, 'day')}</p>
            </div>
          )}
          {recent.length > 0 && (
            <div>
              <p className="text-xs font-medium text-ink-muted">Recent requests</p>
              <ul className="divide-y divide-line">{recent.map((r) => <LeaveRow key={r.id} item={r} />)}</ul>
            </div>
          )}
        </>
      )}
    </DashboardCard>
  )
}
