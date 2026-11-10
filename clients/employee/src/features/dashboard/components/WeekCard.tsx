import { EmptyState } from '@employee360/ui'
import type { EmployeeDashboard } from '@employee360/api-client'
import { DashboardCard } from './DashboardCard'
import { plural, weekdayShort } from './format'

export function WeekCard({ week }: { week?: EmployeeDashboard['week'] }) {
  const days = week?.days ?? []
  const missing = week?.missing_days ?? 0
  const max = Math.max(8, ...days.map((d) => d.hours ?? 0))
  return (
    <DashboardCard title="Last 7 days" to="/status" linkLabel="Log hours">
      {days.length === 0 ? (
        <EmptyState title="No hours to show" description="Your last 7 days will appear here." />
      ) : (
        <>
          {missing > 0 && (
            <p role="alert" className="rounded-sm border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-700">
              {plural(missing, 'working day')} not logged
            </p>
          )}
          <ul className="grid grid-cols-7 gap-2">
            {days.map((d) => {
              const hours = d.hours ?? 0
              const onLeave = d.kind === 'leave'
              const tag = onLeave ? 'On leave' : d.kind === 'holiday' ? 'Holiday' : d.kind === 'weekly_off' ? 'Off' : null
              return (
                <li key={d.date} data-kind={d.kind} className="flex flex-col items-center gap-1" data-testid="week-day">
                  <div className={`flex h-20 w-full items-end rounded-sm ${tag ? 'bg-blue-50' : 'bg-slate-100'}`}>
                    <div
                      className={`w-full rounded-sm ${d.is_working_day ? 'bg-slate-900' : 'bg-slate-400'}`}
                      style={{ height: `${Math.round((hours / max) * 100)}%` }}
                    />
                  </div>
                  <span className="text-xs font-medium text-ink">{hours}h</span>
                  <span className="text-xs text-ink-muted">{weekdayShort(d.date ?? '')}</span>
                  {tag && (
                    <span className="rounded-sm border border-blue-300 bg-blue-50 px-1 text-[10px] font-medium text-blue-700">{tag}</span>
                  )}
                </li>
              )
            })}
          </ul>
        </>
      )}
    </DashboardCard>
  )
}
