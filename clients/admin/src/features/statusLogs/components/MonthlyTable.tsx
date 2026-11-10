import { useState } from 'react'
import { InfoIcon } from '@employee360/ui'
import { Link } from 'react-router-dom'
import { formatHours, monthRange } from '../dates'
import { MonthStatusBadge } from './StatusBadges'
import { MonthlyDetailsModal, type MonthlyDetails } from './MonthlyDetailsModal'
import type { MonthlyEmployeeSummary } from '../queries/statusLogQueries'

const nameOf = (r: MonthlyEmployeeSummary) => [r.first_name, r.last_name].filter(Boolean).join(' ') || r.email || 'Employee'

/** Per-employee month totals; each name links to their status-logs tab prefilled to the month. */
export function MonthlyTable({ rows, month, workingDays }: { rows: MonthlyEmployeeSummary[]; month: string; workingDays: number }) {
  const { from, to } = monthRange(month)
  const [details, setDetails] = useState<MonthlyDetails | null>(null)
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[44rem] text-left">
        <thead className="bg-slate-100 text-xs font-medium text-slate-600">
          <tr>
            <th scope="col" className="px-4 py-3">Employee</th>
            <th scope="col" className="px-4 py-3">Department</th>
            <th scope="col" className="px-4 py-3">Status</th>
            <th scope="col" className="px-4 py-3">Logged / expected</th>
            <th scope="col" className="px-4 py-3">Total hours</th>
            <th scope="col" className="px-4 py-3">Hours by project</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => {
            const daysLogged = r.days_logged ?? 0
            const expected = r.expected_days ?? workingDays
            const missing = r.missing_days ?? 0
            const name = nameOf(r)
            const rowDetails: MonthlyDetails = {
              name,
              logged: daysLogged,
              expected,
              missing,
              leave: r.leave_days ?? 0,
              offDays: r.off_days_worked ?? 0,
            }
            const projects = r.hours_by_project ?? []
            return (
              <tr key={r.user_id} className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50">
                <td className="px-4 py-3">
                  <Link
                    to={`/employees/${r.user_id}/status-logs?from=${from}&to=${to}`}
                    className="font-medium text-slate-900 hover:text-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
                  >
                    {name}
                  </Link>
                  <div className="text-xs text-slate-600">{r.email}</div>
                </td>
                <td className="px-4 py-3">{r.department_name || <span className="text-slate-600">-</span>}</td>
                <td className="px-4 py-3"><MonthStatusBadge missing={missing} expected={expected} /></td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    <span className={missing > 0 ? 'font-semibold text-amber-700' : undefined}>
                      {daysLogged}/{expected} logged
                    </span>
                    <button
                      type="button"
                      aria-label={`View details for ${name}`}
                      title="View details"
                      onClick={() => setDetails(rowDetails)}
                      className="flex h-6 w-6 shrink-0 items-center justify-center rounded-sm text-slate-600 hover:bg-slate-100 hover:text-slate-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900"
                    >
                      <InfoIcon className="h-4 w-4" />
                    </button>
                  </div>
                </td>
                <td className="px-4 py-3">{formatHours(r.total_hours)}</td>
                <td className="px-4 py-3">
                  {projects.length === 0 ? (
                    <span className="text-slate-600">-</span>
                  ) : (
                    <ul className="flex flex-wrap gap-1">
                      {projects.map((p) => (
                        <li key={p.project_id} className="rounded-sm border border-slate-300 bg-slate-100 px-2 py-0.5 text-xs text-slate-900">
                          {p.project_code || p.project_name}: {formatHours(p.hours)}
                        </li>
                      ))}
                    </ul>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      {details && <MonthlyDetailsModal details={details} onClose={() => setDetails(null)} />}
    </div>
  )
}
