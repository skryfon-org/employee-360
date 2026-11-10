import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, FilterBar, InlineAlert, RowActionsMenu } from '@employee360/ui'
import {
  useProjectAllocationsQuery,
  useUpdateAllocationMutation,
  type ProjectAllocation,
} from '../queries/projectQueries'
import { AllocationFormModal } from './AllocationFormModal'
import { secondaryButtonCls } from '../../employees/components/fields'
import { useToast } from '../../../hooks/useToast'

import {
  ALLOCATION_FILTERS,
  ALLOCATION_FILTER_LABELS,
  allocationState,
  parseAllocationFilter,
  todayISO as today,
} from '../allocationState'
import { AllocationStateBadge } from './AllocationStateBadge'
import { AllocationTimeline } from './AllocationTimeline'

type View = 'table' | 'timeline'
const segCls = (on: boolean) =>
  `h-9 px-3 text-sm font-medium border border-slate-300 first:rounded-l-sm last:rounded-r-sm -ml-px first:ml-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 ${
    on ? 'bg-slate-900 text-white border-slate-900' : 'bg-white text-slate-800 hover:bg-slate-100'
  }`

interface Props {
  projectId: string
  archived: boolean
  projectStartDate?: string
  projectEndDate?: string | null
}

export function ProjectAllocations({ projectId, archived, projectStartDate, projectEndDate }: Props) {
  const [params, setParams] = useSearchParams()
  const filter = parseAllocationFilter(params.get('state'))
  const view: View = params.get('view') === 'timeline' ? 'timeline' : 'table'
  const setParam = (key: string, value: string | null) =>
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        if (value) next.set(key, value)
        else next.delete(key)
        return next
      },
      { replace: true },
    )
  const toast = useToast()
  const { data, isPending, isError, error } = useProjectAllocationsQuery(projectId, filter)
  const update = useUpdateAllocationMutation()
  const [adding, setAdding] = useState(false)
  const [editing, setEditing] = useState<ProjectAllocation | null>(null)

  const name = (a: ProjectAllocation) => a.user_name || 'Unknown employee'

  const todayStr = today()
  const rows = data ?? []
  const emptyCopy: Record<string, string> = {
    active: 'No active allocations.',
    upcoming: 'No upcoming allocations.',
    past: 'No past allocations.',
  }

  const endNow = (a: ProjectAllocation) =>
    update.mutate(
      { id: a.id as string, body: { end_date: today() } },
      {
        onSuccess: () => toast.success('Allocation ended.'),
        onError: (err) => toast.error(getErrorMessage(err, 'Could not end allocation.')),
      },
    )

  // A project with no allocations at all has nothing to filter or switch views on.
  const showControls = Boolean(data) && !(rows.length === 0 && filter === 'all')
  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-base font-semibold text-slate-900">Allocations</h2>
          <p className="mt-1 text-xs text-slate-600">
            {archived ? 'Archived projects cannot receive new allocations.' : 'Employees assigned to this project.'}
          </p>
        </div>
        <button type="button" className={secondaryButtonCls} onClick={() => setAdding(true)} disabled={archived}>
          Add allocation
        </button>
      </div>
      <Card padded={false} className="overflow-hidden">
        {showControls && (
          <FilterBar className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between">
            <div className="flex flex-col gap-1 sm:w-48">
              <label htmlFor="allocation-state" className="text-xs font-medium text-slate-900">
                Show
              </label>
              <select
                id="allocation-state"
                value={filter}
                onChange={(e) => setParam('state', e.target.value === 'all' ? null : e.target.value)}
                className="h-11 rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9"
              >
                {ALLOCATION_FILTERS.map((f) => (
                  <option key={f} value={f}>
                    {ALLOCATION_FILTER_LABELS[f]}
                  </option>
                ))}
              </select>
            </div>
            <div role="group" aria-label="View" className="flex">
              {(['table', 'timeline'] as const).map((v) => (
                <button
                  key={v}
                  type="button"
                  aria-pressed={view === v}
                  className={segCls(view === v)}
                  onClick={() => setParam('view', v === 'table' ? null : v)}
                >
                  {v === 'table' ? 'Table' : 'Timeline'}
                </button>
              ))}
            </div>
          </FilterBar>
        )}
        {isPending && <p role="status" className="p-4 text-sm text-slate-600">Loading allocations…</p>}
        {isError && (
          <div className="p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load allocations.')}</InlineAlert>
          </div>
        )}
        {data && rows.length === 0 && filter === 'all' && (
          <EmptyState title="No allocations yet" description="Add an employee to start tracking who works on this project." />
        )}
        {data && rows.length === 0 && filter !== 'all' && (
          <EmptyState
            role="status"
            title={emptyCopy[filter]}
            action={
              <button type="button" className={`${secondaryButtonCls} mt-2`} onClick={() => setParam('state', null)}>
                Show all allocations
              </button>
            }
          />
        )}
        {rows.length > 0 && view === 'timeline' && (
          <div className="p-4">
            <AllocationTimeline
              rows={rows.map((a) => ({
                id: a.id as string,
                name: name(a),
                percent: a.allocation_percent,
                start: a.start_date as string,
                end: a.end_date,
              }))}
              projectStart={projectStartDate}
              projectEnd={projectEndDate}
            />
          </div>
        )}
        {rows.length > 0 && view === 'table' && (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[40rem] text-left">
              <thead className="bg-slate-100 text-xs font-medium text-slate-600">
                <tr>
                  <th scope="col" className="px-4 py-3">Employee</th>
                  <th scope="col" className="px-4 py-3">Allocation</th>
                  <th scope="col" className="px-4 py-3">Role</th>
                  <th scope="col" className="px-4 py-3">Start</th>
                  <th scope="col" className="px-4 py-3">End</th>
                  <th scope="col" className="px-4 py-3">State</th>
                  <th scope="col" className="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((a) => {
                  const ended = Boolean(a.end_date && a.end_date <= today())
                  const label = name(a)
                  return (
                    <tr key={a.id} className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50">
                      <td className="px-4 py-3 font-medium">
                        <Link to={`/employees/${a.user_id}/allocations`} className="text-slate-900 hover:text-slate-700">
                          {label}
                        </Link>
                      </td>
                      <td className="px-4 py-3">{a.allocation_percent}%</td>
                      <td className="px-4 py-3 text-slate-600">{a.role_on_project || '-'}</td>
                      <td className="px-4 py-3">{a.start_date}</td>
                      <td className="px-4 py-3">{a.end_date ?? 'Ongoing'}</td>
                      <td className="px-4 py-3"><AllocationStateBadge state={allocationState(a, todayStr)} /></td>
                      <td className="px-4 py-3">
                        <div className="flex justify-end">
                          <RowActionsMenu
                            label={`allocation ${label}`}
                            actions={[
                              { label: 'Edit', onSelect: () => setEditing(a) },
                              { label: 'End allocation', disabled: ended || update.isPending, onSelect: () => endNow(a) },
                            ]}
                          />
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>
      {adding && <AllocationFormModal projectId={projectId} projectStartDate={projectStartDate} projectEndDate={projectEndDate} onClose={() => setAdding(false)} />}
      {editing && (
        <AllocationFormModal
          projectId={projectId}
          projectStartDate={projectStartDate}
          projectEndDate={projectEndDate}
          allocation={editing}
          employeeName={name(editing)}
          onClose={() => setEditing(null)}
        />
      )}
    </section>
  )
}
