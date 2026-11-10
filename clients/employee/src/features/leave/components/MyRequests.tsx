import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, InlineAlert, useToast } from '@employee360/ui'
import { useState } from 'react'
import { Modal } from '../../../components/ui/Modal'
import { useCancelLeaveMutation, useMyLeaveRequestsQuery, type LeaveRequest } from '../queries/leaveQueries'
import { EMPTY_FILTERS, filterKey, hasFilters, type LeaveFilters } from '../schemas/filters'
import { LeaveFilterBar } from './LeaveFilterBar'
import { LeaveRequestsTable, dateRange } from './LeaveRequestsTable'
import { Pager } from './Pager'
import { ROW_BTN } from './styles'
import { canCancel } from './utils'

/** Filter bar plus list; the list is keyed by the filters so any change starts again at page 1. */
export function MyRequests({ filters, onFiltersChange }: { filters: LeaveFilters; onFiltersChange: (f: LeaveFilters) => void }) {
  return (
    <div className="flex flex-col gap-4">
      <LeaveFilterBar value={filters} onChange={onFiltersChange} />
      <MyList key={filterKey(filters)} filters={filters} onClear={() => onFiltersChange(EMPTY_FILTERS)} />
    </div>
  )
}

function MyList({ filters, onClear }: { filters: LeaveFilters; onClear: () => void }) {
  const [page, setPage] = useState(1)
  const { data, isPending, isError, error } = useMyLeaveRequestsQuery(page, filters)
  const cancel = useCancelLeaveMutation()
  const toast = useToast()
  const [actionError, setActionError] = useState<string | null>(null)
  const [confirming, setConfirming] = useState<LeaveRequest | null>(null)

  if (isPending)
    return (
      <p role="status" className="p-4 text-sm text-slate-600">
        Loading your requests…
      </p>
    )
  if (isError) return <InlineAlert tone="error">{getErrorMessage(error, 'Could not load your leave requests.')}</InlineAlert>
  if (data.data.length === 0 && hasFilters(filters))
    return (
      <Card padded={false}>
        <EmptyState
          title="No requests match your filters"
          description="Try a different status or date range."
          action={
            <button type="button" onClick={onClear} className="mt-2 h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2">
              Clear filters
            </button>
          }
        />
      </Card>
    )
  if (data.data.length === 0)
    return (
      <Card padded={false}>
        <EmptyState title="No leave requests yet" description="Requests you submit will appear here." />
      </Card>
    )

  const onCancel = (r: LeaveRequest) => {
    setActionError(null)
    cancel.mutate(r.id, {
      onSuccess: () => {
        setConfirming(null)
        toast.success('Leave request cancelled')
      },
      onError: (e) => {
        setConfirming(null)
        setActionError(getErrorMessage(e, 'Could not cancel the request.'))
      },
    })
  }

  return (
    <div className="flex flex-col gap-3">
      {actionError && <InlineAlert tone="error">{actionError}</InlineAlert>}
      <LeaveRequestsTable
        rows={data.data}
        variant="mine"
        actions={(r) =>
          canCancel(r) ? (
            <button
              type="button"
              aria-label={`Cancel ${r.leave_type_name} request starting ${r.start_date}`}
              disabled={cancel.isPending}
              onClick={() => {
                setActionError(null)
                setConfirming(r)
              }}
              className={`${ROW_BTN} border-red-300 text-red-600 hover:bg-red-50 focus-visible:ring-red-600`}
            >
              Cancel
            </button>
          ) : null
        }
      />
      <Pager page={page} totalPages={data.meta?.total_pages ?? 1} onChange={setPage} />
      {confirming && (
        <Modal title="Cancel leave request" onClose={() => !cancel.isPending && setConfirming(null)}>
          <dl className="mb-4 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
            <dt className="text-slate-600">Leave type</dt>
            <dd className="text-slate-900">{confirming.leave_type_name}</dd>
            <dt className="text-slate-600">Dates</dt>
            <dd className="text-slate-900">{dateRange(confirming)}</dd>
          </dl>
          <p className="mb-4 text-sm text-slate-900">Cancel this leave request?</p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={cancel.isPending}
              onClick={() => setConfirming(null)}
              className="h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500"
            >
              Keep request
            </button>
            <button
              type="button"
              disabled={cancel.isPending}
              aria-busy={cancel.isPending}
              onClick={() => onCancel(confirming)}
              className="inline-flex h-9 items-center justify-center rounded-sm bg-red-600 px-4 text-sm font-semibold text-white hover:bg-red-700 active:bg-red-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-200 disabled:text-slate-500"
            >
              {cancel.isPending ? 'Cancelling…' : 'Cancel request'}
            </button>
          </div>
        </Modal>
      )}
    </div>
  )
}
