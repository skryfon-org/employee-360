import type { LeaveStatus } from '@employee360/api-client'
import { DateRangePicker } from '@employee360/ui'
import { useLeaveTypesQuery } from '../queries/leaveQueries'
import { EMPTY_FILTERS, hasFilters, type LeaveFilters } from '../schemas/filters'

const FIELD =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 aria-[invalid=true]:border-red-600 sm:text-sm md:h-9'
const LABEL = 'text-xs font-medium text-slate-900'
const LABELS: Record<LeaveStatus, string> = { pending: 'Pending', approved: 'Approved', rejected: 'Rejected', cancelled: 'Cancelled' }

interface Props {
  value: LeaveFilters
  onChange: (next: LeaveFilters) => void
  statuses?: LeaveStatus[]
}

export function LeaveFilterBar({ value, onChange, statuses = ['pending', 'approved', 'rejected', 'cancelled'] }: Props) {
  const types = useLeaveTypesQuery()
  const typeOptions = types.data ?? []
  // A valid but unknown id (e.g. from an old link) stays visible so the user can see why the list is empty.
  const unknownType = !!value.leaveTypeId && types.isSuccess && !typeOptions.some((t) => t.id === value.leaveTypeId)
  return (
    <form role="search" aria-label="Filter leave requests" onSubmit={(e) => e.preventDefault()} className="flex flex-col gap-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-1 sm:w-44">
          <label htmlFor="leave-filter-status" className={LABEL}>
            Status
          </label>
          <select id="leave-filter-status" className={FIELD} value={value.status} onChange={(e) => onChange({ ...value, status: e.target.value as LeaveFilters['status'] })}>
            <option value="">All</option>
            {statuses.map((s) => (
              <option key={s} value={s}>
                {LABELS[s]}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1 sm:w-44">
          <label htmlFor="leave-filter-type" className={LABEL}>
            Leave type
          </label>
          <select id="leave-filter-type" className={FIELD} value={value.leaveTypeId} onChange={(e) => onChange({ ...value, leaveTypeId: e.target.value })}>
            <option value="">All</option>
            {typeOptions.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
            {unknownType && <option value={value.leaveTypeId}>Unknown leave type</option>}
          </select>
        </div>
        <DateRangePicker id="leave-filter-range" label="Dates" value={{ from: value.from, to: value.to }} onChange={(r) => onChange({ ...value, ...r })} />
        <button
          type="button"
          disabled={!hasFilters(value)}
          onClick={() => onChange(EMPTY_FILTERS)}
          className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 md:h-9"
        >
          Clear
        </button>
      </div>
    </form>
  )
}
