import { getErrorMessage } from '@employee360/api-client'
import { ConfirmDialog } from '../../tenants/components/ConfirmDialog'
import { useToast } from '../../../hooks/useToast'
import { useActivateEmployeeMutation, useDeactivateEmployeeMutation } from '../queries/employeeQueries'

/** Confirm dialog for activating or deactivating an employee account. */
export function ToggleActiveDialog({
  employeeId,
  name,
  activate,
  onClose,
}: {
  employeeId: string
  name: string
  /** true = activate an inactive account; false = deactivate an active one. */
  activate: boolean
  onClose: () => void
}) {
  const toast = useToast()
  const act = useActivateEmployeeMutation()
  const deact = useDeactivateEmployeeMutation()
  const m = activate ? act : deact
  const verb = activate ? 'Activate' : 'Deactivate'
  const error = m.isError ? getErrorMessage(m.error, `Could not ${verb.toLowerCase()} employee.`) : undefined

  return (
    <ConfirmDialog
      title={`${verb} ${name}?`}
      confirmLabel={verb}
      tone={activate ? 'neutral' : 'danger'}
      loading={m.isPending}
      error={error}
      onCancel={onClose}
      onConfirm={() =>
        m.mutate(employeeId, {
          onSuccess: () => {
            toast.success(`${name} ${activate ? 'activated' : 'deactivated'}.`)
            onClose()
          },
        })
      }
    >
      {activate
        ? 'This employee will be able to sign in again.'
        : 'This employee will no longer be able to sign in. Their data is kept and you can reactivate them at any time.'}
    </ConfirmDialog>
  )
}
