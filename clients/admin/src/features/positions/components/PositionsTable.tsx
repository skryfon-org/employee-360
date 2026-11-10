import { EmptyState, RowActionsMenu } from '@employee360/ui'
import { getErrorCode, getErrorMessage } from '@employee360/api-client'
import { useState } from 'react'
import { useDeletePositionMutation, type Position } from '../queries/positionQueries'
import { ConfirmDeletePositionModal } from './ConfirmDeletePositionModal'
import { PositionStatusBadge } from './PositionStatusBadge'
import { PositionFormModal } from './PositionFormModal'
import { useToast } from '../../../hooks/useToast'

const CELL =
  'px-4 py-3 max-sm:flex max-sm:items-start max-sm:justify-between max-sm:gap-4 max-sm:p-0 max-sm:before:shrink-0 max-sm:before:text-xs max-sm:before:font-medium max-sm:before:text-slate-600 max-sm:before:content-[attr(data-label)]'

const fmt = (iso?: string) => (iso ? new Date(iso).toLocaleDateString() : '-')


export const IN_USE_MESSAGE =
  'This position is still assigned to users or pending invitations. Reassign them before deleting it.'

export function PositionsTable({ positions, filtered = false }: { positions: Position[]; filtered?: boolean }) {
  const toast = useToast()
  const del = useDeletePositionMutation()
  const [editing, setEditing] = useState<Position | null>(null)
  const [deleting, setDeleting] = useState<Position | null>(null)

  if (positions.length === 0) {
    return (
      <EmptyState title={filtered ? 'No matching positions' : 'No positions yet'} description={filtered ? 'Try a different status filter.' : 'Create a position to organise your people.'} />
    )
  }

  // The backend returns CONFLICT for any 409; on delete that can only mean "in use".
  const deleteError = del.isError
    ? getErrorCode(del.error) === 'CONFLICT'
      ? IN_USE_MESSAGE
      : getErrorMessage(del.error, 'Could not delete position.')
    : undefined

  return (
    <>
      <div className="overflow-x-auto">
        <table className="w-full text-left max-sm:block sm:min-w-[46rem]">
          <thead className="bg-slate-100 text-xs font-medium text-slate-600 max-sm:sr-only">
            <tr>
              <th scope="col" className="px-4 py-3">Name</th>
              <th scope="col" className="px-4 py-3">Description</th>
              <th scope="col" className="px-4 py-3">Status</th>
              <th scope="col" className="px-4 py-3">Updated</th>
              <th scope="col" className="px-4 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="max-sm:block">
            {positions.map((d) => (
              <tr
                key={d.id}
                className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50 max-sm:flex max-sm:flex-col max-sm:gap-2 max-sm:p-4"
              >
                <td className="px-4 py-3 font-medium max-sm:p-0 max-sm:break-words">{d.name}</td>
                <td className={`${CELL} text-slate-600 sm:max-w-md sm:break-words`} data-label="Description">
                  {d.description || '-'}
                </td>
                <td className={CELL} data-label="Status">
                  <PositionStatusBadge active={d.is_active} />
                </td>
                <td className={CELL} data-label="Updated">{fmt(d.updated_at)}</td>
                <td className="px-4 py-3 max-sm:p-0 max-sm:pt-1">
                  <div className="sm:flex sm:justify-end">
                    <RowActionsMenu
                      label={`position ${d.name}`}
                      actions={[
                        { label: 'Edit', onSelect: () => setEditing(d) },
                        {
                          label: 'Delete',
                          danger: true,
                          onSelect: () => {
                            del.reset()
                            setDeleting(d)
                          },
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
      {editing && <PositionFormModal position={editing} onClose={() => setEditing(null)} />}
      {deleting && (
        <ConfirmDeletePositionModal
          name={deleting.name ?? ''}
          loading={del.isPending}
          error={deleteError}
          onCancel={() => setDeleting(null)}
          onConfirm={() =>
            del.mutate(deleting.id as string, {
              onSuccess: () => {
                toast.success(`Position "${deleting.name}" deleted.`)
                setDeleting(null)
              },
              onError: (err) => {
                if (getErrorCode(err) !== 'CONFLICT') toast.error(getErrorMessage(err, 'Could not delete position.'))
              },
            })
          }
        />
      )}
    </>
  )
}
