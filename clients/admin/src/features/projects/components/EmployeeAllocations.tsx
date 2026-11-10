import { useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, FilterBar, InlineAlert, RowActionsMenu } from '@employee360/ui'
import {
  useEmployeeAllocationsQuery,
  useProjectQuery,
  useUpdateAllocationMutation,
  type EmployeeAllocationScope,
  type ProjectAllocation,
  type UserAllocation,
} from '../queries/projectQueries'
import { AllocationFormModal } from './AllocationFormModal'
import { OverAllocationAlert, TotalPercentBadge } from './OverAllocationAlert'
import { ProjectStatusBadge } from './ProjectStatusBadge'
import { Section, secondaryButtonCls } from '../../employees/components/fields'
import { AllocationStateBadge } from './AllocationStateBadge'
import { ALLOCATION_FILTER_LABELS, allocationState, todayISO } from '../allocationState'
import { useToast } from '../../../hooks/useToast'
import { useDialogFocus } from '../../../hooks/useDialogFocus'

const SCOPES = ['active', 'upcoming', 'past', 'all'] as const satisfies readonly EmployeeAllocationScope[]
const EMPTY_COPY: Record<EmployeeAllocationScope, string> = {
  active: 'No active project allocations.',
  upcoming: 'No upcoming project allocations.',
  past: 'No past project allocations.',
  all: 'No project allocations yet.',
}
/** Shown in place of the edit form until the project's date window is known (or if loading it failed). */
function ProjectWindowGate({ error, onRetry, onClose }: { error: unknown; onRetry: () => void; onClose: () => void }) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  useDialogFocus(dialogRef, closeRef, onClose, false)
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4">
      <div ref={dialogRef} role="dialog" aria-modal="true" aria-label="Edit allocation" className="w-full max-w-lg rounded-sm border border-slate-300 bg-white p-4 sm:p-6">
        {error ? (
          <InlineAlert tone="error">{getErrorMessage(error, 'Could not load the project.')}</InlineAlert>
        ) : (
          <p role="status" className="text-sm text-slate-600">Loading project…</p>
        )}
        <div className="mt-4 flex justify-end gap-2">
          {Boolean(error) && <button type="button" className={secondaryButtonCls} onClick={onRetry}>Retry</button>}
          <button ref={closeRef} type="button" className={secondaryButtonCls} onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  )
}

/**
 * Employee detail "Allocations" section with an Active/Upcoming/Past/All scope filter.
 * The server's `total_percent` always means "live allocations active today", so it is only
 * shown for Active and All, and never recomputed from rows.
 */
interface Props {
  employeeId: string
  employeeName?: string
  /** Inactive employees cannot receive new allocations (backend rejects them). */
  active?: boolean
}

