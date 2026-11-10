import type { ReactNode } from 'react'
import type { LeaveRequest } from '../queries/leaveQueries'
import { LeaveStatusBadge } from './LeaveStatusBadge'

const PART: Record<string, string> = { full: '', first_half: ' (first half)', second_half: ' (second half)' }

export const dateRange = (r: LeaveRequest) => (r.start_date === r.end_date ? r.start_date : `${r.start_date} to ${r.end_date}`)

interface Props {
  rows: LeaveRequest[]
  /** `mine` shows the approver; `team` shows the applicant. */
  variant: 'mine' | 'team'
  /** Per-row action area (buttons). */
  actions?: (r: LeaveRequest) => ReactNode
}

export function LeaveRequestsTable({ rows, variant, actions }: Props) {
  return (
    <div className="overflow-x-auto rounded-sm border border-slate-200 bg-white">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-100 text-xs font-medium text-slate-600">
          <tr>
            {variant === 'team' && <th className="px-3 py-2">Employee</th>}
            <th className="px-3 py-2">Type</th>
            <th className="px-3 py-2">Dates</th>
            <th className="px-3 py-2">Days</th>
            {variant === 'mine' && <th className="px-3 py-2">Approver</th>}
            <th className="px-3 py-2">Status</th>
            <th className="px-3 py-2">{variant === 'team' ? 'Reason / comment' : 'Decision comment'}</th>
            {actions && <th className="px-3 py-2">Actions</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id} className="border-t border-slate-200 align-top text-slate-900 hover:bg-slate-50">
              {variant === 'team' && <td className="px-3 py-2">{r.user_name || r.user_email}</td>}
              <td className="px-3 py-2">{r.leave_type_name}</td>
              <td className="px-3 py-2">
                {dateRange(r)}
                {PART[r.day_part]}
              </td>
              <td className="px-3 py-2">{r.days}</td>
              {variant === 'mine' && <td className="px-3 py-2">{r.approver_name}</td>}
              <td className="px-3 py-2">
                <LeaveStatusBadge status={r.status} />
              </td>
              <td className="px-3 py-2 text-slate-600">
                {variant === 'team' && r.reason && <p>{r.reason}</p>}
                {r.decision_comment ? <p>{r.decision_comment}</p> : variant === 'mine' ? '-' : null}
              </td>
              {actions && <td className="px-3 py-2">{actions(r)}</td>}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
