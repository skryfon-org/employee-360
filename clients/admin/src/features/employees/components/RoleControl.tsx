import { useState } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { ConfirmDialog } from '../../tenants/components/ConfirmDialog'
import { useToast } from '../../../hooks/useToast'
import { useChangeRoleMutation, type AssignableRole } from '../queries/employeeQueries'
import { SelectField, secondaryButtonCls } from './fields'

/**
 * Role change control. Backend rule: only super_admin may change roles (grant or revoke admin),
 * never on oneself, and super_admin accounts are untouchable. The caller decides visibility.
 */
export function RoleControl({ employeeId, name, roles, onDone }: { employeeId: string; name: string; roles: string[]; onDone?: () => void }) {
  const toast = useToast()
  const change = useChangeRoleMutation()
  const current: AssignableRole = roles.includes('admin') ? 'admin' : 'employee'
  const [selected, setSelected] = useState<AssignableRole>(current)
  const [confirming, setConfirming] = useState(false)

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
      <div className="sm:w-48">
        <SelectField label="Role" id="employee-role" value={selected} onChange={(e) => setSelected(e.target.value as AssignableRole)}>
          <option value="employee">Employee</option>
          <option value="admin">Admin</option>
        </SelectField>
      </div>
      <button
        type="button"
        className={secondaryButtonCls}
        disabled={selected === current}
        onClick={() => {
          change.reset()
          setConfirming(true)
        }}
      >
        Change role
      </button>
      {onDone && (
        <button type="button" className={secondaryButtonCls} onClick={() => { setSelected(current); onDone() }}>
          Cancel
        </button>
      )}
      {confirming && (
        <ConfirmDialog
          title={selected === 'admin' ? `Grant admin access to ${name}?` : `Remove admin access from ${name}?`}
          confirmLabel={selected === 'admin' ? 'Grant admin' : 'Remove admin'}
          tone={selected === 'admin' ? 'neutral' : 'danger'}
          loading={change.isPending}
          error={change.isError ? getErrorMessage(change.error, 'Could not change role.') : undefined}
          onCancel={() => {
            setConfirming(false)
            setSelected(current)
          }}
          onConfirm={() =>
            change.mutate(
              { id: employeeId, role: selected },
              {
                onSuccess: () => {
                  toast.success(`Role updated for ${name}.`)
                  setConfirming(false)
                  onDone?.()
                },
              },
            )
          }
        >
          {selected === 'admin'
            ? 'Admins can manage employees, departments, positions and holidays for the organization.'
            : 'They will return to the employee role and lose admin access.'}
        </ConfirmDialog>
      )}
    </div>
  )
}
