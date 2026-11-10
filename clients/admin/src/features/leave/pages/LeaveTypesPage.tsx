import { useState } from 'react'
import { Link } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, InlineAlert, PageContainer, PageHeader, RowActionsMenu } from '@employee360/ui'
import { useToast } from '../../../hooks/useToast'
import { ConfirmDialog } from '../../tenants/components/ConfirmDialog'
import { LeaveTypeFormModal } from '../components/LeaveTypeFormModal'
import { PRIMARY_BTN, SECONDARY_BTN } from '../components/styles'
import { useLeaveTypesQuery, useUpdateLeaveTypeMutation, type LeaveType } from '../queries/leaveQueries'

type Dialog = { kind: 'create' } | { kind: 'edit'; type: LeaveType } | { kind: 'deactivate'; type: LeaveType } | null

function StatusBadge({ active }: { active: boolean }) {
  const cls = active ? 'bg-green-50 border-green-300 text-green-700' : 'bg-slate-100 border-slate-300 text-slate-600'
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${cls}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${active ? 'bg-green-700' : 'bg-slate-600'}`} aria-hidden="true" />
      {active ? 'Active' : 'Inactive'}
    </span>
  )
}

export default function LeaveTypesPage() {
  const toast = useToast()
  const query = useLeaveTypesQuery()
  const update = useUpdateLeaveTypeMutation()
  const [dialog, setDialog] = useState<Dialog>(null)
  const types = query.data

  const setActive = (t: LeaveType, active: boolean) =>
    update.mutate(
      { id: t.id, body: { is_active: active } },
      {
        onSuccess: () => {
          toast.success(`Leave type "${t.name}" ${active ? 'activated' : 'deactivated'}.`)
          setDialog(null)
        },
        onError: (err) => toast.error(getErrorMessage(err, `Could not ${active ? 'activate' : 'deactivate'} leave type.`)),
      },
    )

  return (
    <PageContainer>
      <PageHeader
        title="Leave types"
        description="Types employees can choose when applying for leave. Types are deactivated, never deleted."
        actions={
          <>
            <Link to="/leave" className={SECONDARY_BTN}>
              Back to requests
            </Link>
            <button type="button" className={PRIMARY_BTN} onClick={() => setDialog({ kind: 'create' })}>
              Add leave type
            </button>
          </>
        }
      />
      <Card padded={false} role="region" aria-label="Leave types" className="overflow-hidden">
        {query.isPending && <p className="p-6 text-sm text-slate-600">Loading leave types...</p>}
        {query.isError && (
          <div className="flex flex-col items-start gap-3 p-4">
            <InlineAlert tone="error">{getErrorMessage(query.error, 'Could not load leave types.')}</InlineAlert>
            <button type="button" className={SECONDARY_BTN} onClick={() => void query.refetch()}>
              Retry
            </button>
          </div>
        )}
        {types && types.length === 0 && (
          <EmptyState title="No leave types yet" description="Add one so employees can apply for leave." />
        )}
        {types && types.length > 0 && (
          <div className="overflow-x-auto">
            <table aria-label="Leave types" className="w-full min-w-[36rem] text-left">
              <thead className="bg-slate-100 text-xs font-medium text-slate-600">
                <tr>
                  <th scope="col" className="px-4 py-3">Name</th>
                  <th scope="col" className="px-4 py-3">Code</th>
                  <th scope="col" className="px-4 py-3">Description</th>
                  <th scope="col" className="px-4 py-3">Status</th>
                  <th scope="col" className="px-4 py-3 text-right"><span className="sr-only">Actions</span></th>
                </tr>
              </thead>
              <tbody>
                {types.map((t) => (
                  <tr key={t.id} className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50">
                    <td className="px-4 py-3 font-medium">{t.name}</td>
                    <td className="px-4 py-3">{t.code}</td>
                    <td className="px-4 py-3 text-slate-600 sm:max-w-md sm:break-words">{t.description || '-'}</td>
                    <td className="px-4 py-3">
                      <StatusBadge active={t.is_active} />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end">
                        <RowActionsMenu
                          label={t.name}
                          actions={[
                            { label: 'Edit', onSelect: () => setDialog({ kind: 'edit', type: t }) },
                            t.is_active
                              ? { label: 'Deactivate', danger: true, onSelect: () => setDialog({ kind: 'deactivate', type: t }) }
                              : { label: 'Activate', onSelect: () => setActive(t, true) },
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
      </Card>
      {(dialog?.kind === 'create' || dialog?.kind === 'edit') && (
        <LeaveTypeFormModal leaveType={dialog.kind === 'edit' ? dialog.type : undefined} onClose={() => setDialog(null)} />
      )}
      {dialog?.kind === 'deactivate' && (
        <ConfirmDialog
          title={`Deactivate ${dialog.type.name}?`}
          confirmLabel="Deactivate"
          loading={update.isPending}
          error={update.isError ? getErrorMessage(update.error, 'Could not deactivate leave type.') : undefined}
          onCancel={() => setDialog(null)}
          onConfirm={() => setActive(dialog.type, false)}
        >
          Employees will no longer be able to choose this type for new requests. Existing requests are kept.
        </ConfirmDialog>
      )}
    </PageContainer>
  )
}