export function EmployeeAllocations({ employeeId, employeeName, active = true }: Props) {
  const [scope, setScope] = useState<EmployeeAllocationScope>('active')
  const { data, isPending, isError, error } = useEmployeeAllocationsQuery(employeeId, scope)
  const showTotal = scope === 'active' || scope === 'all'
  const todayStr = todayISO()
  const toast = useToast()
  const update = useUpdateAllocationMutation()
  const [adding, setAdding] = useState(false)
  const [editing, setEditing] = useState<UserAllocation | null>(null)
  // Allocation rows carry no project dates, so fetch the project to validate edits against its window.
  const editProject = useProjectQuery(editing?.project_id)
  // Future-dated or already-ended allocations are invisible under Active, so jump to the matching scope.
  const switchScope = (start: string, end: string | null) => {
    const state = allocationState({ start_date: start, end_date: end }, todayISO())
    if (state === 'upcoming') setScope('upcoming')
    else if (state === 'past') setScope('past')
  }
  const endNow = (a: UserAllocation) =>
    update.mutate(
      { id: a.id as string, body: { end_date: todayISO() } },
      {
        onSuccess: () => toast.success('Allocation ended.'),
        onError: (err) => toast.error(getErrorMessage(err, 'Could not end allocation.')),
      },
    )
  const asAllocation = (a: UserAllocation): ProjectAllocation => ({
    id: a.id,
    project_id: a.project_id,
    user_id: a.user_id,
    allocation_percent: a.allocation_percent,
    role_on_project: a.role_on_project,
    start_date: a.start_date,
    end_date: a.end_date,
  })
  const total = data?.total_percent ?? 0
  const rows = data?.allocations ?? []
  return (
    <Section title="Allocations" description="Project allocations for this employee.">
      <Card padded={false} className="overflow-hidden">
        <FilterBar className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between">
          <div className="flex flex-col gap-1 sm:w-48">
            <label htmlFor="allocation-scope" className="text-xs font-medium text-slate-900">
              Show
            </label>
            <select
              id="allocation-scope"
              value={scope}
              onChange={(e) => setScope(e.target.value as EmployeeAllocationScope)}
              className="h-11 rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9"
            >
              {SCOPES.map((f) => (
                <option key={f} value={f}>
                  {ALLOCATION_FILTER_LABELS[f]}
                </option>
              ))}
            </select>
          </div>
          {active && (
            <button type="button" className={secondaryButtonCls} onClick={() => setAdding(true)}>
              Allocate to project
            </button>
          )}
        </FilterBar>
        {isPending && <p role="status" className="p-4 text-sm text-slate-600">Loading allocations…</p>}
        {isError && (
          <div className="p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load allocations.')}</InlineAlert>
          </div>
        )}
        {data && (
          <>
            {showTotal && (
              <div className="flex flex-col gap-3 border-b border-slate-200 p-4">
                <div className="flex items-center gap-2">
                  <TotalPercentBadge total={total} />
                  {scope === 'all' && <span className="text-xs text-slate-600">Current allocations only</span>}
                </div>
                {total > 100 && <OverAllocationAlert total={total} />}
              </div>
            )}
            {rows.length === 0 ? (
              <EmptyState role="status" title="No allocations" description={EMPTY_COPY[scope]} />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[40rem] text-left">
                  <thead className="bg-slate-100 text-xs font-medium text-slate-600">
                    <tr>
                      <th scope="col" className="px-4 py-3">Project</th>
                      <th scope="col" className="px-4 py-3">Status</th>
                      <th scope="col" className="px-4 py-3">Allocation</th>
                      <th scope="col" className="px-4 py-3">Role</th>
                      <th scope="col" className="px-4 py-3">Start</th>
                      <th scope="col" className="px-4 py-3">End</th>
                      <th scope="col" className="px-4 py-3">State</th>
                      <th scope="col" className="px-4 py-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((a) => (
                      <tr key={a.id} className="border-t border-slate-200 text-sm text-slate-900">
                        <td className="px-4 py-3 font-medium">
                          <Link to={`/projects/${a.project_id}`} className="text-slate-900 hover:text-slate-700">
                            {a.project_name}
                          </Link>
                          <span className="ml-2 text-xs text-slate-600">{a.project_code}</span>
                        </td>
                        <td className="px-4 py-3">
                          <ProjectStatusBadge status={a.project_status} />
                        </td>
                        <td className="px-4 py-3">
                          {a.allocation_percent}%
                          {scope === 'active' && a.counts_toward_total === false && <span className="ml-1 text-xs text-slate-600">(not counted)</span>}
                        </td>
                        <td className="px-4 py-3 text-slate-600">{a.role_on_project || '-'}</td>
                        <td className="px-4 py-3">{a.start_date}</td>
                        <td className="px-4 py-3">{a.end_date ?? 'Ongoing'}</td>
                        <td className="px-4 py-3"><AllocationStateBadge state={allocationState(a, todayStr)} /></td>
                        <td className="px-4 py-3">
                          <div className="flex justify-end">
                            <RowActionsMenu
                              label={`allocation ${a.project_name}`}
                              actions={[
                                { label: 'Edit', onSelect: () => setEditing(a) },
                                {
                                  label: 'End allocation',
                                  disabled: Boolean(a.end_date && a.end_date <= todayStr) || a.start_date === undefined || a.start_date > todayStr || update.isPending,
                                  title: a.start_date && a.start_date > todayStr ? 'Upcoming allocations cannot be ended; edit the end date instead' : undefined,
                                  onSelect: () => endNow(a),
                                },
                              ]}
                            />
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )}
      </Card>
      {adding && <AllocationFormModal userId={employeeId} onSaved={switchScope} onClose={() => setAdding(false)} />}
      {editing && !editProject.data && (
        <ProjectWindowGate error={editProject.isError ? editProject.error : null} onRetry={() => void editProject.refetch()} onClose={() => setEditing(null)} />
      )}
      {editing && editProject.data && (
        <AllocationFormModal
          projectId={editing.project_id as string}
          allocation={asAllocation(editing)}
          employeeName={employeeName}
          projectStartDate={editProject.data.start_date}
          projectEndDate={editProject.data.end_date}
          onSaved={switchScope}
          onClose={() => setEditing(null)}
        />
      )}
    </Section>
  )
}
