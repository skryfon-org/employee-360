import { Link } from 'react-router-dom'
import { PageHeader, formatHolidayDate } from '@employee360/ui'
import type { EmployeeDashboard } from '@employee360/api-client'
import { greetingFor } from './format'

const primaryBtn =
  'inline-flex h-9 items-center rounded-sm bg-slate-900 px-4 text-sm font-medium text-white hover:bg-slate-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'
const secondaryBtn =
  'inline-flex h-9 items-center rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

function dayChip(today: NonNullable<EmployeeDashboard['greeting']>['today']): { label: string; box: string } {
  switch (today?.kind) {
    case 'holiday':
      return { label: today.holiday_name ? `Holiday: ${today.holiday_name}` : 'Holiday', box: 'border-amber-300 bg-amber-50 text-amber-700' }
    case 'weekly_off':
      return { label: 'Weekly off', box: 'border-slate-300 bg-slate-100 text-slate-600' }
    case 'leave':
      return { label: 'On leave today', box: 'border-blue-300 bg-blue-50 text-blue-700' }
    default:
      return { label: 'Working day', box: 'border-green-300 bg-green-50 text-green-700' }
  }
}

export function DashboardHeader({ greeting }: { greeting?: EmployeeDashboard['greeting'] }) {
  const name = greeting?.first_name
  const title = `${greetingFor(new Date().getHours())}${name ? `, ${name}` : ''}`
  const today = greeting?.today
  const chip = dayChip(today)
  return (
    <div className="flex flex-col gap-3">
      <PageHeader
        title={title}
        description={today?.date ? formatHolidayDate(today.date) : undefined}
        actions={
          <>
            <Link to="/leave" className={primaryBtn}>Apply leave</Link>
            <Link to="/status" className={secondaryBtn}>Log hours</Link>
          </>
        }
      />
      <div className="flex flex-wrap items-center gap-3">
        {today && (
          <span data-testid="day-chip" className={`inline-flex h-6 items-center rounded-sm border px-2 text-xs font-medium ${chip.box}`}>
            {chip.label}
          </span>
        )}
        {greeting?.welcome_message && <p className="text-sm text-ink-muted">{greeting.welcome_message}</p>}
      </div>
    </div>
  )
}
