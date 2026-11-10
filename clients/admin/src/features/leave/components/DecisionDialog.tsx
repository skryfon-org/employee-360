import { useRef, useState } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert, SubmitButton } from '@employee360/ui'
import { useDialogFocus } from '../../../hooks/useDialogFocus'
import { useToast } from '../../../hooks/useToast'
import { useDecideLeaveMutation, type LeaveRequest } from '../queries/leaveQueries'
import { dateRange } from './LeaveRequestsTable'
import { FIELD, LABEL, SECONDARY_BTN } from './styles'

export const COMMENT_MAX = 500

export function DecisionDialog({
  request,
  decision,
  onClose,
  onDecided,
}: {
  request: LeaveRequest
  decision: 'approve' | 'reject'
  onClose: () => void
  /** Called after a successful decision (before onClose). */
  onDecided?: () => void
}) {
  const [comment, setComment] = useState('')
  const [error, setError] = useState<string | null>(null)
  const decide = useDecideLeaveMutation()
  const toast = useToast()
  const verb = decision === 'approve' ? 'Approve' : 'Reject'
  const dialogRef = useRef<HTMLDivElement>(null)
  const commentRef = useRef<HTMLTextAreaElement>(null)
  useDialogFocus(dialogRef, commentRef, onClose, decide.isPending)

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    decide.mutate(
      { id: request.id, decision, comment: comment.trim() },
      {
        onSuccess: () => {
          toast.success(decision === 'approve' ? 'Leave request approved.' : 'Leave request rejected.')
          onDecided?.()
          onClose()
        },
        onError: (err) => setError(getErrorMessage(err, 'Could not save your decision.')),
      },
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="decision-title"
        className="max-h-[calc(100dvh-2rem)] w-full max-w-md overflow-y-auto rounded-sm border border-slate-300 bg-white p-4 sm:p-6"
      >
        <h2 id="decision-title" className="text-base font-semibold text-slate-900">
          {verb} leave request
        </h2>
        <form onSubmit={submit} className="mt-4 flex flex-col gap-4">
          <p className="text-sm text-slate-900">
            {request.user_name || request.user_email} - {request.leave_type_name}, {dateRange(request)}
          </p>
          <div className="flex flex-col gap-1">
            <label htmlFor="decision-comment" className={LABEL}>
              Comment (optional)
            </label>
            <textarea
              id="decision-comment"
              ref={commentRef}
              rows={3}
              maxLength={COMMENT_MAX}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              className={`${FIELD} h-auto py-2 md:h-auto`}
            />
          </div>
          {error && <InlineAlert tone="error">{error}</InlineAlert>}
          <div className="flex justify-end gap-2">
            <button type="button" onClick={onClose} disabled={decide.isPending} className={SECONDARY_BTN}>
              Cancel
            </button>
            <SubmitButton loading={decide.isPending}>{verb}</SubmitButton>
          </div>
        </form>
      </div>
    </div>
  )
}
