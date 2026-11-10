import { FilterBar } from '@employee360/ui'
import { useEffect, useState } from 'react'
import { useDebouncedValue } from '../../../hooks/useDebouncedValue'
import { useDepartmentOptionsQuery, usePositionOptionsQuery } from '../queries/employeeQueries'
import { EMPLOYMENT_STATUSES, humanize, type EmployeeListParams } from '../schemas/employeeSchemas'

const SELECT =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9'

function Labeled({ id, label, children }: { id: string; label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-slate-900">
        {label}
      </label>
      {children}
    </div>
  )
}

export function EmployeesFilters({
  params,
  update,
}: {
  params: EmployeeListParams
  update: (patch: Partial<EmployeeListParams>) => void
}) {
  const departments = useDepartmentOptionsQuery().data ?? []
  const positions = usePositionOptionsQuery().data ?? []
  const [search, setSearch] = useState(params.search)
  const debounced = useDebouncedValue(search, 300)

  // Push the debounced search term into the URL params.
  useEffect(() => {
    if (debounced.trim() !== params.search) update({ search: debounced.trim() })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debounced])

  return (
    <FilterBar className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <Labeled id="emp-search" label="Search">
        <input
          id="emp-search"
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Name, email or code"
          className="h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9"
        />
      </Labeled>
      <Labeled id="emp-status" label="Status">
        <select
          id="emp-status"
          className={SELECT}
          value={params.status ?? ''}
          onChange={(e) => update({ status: (e.target.value || undefined) as EmployeeListParams['status'] })}
        >
          <option value="">All</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </Labeled>
      <Labeled id="emp-dept" label="Department">
        <select
          id="emp-dept"
          className={SELECT}
          value={params.department_id ?? ''}
          onChange={(e) => update({ department_id: e.target.value || undefined })}
        >
          <option value="">All</option>
          {departments.map((d) => (
            <option key={d.id} value={d.id}>
              {d.name}
            </option>
          ))}
        </select>
      </Labeled>
      <Labeled id="emp-position" label="Position">
        <select
          id="emp-position"
          className={SELECT}
          value={params.position_id ?? ''}
          onChange={(e) => update({ position_id: e.target.value || undefined })}
        >
          <option value="">All</option>
          {positions.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </Labeled>
      <Labeled id="emp-employment" label="Employment status">
        <select
          id="emp-employment"
          className={SELECT}
          value={params.employment_status ?? ''}
          onChange={(e) => update({ employment_status: (e.target.value || undefined) as EmployeeListParams['employment_status'] })}
        >
          <option value="">All</option>
          {EMPLOYMENT_STATUSES.map((s) => (
            <option key={s} value={s}>
              {humanize(s)}
            </option>
          ))}
        </select>
      </Labeled>
    </FilterBar>
  )
}
