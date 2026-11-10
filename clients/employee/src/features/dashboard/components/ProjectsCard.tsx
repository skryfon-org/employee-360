import { EmptyState } from '@employee360/ui'
import type { EmployeeDashboard } from '@employee360/api-client'
import { DashboardCard } from './DashboardCard'
import { shortDate } from './format'

export function ProjectsCard({ projects }: { projects?: EmployeeDashboard['projects'] }) {
  const items = projects ?? []
  return (
    <DashboardCard title="My projects" to="/projects" linkLabel="View projects">
      {items.length === 0 ? (
        <EmptyState title="No active projects" description="Projects you are allocated to will appear here." />
      ) : (
        <ul className="divide-y divide-line">
          {items.map((p) => (
            <li key={p.project_id} className="flex items-center justify-between gap-3 py-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-ink">{p.project_name}</p>
                <p className="truncate text-xs text-ink-muted">
                  {[p.project_code, p.role_on_project, p.end_date ? `until ${shortDate(p.end_date)}` : 'ongoing'].filter(Boolean).join(' · ')}
                </p>
              </div>
              <span className="shrink-0 rounded-sm border border-slate-300 bg-slate-100 px-2 py-0.5 text-xs font-medium text-ink">
                {p.allocation_percent ?? 0}%
              </span>
            </li>
          ))}
        </ul>
      )}
    </DashboardCard>
  )
}
