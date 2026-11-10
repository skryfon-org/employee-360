import { humanize } from '../schemas/employeeSchemas'

/** Account status: Active (green) / Inactive (neutral). */
export function EmployeeStatusBadge({ active }: { active?: boolean }) {
  const cls = active ? 'bg-green-50 border-green-300 text-green-700' : 'bg-slate-100 border-slate-300 text-slate-600'
  const dot = active ? 'bg-green-700' : 'bg-slate-600'
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${cls}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${dot}`} />
      {active ? 'Active' : 'Inactive'}
    </span>
  )
}

const EMPLOYMENT_TONE: Record<string, string> = {
  active: 'bg-green-50 border-green-300 text-green-700',
  probation: 'bg-amber-50 border-amber-300 text-amber-700',
  on_leave: 'bg-amber-50 border-amber-300 text-amber-700',
  notice_period: 'bg-amber-50 border-amber-300 text-amber-700',
  resigned: 'bg-slate-100 border-slate-300 text-slate-600',
  terminated: 'bg-red-50 border-red-200 text-red-700',
}

/** Employment lifecycle status (probation, on leave, ...). */
export function EmploymentStatusBadge({ status }: { status?: string }) {
  if (!status) return <span className="text-slate-600">-</span>
  return (
    <span
      className={`inline-flex h-6 items-center rounded-sm border px-2 text-xs font-medium ${EMPLOYMENT_TONE[status] ?? 'bg-slate-100 border-slate-300 text-slate-600'}`}
    >
      {humanize(status)}
    </span>
  )
}
