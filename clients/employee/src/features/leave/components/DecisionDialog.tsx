import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert, SubmitButton, useToast } from '@employee360/ui'
import { useState } from 'react'
import { Modal } from '../../../components/ui/Modal'
import { useDecideLeaveMutation, type LeaveRequest } from '../queries/leaveQueries'
import { dateRange } from './LeaveRequestsTable'

export function DecisionDialog({ request, decision, onClose }: { request: LeaveRequest; decision: 'approve' | 'reject'; onClose: () => void }) {
  const [comment, setComment] = useState('')
  const [error, setError] = useState<string | null>(null)
  const decide = useDecideLeaveMutation()
  const toast = useToast()
  const verb = decision === 'approve' ? 'Approve' : 'Reject'

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    decide.mutate(
      { id: request.id, decision, comment: comment.trim() },
      {
        onSuccess: () => {
          toast.success(decision === 'approve' ? 'Leave request approved' : 'Leave request rejected')
          onClose()
        },
        onError: (err) => setError(getErrorMessage(err, 'Could not save your decision.')),
      },
    )
  }

  return (
    <Modal title={`${verb} leave request`} onClose={onClose}>
      <form onSubmit={submit} className="flex flex-col gap-4">
        <p className="text-sm text-slate-900">
          {request.user_name || request.user_email} - {request.leave_type_name}, {dateRange(request)}
        </p>
        <div className="flex flex-col gap-1">
          <label htmlFor="decision-comment" className="text-xs font-medium text-slate-900">
            Comment (optional)
          </label>
          <textarea
            id="decision-comment"
            rows={3}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            className="w-full rounded-sm border border-slate-300 bg-white px-3 py-2 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm"
          />
        </div>
        {error && <InlineAlert tone="error">{error}</InlineAlert>}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            Cancel
          </button>
          <SubmitButton loading={decide.isPending}>{verb}</SubmitButton>
        </div>
      </form>
    </Modal>
  )
}
