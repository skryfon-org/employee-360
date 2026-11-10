import { useRef } from 'react'
import { useDialogFocus } from '../../../hooks/useDialogFocus'
import type { LeaveRequest } from '../queries/leaveQueries'
import { LeaveStatusBadge } from './LeaveStatusBadge'
import { PRIMARY_BTN, SECONDARY_BTN } from './styles'

/** Read-only request details (full reason) with the Approve / Reject actions for decidable requests. */
export function LeaveRequestDetailModal({
  request,
  dates,
  canDecide,
  onDecide,
  onClose,
}: {
  request: LeaveRequest
  dates: string
  canDecide: boolean
  onDecide: (decision: 'approve' | 'reject') => void
  onClose: () => void
}) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  useDialogFocus(dialogRef, closeRef, onClose, false)
  const who = request.user_name || request.user_email

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="leave-detail-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-md overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <div className="flex items-start justify-between gap-3">
          <h2 id="leave-detail-title" className="text-base font-semibold text-slate-900">
            Leave request
          </h2>
          <LeaveStatusBadge status={request.status} />
        </div>
        <p className="mt-2 text-sm text-slate-900">
          {who} - {request.leave_type_name}, {dates}
        </p>
        <div className="mt-4">
          <h3 className="text-xs font-medium text-slate-900">Reason</h3>
          <p className="mt-1 whitespace-pre-wrap break-words text-sm text-slate-600">{request.reason || 'No reason provided.'}</p>
        </div>
        {request.decision_comment && (
          <div className="mt-4">
            <h3 className="text-xs font-medium text-slate-900">Decision comment</h3>
            <p className="mt-1 whitespace-pre-wrap break-words text-sm text-slate-600">{request.decision_comment}</p>
          </div>
        )}
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <button type="button" ref={closeRef} onClick={onClose} className={SECONDARY_BTN}>
            Close
          </button>
          {canDecide && (
            <>
              <button
                type="button"
                aria-label={`Reject request from ${who}`}
                onClick={() => onDecide('reject')}
                className="inline-flex h-11 items-center rounded-sm border border-red-300 bg-white px-4 text-sm font-medium text-red-600 hover:bg-red-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 md:h-9"
              >
                Reject
              </button>
              <button type="button" aria-label={`Approve request from ${who}`} onClick={() => onDecide('approve')} className={PRIMARY_BTN}>
                Approve
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
