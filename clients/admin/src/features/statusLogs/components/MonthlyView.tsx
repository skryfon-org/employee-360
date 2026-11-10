import { useState } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, FilterBar, InlineAlert, Skeleton, SkeletonRegion } from '@employee360/ui'
import { SelectField } from '../../employees/components/fields'
import { addMonths, formatHours, monthLabel, monthOf } from '../dates'
import { useDepartmentFilterOptions, useMonthlyReportQuery, useProjectFilterOptions } from '../queries/statusLogQueries'
import { INPUT_CLS } from './DailyView'
import { MonthlyTable } from './MonthlyTable'

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <Card>
      <p className="text-xs font-medium text-slate-600">{label}</p>
      <p className="text-xl font-semibold text-slate-900">{value}</p>
    </Card>
  )
}

/** Org-wide monthly report: working days, logged vs not logged, hours per employee. */
export function MonthlyView() {
  const current = monthOf()
  const [month, setMonth] = useState(current)
  const [department, setDepartment] = useState('')
  const [project, setProject] = useState('')
  const [missingOnly, setMissingOnly] = useState(false)
  const departments = useDepartmentFilterOptions()
  const projects = useProjectFilterOptions()
  const { data, isPending, isError, error, refetch } = useMonthlyReportQuery(month, department, project)

  const rows = data?.employees ?? []
  const visible = missingOnly ? rows.filter((r) => (r.missing_days ?? 0) > 0) : rows

  return (
    <div className="flex flex-col gap-6">
      <p className="text-sm text-slate-600">{monthLabel(month)}</p>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4" aria-label="Monthly summary">
        <Stat label="Working days" value={data ? (data.working_days ?? 0) : '-'} />
        <Stat label="Logged employees" value={data ? (data.logged_employees ?? 0) : '-'} />
        <Stat label="Employees with missing logs" value={data ? (data.not_logged_employees ?? 0) : '-'} />
        <Stat label="Total hours" value={data ? formatHours(data.total_hours) : '-'} />
      </div>

      <Card padded={false} className="overflow-hidden">
        <FilterBar className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div className="flex flex-col gap-1">
            <label htmlFor="sl-month" className="text-xs font-medium text-slate-900">Month</label>
            <div className="flex gap-1">
              <button type="button" aria-label="Previous month" className="h-11 rounded-sm border border-slate-300 bg-white px-2 text-sm hover:bg-slate-100 md:h-9" onClick={() => setMonth(addMonths(month, -1))}>‹</button>
              <input id="sl-month" type="month" className={INPUT_CLS} value={month} max={current} onChange={(e) => e.target.value && setMonth(e.target.value)} />
              <button type="button" aria-label="Next month" className="h-11 rounded-sm border border-slate-300 bg-white px-2 text-sm hover:bg-slate-100 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 md:h-9" disabled={month >= current} onClick={() => setMonth(addMonths(month, 1))}>›</button>
            </div>
          </div>
          <SelectField label="Department" name="department" value={department} onChange={(e) => setDepartment(e.target.value)}>
            <option value="">All departments</option>
            {(departments.data ?? []).map((d) => (
              <option key={d.id} value={d.id}>{d.name}</option>
            ))}
          </SelectField>
          <SelectField label="Project" name="project" value={project} onChange={(e) => setProject(e.target.value)}>
            <option value="">All projects</option>
            {(projects.data ?? []).map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </SelectField>
          <label className="flex items-center gap-2 text-sm text-slate-900 lg:self-end lg:pb-2">
            <input type="checkbox" className="h-4 w-4 rounded-sm border-slate-300" checked={missingOnly} onChange={(e) => setMissingOnly(e.target.checked)} />
            Missing logs only
          </label>
        </FilterBar>
        {isPending ? (
          <SkeletonRegion label="Loading monthly report…" className="flex flex-col gap-4 p-4">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-48 w-full" />
          </SkeletonRegion>
        ) : isError ? (
          <div className="flex flex-col gap-2 p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load the monthly report.')}</InlineAlert>
            <button type="button" className="h-9 w-fit rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium hover:bg-slate-100" onClick={() => void refetch()}>Retry</button>
          </div>
        ) : visible.length === 0 ? (
          <EmptyState title={missingOnly ? 'No employees with missing logs this month.' : 'No employees found.'} />
        ) : (
          <MonthlyTable rows={visible} month={month} workingDays={data.working_days ?? 0} />
        )}
      </Card>
    </div>
  )
}
