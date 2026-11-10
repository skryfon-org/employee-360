import { EmptyState, RowActionsMenu } from '@employee360/ui'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { EmployeeSummary } from '../queries/employeeQueries'
import { EmployeeStatusBadge, EmploymentStatusBadge } from './EmployeeStatusBadge'
import { ToggleActiveDialog } from './ToggleActiveDialog'
import { fullName } from '../employeeName'
import { humanize } from '../schemas/employeeSchemas'

const CELL =
  'px-4 py-3 max-sm:flex max-sm:items-start max-sm:justify-between max-sm:gap-4 max-sm:p-0 max-sm:before:shrink-0 max-sm:before:text-xs max-sm:before:font-medium max-sm:before:text-slate-600 max-sm:before:content-[attr(data-label)]'

export function EmployeesTable({ employees, filtered = false }: { employees: EmployeeSummary[]; filtered?: boolean }) {
  const navigate = useNavigate()
  const [toggling, setToggling] = useState<EmployeeSummary | null>(null)

  if (employees.length === 0) {
    return (
      <EmptyState title={filtered ? 'No matching employees' : 'No employees yet'} description={filtered ? 'Try different search terms or filters.' : 'Invite people to add them to your organization.'} />
    )
  }

  return (
    <>
      <div className="overflow-x-auto">
        <table className="w-full text-left max-sm:block sm:min-w-[56rem]">
          <thead className="bg-slate-100 text-xs font-medium text-slate-600 max-sm:sr-only">
            <tr>
              <th scope="col" className="px-4 py-3">Employee</th>
              <th scope="col" className="px-4 py-3">Department</th>
              <th scope="col" className="px-4 py-3">Position</th>
              <th scope="col" className="px-4 py-3">Employment</th>
              <th scope="col" className="px-4 py-3">Status</th>
              <th scope="col" className="px-4 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="max-sm:block">
            {employees.map((e) => {
              const name = fullName(e)
              return (
                <tr
                  key={e.id}
                  className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50 max-sm:flex max-sm:flex-col max-sm:gap-2 max-sm:p-4"
                >
                  <td className="px-4 py-3 max-sm:p-0 max-sm:break-words">
                    <Link to={`/employees/${e.id}`} className="font-medium text-slate-900 hover:text-slate-700">
                      {name}
                    </Link>
                    <p className="text-xs text-slate-600">
                      {e.email}
                      {e.employee_code ? ` · ${e.employee_code}` : ''}
                    </p>
                  </td>
                  <td className={CELL} data-label="Department">{e.department_name || '-'}</td>
                  <td className={CELL} data-label="Position">{e.position_name || '-'}</td>
                  <td className={CELL} data-label="Employment">
                    <span className="flex flex-col items-start gap-1">
                      <EmploymentStatusBadge status={e.employment_status} />
                      {e.employment_type && <span className="text-xs text-slate-600">{humanize(e.employment_type)}</span>}
                    </span>
                  </td>
                  <td className={CELL} data-label="Status">
                    <span className="flex flex-col items-start gap-1">
                      <EmployeeStatusBadge active={e.is_active} />
                      {!e.profile_completed && <span className="text-xs text-slate-600">Profile incomplete</span>}
                    </span>
                  </td>
                  <td className="px-4 py-3 max-sm:p-0 max-sm:pt-1">
                    <div className="sm:flex sm:justify-end">
                      <RowActionsMenu
                        label={`employee ${name}`}
                        actions={[
                          { label: 'View', onSelect: () => navigate(`/employees/${e.id}`) },
                          { label: e.is_active ? 'Deactivate' : 'Activate', danger: e.is_active, onSelect: () => setToggling(e) },
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
      {toggling?.id && (
        <ToggleActiveDialog
          employeeId={toggling.id}
          name={fullName(toggling)}
          activate={!toggling.is_active}
          onClose={() => setToggling(null)}
        />
      )}
    </>
  )
}
