import type { DailyEmployeeSummary } from '../queries/statusLogQueries'

export type DayStatus = NonNullable<DailyEmployeeSummary['day_status']>

const TONES = {
  success: { box: 'bg-green-50 border-green-300 text-green-700', dot: 'bg-green-700' },
  warning: { box: 'bg-amber-50 border-amber-300 text-amber-700', dot: 'bg-amber-700' },
  neutral: { box: 'bg-slate-100 border-slate-300 text-slate-700', dot: 'bg-slate-600' },
} as const

function Badge({ tone, children }: { tone: keyof typeof TONES; children: string }) {
  const t = TONES[tone]
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 whitespace-nowrap rounded-sm border px-2 text-xs font-medium ${t.box}`}>
      <span aria-hidden="true" className={`h-1.5 w-1.5 shrink-0 rounded-full ${t.dot}`} />
      {children}
    </span>
  )
}

/** Daily status. Only "Missing" is a warning; leave/holiday/weekly off are neutral. Dot plus text so color is never the only signal. */
export function DayStatusBadge({ status, halfDay }: { status?: DayStatus; halfDay?: boolean }) {
  switch (status) {
    case 'logged':
      return <Badge tone="success">Logged</Badge>
    case 'on_leave':
      return <Badge tone="neutral">On leave</Badge>
    case 'holiday':
      return <Badge tone="neutral">Holiday</Badge>
    case 'weekly_off':
      return <Badge tone="neutral">Weekly off</Badge>
    default:
      return <Badge tone="warning">{halfDay ? 'Missing (half-day leave)' : 'Missing'}</Badge>
  }
}

/** Monthly status: warning only when logs are missing; "Nothing expected" when no working days were expected. */
export function MonthStatusBadge({ missing, expected }: { missing: number; expected: number }) {
  if (missing > 0) return <Badge tone="warning">Missing logs</Badge>
  return expected > 0 ? <Badge tone="success">Up to date</Badge> : <Badge tone="neutral">Nothing expected</Badge>
}
