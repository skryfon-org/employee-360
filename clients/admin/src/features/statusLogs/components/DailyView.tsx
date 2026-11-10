import { useState } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, FilterBar, InlineAlert, Skeleton, SkeletonRegion } from '@employee360/ui'
import { SelectField } from '../../employees/components/fields'
import { DailyTable } from './DailyTable'
import { addDays, longDate, todayIso } from '../dates'
import { useDailySummaryQuery, useDepartmentFilterOptions, useProjectFilterOptions } from '../queries/statusLogQueries'

export const INPUT_CLS =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9'

function Stat({ label, value }: { label: string; value: number | string }) {
  return (
    <Card>
      <p className="text-xs font-medium text-slate-600">{label}</p>
      <p className="text-xl font-semibold text-slate-900">{value}</p>
    </Card>
  )
}

/** Admin daily status monitoring: who logged (and who did not) on a date. */
export function DailyView() {
  const [date, setDate] = useState(todayIso())
  const [department, setDepartment] = useState('')
  const [project, setProject] = useState('')
  const [missingOnly, setMissingOnly] = useState(false)
  const departments = useDepartmentFilterOptions()
  const projects = useProjectFilterOptions()
  const { data, isPending, isError, error, refetch } = useDailySummaryQuery(date, department, project)

  const ready = !isPending && !isError
  const rows = data ?? []
  const logged = rows.filter((r) => r.day_status === 'logged').length
  const missing = rows.filter((r) => r.day_status === 'missing').length
  const away = rows.length - logged - missing
  const visible = missingOnly ? rows.filter((r) => r.day_status === 'missing') : rows

  return (
    <div className="flex flex-col gap-6">
      <p className="text-sm text-slate-600">{longDate(date)}</p>

      <div className="grid grid-cols-3 gap-4" aria-label="Summary">
        <Stat label="Logged" value={ready ? logged : '-'} />
        <Stat label="Missing" value={ready ? missing : '-'} />
        <Stat label="Leave / off" value={ready ? away : '-'} />
      </div>

      <Card padded={false} className="overflow-hidden">
        <FilterBar className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div className="flex flex-col gap-1">
            <label htmlFor="sl-date" className="text-xs font-medium text-slate-900">Date</label>
            <div className="flex gap-1">
              <button type="button" aria-label="Previous day" className="h-11 rounded-sm border border-slate-300 bg-white px-2 text-sm hover:bg-slate-100 md:h-9" onClick={() => setDate(addDays(date, -1))}>‹</button>
              <input id="sl-date" type="date" className={INPUT_CLS} value={date} max={todayIso()} onChange={(e) => e.target.value && setDate(e.target.value)} />
              <button type="button" aria-label="Next day" className="h-11 rounded-sm border border-slate-300 bg-white px-2 text-sm hover:bg-slate-100 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 md:h-9" disabled={date >= todayIso()} onClick={() => setDate(addDays(date, 1))}>›</button>
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
            Missing only
          </label>
        </FilterBar>
        {isPending ? (
          <SkeletonRegion label="Loading status logs…" className="flex flex-col gap-4 p-4">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-48 w-full" />
          </SkeletonRegion>
        ) : isError ? (
          <div className="flex flex-col gap-2 p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load status logs.')}</InlineAlert>
            <button type="button" className="h-9 w-fit rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium hover:bg-slate-100" onClick={() => void refetch()}>Retry</button>
          </div>
        ) : visible.length === 0 ? (
          <EmptyState title={missingOnly ? 'No missing logs for this date.' : 'No employees found.'} />
        ) : (
          <DailyTable rows={visible} date={date} />
        )}
      </Card>
    </div>
  )
}
