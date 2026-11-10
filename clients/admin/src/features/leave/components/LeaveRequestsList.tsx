import { getErrorMessage } from '@employee360/api-client'
import { EmptyState, InlineAlert } from '@employee360/ui'
import { useAllLeaveRequestsQuery } from '../queries/leaveQueries'
import { DEFAULT_FILTERS, isDefaultFilters, type LeaveFilters } from '../schemas/filters'
import { LeaveRequestsTable } from './LeaveRequestsTable'
import { Pager } from './Pager'
import { SECONDARY_BTN } from './styles'

/** Fetches and renders one page of requests for the given filters (plus an optional fixed employee). */
export function LeaveRequestsList({
  filters,
  employeeId,
  defaults = DEFAULT_FILTERS,
  onPage,
  onReset,
}: {
  filters: LeaveFilters
  /** Pins the list to one employee (employee detail tab); overrides the filter. */
  employeeId?: string
  defaults?: LeaveFilters
  onPage: (page: number) => void
  onReset: () => void
}) {
  const { data, isPending, isError, error, refetch } = useAllLeaveRequestsQuery({
    status: filters.status || undefined,
    employeeId: employeeId ?? (filters.employeeId || undefined),
    leaveTypeId: filters.leaveTypeId || undefined,
    from: filters.from || undefined,
    to: filters.to || undefined,
    page: filters.page,
  })
  const rows = data?.data ?? []

  if (isPending) return <p role="status" className="p-4 text-sm text-slate-600">Loading…</p>
  if (isError) {
    return (
      <div className="flex flex-col items-start gap-3 p-4">
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load leave requests.')}</InlineAlert>
        <button type="button" className={SECONDARY_BTN} onClick={() => void refetch()}>
          Retry
        </button>
      </div>
    )
  }
  if (rows.length === 0) {
    const defaultView = isDefaultFilters(filters, defaults)
    const pendingQueue = defaults.status === 'pending'
    const title = !defaultView ? 'No matching requests' : pendingQueue ? 'No pending requests' : 'No leave requests yet'
    const description = !defaultView
      ? 'Try different filters.'
      : pendingQueue
        ? 'New leave requests awaiting a decision will appear here.'
        : 'Leave requests will appear here once employees apply.'
    return (
      <EmptyState
        title={title}
        description={description}
        action={
          !defaultView && (
            <button type="button" onClick={onReset} className={`${SECONDARY_BTN} mt-2`}>
              Reset filters
            </button>
          )
        }
      />
    )
  }
  return (
    <>
      <LeaveRequestsTable rows={rows} showEmployee={!employeeId} />
      <Pager page={filters.page} totalPages={data?.meta?.total_pages ?? 1} onChange={onPage} />
    </>
  )
}
