import type { LeaveStatus } from '@employee360/api-client'
import { DateRangePicker } from '@employee360/ui'
import { useState } from 'react'
import { EmployeeCombobox } from '../../projects/components/EmployeeCombobox'
import type { EmployeeSummary } from '../../projects/queries/projectQueries'
import { useLeaveEmployeeQuery, useLeaveTypesQuery } from '../queries/leaveQueries'
import { DEFAULT_FILTERS, isDefaultFilters, type LeaveFilters } from '../schemas/filters'
import { FIELD, LABEL, SECONDARY_BTN } from './styles'

const LABELS: Record<LeaveStatus, string> = { pending: 'Pending', approved: 'Approved', rejected: 'Rejected', cancelled: 'Cancelled' }

interface Props {
  value: LeaveFilters
  onChange: (next: Partial<LeaveFilters>) => void
  /** Hide the employee filter (used on the employee detail tab). */
  hideEmployee?: boolean
  /** View shown on first load and restored by Reset (default: pending queue). */
  defaults?: LeaveFilters
}

export function LeaveFilterBar({ value, onChange, hideEmployee, defaults = DEFAULT_FILTERS }: Props) {
  const types = useLeaveTypesQuery()
  const typeOptions = types.data ?? []
  // A valid but unknown id (e.g. from an old link) stays visible so it is clear why the list is empty.
  const unknownType = !!value.leaveTypeId && types.isSuccess && !typeOptions.some((t) => t.id === value.leaveTypeId)

  return (
    <form role="search" aria-label="Filter leave requests" onSubmit={(e) => e.preventDefault()} className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end">
      {!hideEmployee && (
        <div className="lg:w-64">
          <EmployeeFilter employeeId={value.employeeId} onChange={(employeeId) => onChange({ employeeId, page: 1 })} />
        </div>
      )}
      <div className="flex flex-col gap-1 lg:w-40">
        <label htmlFor="leave-filter-status" className={LABEL}>
          Status
        </label>
        <select id="leave-filter-status" className={FIELD} value={value.status} onChange={(e) => onChange({ status: e.target.value as LeaveFilters['status'], page: 1 })}>
          <option value="">All</option>
          {(Object.keys(LABELS) as LeaveStatus[]).map((s) => (
            <option key={s} value={s}>
              {LABELS[s]}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-col gap-1 lg:w-44">
        <label htmlFor="leave-filter-type" className={LABEL}>
          Leave type
        </label>
        <select id="leave-filter-type" className={FIELD} value={value.leaveTypeId} onChange={(e) => onChange({ leaveTypeId: e.target.value, page: 1 })}>
          <option value="">All</option>
          {typeOptions.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
              {t.is_active ? '' : ' (inactive)'}
            </option>
          ))}
          {unknownType && <option value={value.leaveTypeId}>Unknown leave type</option>}
        </select>
      </div>
      <DateRangePicker id="leave-filter-range" label="Dates" value={{ from: value.from, to: value.to }} onChange={(r) => onChange({ ...r, page: 1 })} />
      <button type="button" disabled={isDefaultFilters(value, defaults)} onClick={() => onChange({ ...defaults })} className={SECONDARY_BTN}>
        Reset
      </button>
    </form>
  )
}

/** Searchable employee picker (server-side, debounced). Keeps only the id in the filter state. */
function EmployeeFilter({ employeeId, onChange }: { employeeId: string; onChange: (id: string) => void }) {
  const [picked, setPicked] = useState<EmployeeSummary | null>(null)
  // A valid id from an old link/URL is resolved to a name; an unknown one stays visible as such.
  const resolved = useLeaveEmployeeQuery(employeeId && picked?.id !== employeeId ? employeeId : '')
  let selected: EmployeeSummary | null = null
  if (employeeId) {
    if (picked?.id === employeeId) selected = picked
    else if (resolved.data) selected = resolved.data
    else if (resolved.isError) selected = { id: employeeId, first_name: 'Unknown employee' }
    else return <p className="text-xs text-slate-600">Loading employee…</p>
  }
  return (
    <EmployeeCombobox
      key={employeeId ? 'set' : 'none'}
      label="Employee"
      selected={selected}
      activeOnly={false}
      showAllocation={false}
      onSelect={(e) => {
        setPicked(e)
        onChange(e?.id ?? '')
      }}
    />
  )
}
