import { useState } from 'react'
import { FilterBar, Card } from '@employee360/ui'
import { DEFAULT_FILTERS, type LeaveFilters } from '../schemas/filters'
import { LeaveFilterBar } from './LeaveFilterBar'
import { LeaveRequestsList } from './LeaveRequestsList'

const ALL: LeaveFilters = { ...DEFAULT_FILTERS, status: '' }

/** "Leave" tab of the employee detail page: that employee's requests, newest per server order, any status by default. */
export function EmployeeLeaveRequests({ employeeId }: { employeeId: string }) {
  const [filters, setFilters] = useState<LeaveFilters>(ALL)
  const update = (patch: Partial<LeaveFilters>) => setFilters((f) => ({ ...f, ...patch }))
  return (
    <Card padded={false} className="overflow-hidden">
      <FilterBar>
        <LeaveFilterBar value={filters} onChange={update} hideEmployee defaults={ALL} />
      </FilterBar>
      <LeaveRequestsList filters={filters} employeeId={employeeId} defaults={ALL} onPage={(page) => update({ page })} onReset={() => setFilters(ALL)} />
    </Card>
  )
}
