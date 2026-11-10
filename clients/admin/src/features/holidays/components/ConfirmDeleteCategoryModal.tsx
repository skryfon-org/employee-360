import { ConfirmDialog } from '../../tenants/components/ConfirmDialog'

export function ConfirmDeleteCategoryModal({
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
    <ConfirmDialog title="Delete category" confirmLabel="Delete" loading={loading} error={error} onConfirm={onConfirm} onCancel={onCancel}>
      Delete <span className="break-words font-medium text-slate-900">{name}</span>? This cannot be undone.
    </ConfirmDialog>
  )
}
