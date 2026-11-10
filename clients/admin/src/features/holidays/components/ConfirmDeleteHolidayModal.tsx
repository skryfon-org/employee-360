import { ConfirmDialog } from '../../tenants/components/ConfirmDialog'

export function ConfirmDeleteHolidayModal({
  name,
  loading,
  error,
  onConfirm,
  onCancel,
}: {
  name: string
  loading: boolean
  error?: string
  onConfirm: () => void
  onCancel: () => void
}) {
  return (
    <ConfirmDialog title="Delete holiday" confirmLabel="Delete" loading={loading} error={error} onConfirm={onConfirm} onCancel={onCancel}>
      Delete <span className="break-words font-medium text-slate-900">{name}</span>? Employees will no longer see it.
    </ConfirmDialog>
  )
}
