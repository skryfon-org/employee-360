import { useState, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { Card } from '@employee360/ui'
import { EmployeeAllocations } from '../../projects/components/EmployeeAllocations'
import { fullName } from '../employeeName'
import { EmployeeLeaveRequests } from '../../leave/components/EmployeeLeaveRequests'
import { EmployeeStatusRange } from '../../statusLogs/components/EmployeeStatusRange'
import { AssignmentHistory } from '../components/AssignmentHistory'
import { EmergencyContactsEditor } from '../components/EmergencyContactsEditor'
import { EmploymentForm } from '../components/EmploymentForm'
import { PersonalAddressForm } from '../components/PersonalAddressForm'
import { RoleControl } from '../components/RoleControl'
import { DefinitionList, Section, secondaryButtonCls } from '../components/fields'
import { useDepartmentOptionsQuery, useManagerOptionsQuery, usePositionOptionsQuery } from '../queries/employeeQueries'
import { humanize } from '../schemas/employeeSchemas'
import { useEmployeeDetail } from '../hooks/useEmployeeDetail'

/**
 * Card that opens in read-only view mode. "Edit" (hidden when the viewer cannot edit) swaps in the form;
 * the form calls `done` on a successful save or Cancel. Edit-mode state is local to the section.
 */
function EditableCard({
  label,
  canEdit = true,
  view,
  edit,
}: {
  label: string
  canEdit?: boolean
  view: ReactNode
  edit: (done: () => void) => ReactNode
}) {
  const { readOnly } = useEmployeeDetail()
  const [editing, setEditing] = useState(false)
  if (editing && !readOnly && canEdit) return <Card>{edit(() => setEditing(false))}</Card>
  return (
    <Card>
      <div className="flex flex-col gap-4">
        {view}
        {!readOnly && canEdit && (
          <div className="flex justify-end sticky bottom-0 border-t border-slate-200 bg-white pt-3">
            <button type="button" className={secondaryButtonCls} onClick={() => setEditing(true)}>
              Edit {label}
            </button>
          </div>
        )}
      </div>
    </Card>
  )
}

export function OverviewSection() {
  const { employee: e } = useEmployeeDetail()
  const rows: [string, string | undefined][] = [
    ['Employee code', e.employee_code],
    ['Department', e.department_name],
    ['Position', e.position_name],
    ['Employment type', e.employment_type ? humanize(e.employment_type) : undefined],
    ['Employment status', e.employment_status ? humanize(e.employment_status) : undefined],
    ['Joining date', e.joining_date],
    ['Work location', e.work_location],
    ['Phone', e.profile?.phone],
  ]
  return (
    <Card>
      <Section title="Overview" description="Summary of this employee. Use the tabs above to edit details.">
        <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {rows.map(([label, value]) => (
            <div key={label} className="flex flex-col gap-1">
              <dt className="text-xs font-medium text-slate-600">{label}</dt>
              <dd className="text-sm text-slate-900">{value || '-'}</dd>
            </div>
          ))}
        </dl>
      </Section>
    </Card>
  )
}

export function EmploymentSection() {
  const { employee: e, formKey } = useEmployeeDetail()
  const managers = useManagerOptionsQuery().data ?? []
  const departments = useDepartmentOptionsQuery().data ?? []
  const positions = usePositionOptionsQuery().data ?? []
  const p = e.profile ?? {}
  const m = managers.find((x) => x.id === (e.manager_id ?? p.manager_id))
  const type = e.employment_type ?? p.employment_type
  const status = e.employment_status ?? p.employment_status
  return (
    <EditableCard
      key={`emp-${formKey}`}
      label="employment"
      view={
        <Section title="Employment" description="Only admins can change these fields.">
          <DefinitionList
            rows={[
              ['Employee code', e.employee_code ?? p.employee_code],
              ['Work location', e.work_location ?? p.work_location],
              ['Employment type', type ? humanize(type) : undefined],
              ['Employment status', status ? humanize(status) : undefined],
              ['Joining date', e.joining_date ?? p.joining_date],
              ['Probation end date', p.probation_end_date],
              ['Confirmation date', p.confirmation_date],
              ['Exit date', p.exit_date],
              ['Department', e.department_name ?? departments.find((d) => d.id === e.department_id)?.name],
              ['Position', e.position_name ?? positions.find((x) => x.id === e.position_id)?.name],
              ['Manager', m ? [m.first_name, m.last_name].filter(Boolean).join(' ') || m.email : undefined],
            ]}
          />
        </Section>
      }
      edit={(done) => <EmploymentForm employee={e} onDone={done} />}
    />
  )
}

export function PersonalSection() {
  const { employee: e, formKey } = useEmployeeDetail()
  const p = e.profile ?? {}
  return (
    <EditableCard
      key={`personal-${formKey}`}
      label="personal details"
      view={
        <Section title="Personal details" description="Admins can correct these on behalf of the employee.">
          <DefinitionList
            rows={[
              ['First name', e.first_name],
              ['Last name', e.last_name],
              ['Phone', p.phone],
              ['Personal email', p.personal_email],
              ['Date of birth', p.date_of_birth],
              ['Gender', p.gender ? humanize(p.gender) : undefined],
              ['Blood group', p.blood_group],
              ['Marital status', p.marital_status ? humanize(p.marital_status) : undefined],
              ['Nationality', p.nationality],
            ]}
          />
        </Section>
      }
      edit={(done) => <PersonalAddressForm employee={e} part="personal" onDone={done} />}
    />
  )
}

export function AddressSection() {
  const { employee: e, formKey } = useEmployeeDetail()
  const p = e.profile ?? {}
  return (
    <EditableCard
      key={`address-${formKey}`}
      label="address"
      view={
        <Section title="Address" description="Admins can correct these on behalf of the employee.">
          <DefinitionList
            rows={[
              ['Address line 1', p.address_line1],
              ['Address line 2', p.address_line2],
              ['City', p.city],
              ['State', p.state],
              ['Country', p.country],
              ['Pincode', p.pincode],
            ]}
          />
        </Section>
      }
      edit={(done) => <PersonalAddressForm employee={e} part="address" onDone={done} />}
    />
  )
}

export function EmergencyContactsSection() {
  const { employee, readOnly } = useEmployeeDetail()
  const [editing, setEditing] = useState(false)
  const canEdit = !readOnly
  return (
    <Card>
      <div className="flex flex-col gap-4">
        <EmergencyContactsEditor employeeId={employee.id as string} contacts={employee.emergency_contacts ?? []} editing={editing && canEdit} />
        {canEdit && (
          <div className="flex justify-end sticky bottom-0 border-t border-slate-200 bg-white pt-3">
            <button type="button" className={secondaryButtonCls} onClick={() => setEditing((v) => !v)}>
              {editing ? 'Done editing contacts' : 'Edit emergency contacts'}
            </button>
          </div>
        )}
      </div>
    </Card>
  )
}

export function AssignmentsSection() {
  const { employee } = useEmployeeDetail()
  return (
    <Card>
      <AssignmentHistory employeeId={employee.id as string} />
    </Card>
  )
}

export function AllocationsSection() {
  const { employee } = useEmployeeDetail()
  return (
    <EmployeeAllocations
      employeeId={employee.id as string}
      employeeName={fullName(employee)}
      active={employee.is_active !== false}
    />
  )
}

export function RoleSection() {
  const { employee, canChangeRole } = useEmployeeDetail()
  if (!canChangeRole) return <Navigate to=".." relative="path" replace />
  const name = [employee.first_name, employee.last_name].filter(Boolean).join(' ') || (employee.email ?? '')
  const roles = employee.roles ?? []
  return (
    <EditableCard
      label="role"
      view={
        <Section title="Role" description="Super admins can grant or remove admin access.">
          <DefinitionList rows={[['Current role', roles.includes('admin') ? 'Admin' : 'Employee']]} />
        </Section>
      }
      edit={(done) => <RoleControl employeeId={employee.id as string} name={name} roles={roles} onDone={done} />}
    />
  )
}

export function StatusLogsSection() {
  const { employee } = useEmployeeDetail()
  return (
    <Section title="Status logs" description="Daily status entries logged by this employee.">
      <EmployeeStatusRange employeeId={employee.id as string} />
    </Section>
  )
}

export function LeaveSection() {
  const { employee } = useEmployeeDetail()
  return (
    <Section title="Leave" description="Leave requests submitted by this employee.">
      <EmployeeLeaveRequests employeeId={employee.id as string} />
    </Section>
  )
}
