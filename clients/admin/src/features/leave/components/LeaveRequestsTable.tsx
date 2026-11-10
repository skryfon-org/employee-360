import { useState } from 'react'
import { useAuthStore } from '../../../stores/authStore'
import type { LeaveRequest } from '../queries/leaveQueries'
import { DecisionDialog } from './DecisionDialog'
import { LeaveRequestDetailModal } from './LeaveRequestDetailModal'
import { LeaveStatusBadge } from './LeaveStatusBadge'
import { ROW_BTN } from './styles'

const PART: Record<string, string> = { full: '', first_half: ' (first half)', second_half: ' (second half)' }

export const dateRange = (r: LeaveRequest) => (r.start_date === r.end_date ? r.start_date : `${r.start_date} to ${r.end_date}`)

/**
 * Admin requests table. The View modal shows the full reason; Approve/reject appear there only for pending requests that are not the viewer's own
 * (the server rejects self-decisions with 403 regardless).
 */
export function LeaveRequestsTable({ rows, showEmployee = true }: { rows: LeaveRequest[]; showEmployee?: boolean }) {
  const viewerId = useAuthStore((s) => s.user?.id)
  const [viewing, setViewing] = useState<LeaveRequest | null>(null)
  const [target, setTarget] = useState<{ request: LeaveRequest; decision: 'approve' | 'reject' } | null>(null)

  return (
    <>
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-100 text-xs font-medium text-slate-600">
            <tr>
              {showEmployee && <th scope="col" className="px-3 py-2">Employee</th>}
              <th scope="col" className="px-3 py-2">Type</th>
              <th scope="col" className="px-3 py-2">Dates</th>
              <th scope="col" className="px-3 py-2">Days</th>
              <th scope="col" className="px-3 py-2">Approver</th>
              <th scope="col" className="px-3 py-2">Decided by</th>
              <th scope="col" className="px-3 py-2">Status</th>
              <th scope="col" className="px-3 py-2"><span className="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => {
              const who = r.user_name || r.user_email
              return (
                <tr key={r.id} className="border-t border-slate-200 align-top text-slate-900 hover:bg-slate-50">
                  {showEmployee && <td className="px-3 py-2">{who}</td>}
                  <td className="px-3 py-2">{r.leave_type_name}</td>
                  <td className="px-3 py-2">
                    {dateRange(r)}
                    {PART[r.day_part]}
                  </td>
                  <td className="px-3 py-2">{r.days}</td>
                  <td className="px-3 py-2">{r.approver_name || '-'}</td>
                  <td className="px-3 py-2">{r.decided_by_name ?? '-'}</td>
                  <td className="px-3 py-2">
                    <LeaveStatusBadge status={r.status} />
                  </td>
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      aria-label={`View request from ${who}`}
                      onClick={() => setViewing(r)}
                      className={`${ROW_BTN} border-slate-300 text-slate-800 hover:bg-slate-100 focus-visible:ring-slate-900`}
                    >
                      View
                    </button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {viewing && !target && (
        <LeaveRequestDetailModal
          request={viewing}
          dates={`${dateRange(viewing)}${PART[viewing.day_part]}`}
          canDecide={viewing.status === 'pending' && viewing.user_id !== viewerId}
          onDecide={(decision) => setTarget({ request: viewing, decision })}
          onClose={() => setViewing(null)}
        />
      )}
      {target && (
        <DecisionDialog
          request={target.request}
          decision={target.decision}
          onClose={() => setTarget(null)}
          onDecided={() => setViewing(null)}
        />
      )}
    </>
  )
}
