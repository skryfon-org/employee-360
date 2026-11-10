import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert } from '@employee360/ui'
import { useAssignmentsQuery, useDepartmentOptionsQuery, usePositionOptionsQuery } from '../queries/employeeQueries'
import { Section } from './fields'

export function AssignmentHistory({ employeeId }: { employeeId: string }) {
  const { data, isPending, isError, error } = useAssignmentsQuery(employeeId)
  const departments = useDepartmentOptionsQuery().data ?? []
  const positions = usePositionOptionsQuery().data ?? []
  const dept = (id?: string) => (id ? (departments.find((d) => d.id === id)?.name ?? 'Unknown department') : '-')
  const pos = (id?: string) => (id ? (positions.find((p) => p.id === id)?.name ?? 'Unknown position') : '-')

  return (
    <Section title="Assignment history" description="Department and position changes over time, newest first.">
      {isPending && <p role="status" className="text-sm text-slate-600">Loading assignment history…</p>}
      {isError && <InlineAlert tone="error">{getErrorMessage(error, 'Could not load assignment history.')}</InlineAlert>}
      {data && data.length === 0 && <p className="text-sm text-slate-600">No assignments recorded yet.</p>}
      {data && data.length > 0 && (
        <div className="overflow-x-auto rounded-sm border border-slate-200">
          <table className="w-full min-w-[40rem] text-left">
            <thead className="bg-slate-100 text-xs font-medium text-slate-600">
              <tr>
                <th scope="col" className="px-4 py-3">Department</th>
                <th scope="col" className="px-4 py-3">Position</th>
                <th scope="col" className="px-4 py-3">From</th>
                <th scope="col" className="px-4 py-3">To</th>
                <th scope="col" className="px-4 py-3">Reason</th>
              </tr>
            </thead>
            <tbody>
              {[...data]
                .sort((a, b) => (b.effective_from ?? '').localeCompare(a.effective_from ?? ''))
                .map((a) => (
                  <tr key={a.id} className="border-t border-slate-200 text-sm text-slate-900">
                    <td className="px-4 py-3">{dept(a.department_id)}</td>
                    <td className="px-4 py-3">{pos(a.position_id)}</td>
                    <td className="px-4 py-3">{a.effective_from ?? '-'}</td>
                    <td className="px-4 py-3">{a.is_current || !a.effective_to ? 'Present' : a.effective_to}</td>
                    <td className="px-4 py-3 text-slate-600">{a.change_reason || '-'}</td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      )}
    </Section>
  )
}
