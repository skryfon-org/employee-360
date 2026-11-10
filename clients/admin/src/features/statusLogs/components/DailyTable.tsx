import { useState } from 'react'
import { Link } from 'react-router-dom'
import { formatHours, longDate } from '../dates'
import { DayEntries } from './DayEntries'
import { EntriesModal } from './EntriesModal'
import { DayStatusBadge } from './StatusBadges'
import type { DailyEmployeeSummary } from '../queries/statusLogQueries'

const FOCUS = 'focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'
const nameOf = (r: DailyEmployeeSummary) => [r.first_name, r.last_name].filter(Boolean).join(' ') || r.email || 'Employee'

function ProjectHours({ row }: { row: DailyEmployeeSummary }) {
  const byProject = row.hours_by_project ?? []
  const byCategory = row.hours_by_category ?? []
  if (!row.logged) return <span className="text-slate-600">-</span>
  if (byProject.length === 0 && byCategory.length === 0) return <span className="text-slate-600">-</span>
  return (
    <ul className="flex flex-wrap gap-1">
      {byProject.map((p) => (
        <li key={p.project_id} className="rounded-sm border border-slate-300 bg-slate-100 px-2 py-0.5 text-xs text-slate-900">
          {p.project_code || p.project_name}: {formatHours(p.hours)}
        </li>
      ))}
      {byProject.length === 0 &&
        byCategory.map((c) => (
          <li key={c.category} className="rounded-sm border border-slate-300 bg-slate-100 px-2 py-0.5 text-xs text-slate-900">
            {c.category}: {formatHours(c.hours)}
          </li>
        ))}
    </ul>
  )
}

/** Employees for the selected date; rows with entries open the day's entries and notes in a modal. */
export function DailyTable({ rows, date }: { rows: DailyEmployeeSummary[]; date: string }) {
  const [open, setOpen] = useState<{ id: string; name: string } | null>(null)
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[40rem] text-left">
        <thead className="bg-slate-100 text-xs font-medium text-slate-600">
          <tr>
            <th scope="col" className="px-4 py-3">Employee</th>
            <th scope="col" className="px-4 py-3">Department</th>
            <th scope="col" className="px-4 py-3">Status</th>
            <th scope="col" className="px-4 py-3">Total hours</th>
            <th scope="col" className="px-4 py-3">Hours by project</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => {
            const id = r.user_id as string
            const name = nameOf(r)
            return (
              <tr key={id} className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50">
                <td className="px-4 py-3">
                  <Link to={`/employees/${id}/status-logs`} className="font-medium text-slate-900 hover:text-slate-700">{name}</Link>
                  <div className="text-xs text-slate-600">{r.email}</div>
                </td>
                <td className="px-4 py-3">{r.department_name || <span className="text-slate-600">-</span>}</td>
                <td className="px-4 py-3"><DayStatusBadge status={r.day_status} halfDay={r.half_day} /></td>
                <td className="px-4 py-3">{formatHours(r.total_hours)}</td>
                <td className="px-4 py-3">
                  <div className="flex items-center justify-between gap-2">
                    <ProjectHours row={r} />
                    {r.logged && (
                      <button
                        type="button"
                        aria-haspopup="dialog"
                        aria-label={`View entries for ${name}`}
                        onClick={() => setOpen({ id, name })}
                        className={`h-8 shrink-0 rounded-sm border border-slate-300 bg-white px-2 text-xs font-medium text-slate-800 hover:bg-slate-100 ${FOCUS}`}
                      >
                        Entries
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      {open && (
        <EntriesModal title={open.name} subtitle={longDate(date)} onClose={() => setOpen(null)}>
          <DayEntries employeeId={open.id} date={date} />
        </EntriesModal>
      )}
    </div>
  )
}
