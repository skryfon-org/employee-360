import { EmptyState, RowActionsMenu } from '@employee360/ui'
import { useState } from 'react'
import { getErrorMessage, type InvitationListItem } from '@employee360/api-client'
import { useResendInvitationMutation, useRevokeInvitationMutation } from '../queries/invitationQueries'
import { StatusBadge } from './StatusBadge'
import { ConfirmRevokeModal } from './ConfirmRevokeModal'
import { useToast } from '../../../hooks/useToast'

const CELL =
  "px-4 py-3 max-sm:flex max-sm:items-center max-sm:justify-between max-sm:p-0 max-sm:before:text-xs max-sm:before:font-medium max-sm:before:text-slate-600 max-sm:before:content-[attr(data-label)]"

const fmt = (iso?: string) => (iso ? new Date(iso).toLocaleDateString() : '-')


export function InvitationsTable({ invitations, filtered = false }: { invitations: InvitationListItem[]; filtered?: boolean }) {
  const toast = useToast()
  const resend = useResendInvitationMutation()
  const revoke = useRevokeInvitationMutation()
  const [revoking, setRevoking] = useState<InvitationListItem | null>(null)

  if (invitations.length === 0) {
    return (
      <EmptyState title={filtered ? 'No invitations match your filters' : 'No invitations yet'} description={filtered ? 'Try a different status or search term.' : 'Create an invitation to onboard your first user.'} />
    )
  }

  return (
    <>
      <div className="overflow-x-auto">
      <table className="w-full text-left max-sm:block sm:min-w-[64rem]">
        <thead className="bg-slate-100 text-xs font-medium text-slate-600 max-sm:sr-only">
          <tr>
            <th scope="col" className="px-4 py-3">Email</th>
            <th scope="col" className="px-4 py-3">Role</th>
            <th scope="col" className="px-4 py-3">Invited by</th>
            <th scope="col" className="px-4 py-3">Invited on</th>
            <th scope="col" className="px-4 py-3">Expires on</th>
            <th scope="col" className="px-4 py-3">Status</th>
            <th scope="col" className="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="max-sm:block">
          {invitations.map((inv) => {
            const actionable = inv.status === 'pending' || inv.status === 'expired'
            return (
              <tr key={inv.id} className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50 max-sm:flex max-sm:flex-col max-sm:gap-2 max-sm:p-4">
                <td className="px-4 py-3 max-sm:p-0 max-sm:break-all max-sm:font-medium">{inv.email}</td>
                <td className={CELL} data-label="Role">{inv.role || '-'}</td>
                <td className={CELL} data-label="Invited by">{inv.invited_by?.name || inv.invited_by?.email || '-'}</td>
                <td className={CELL} data-label="Invited on">{fmt(inv.invited_on ?? inv.created_at)}</td>
                <td className={CELL} data-label="Expires on">{fmt(inv.expires_at)}</td>
                <td className={CELL} data-label="Status">
                  <StatusBadge status={inv.status} />
                </td>
                <td className="px-4 py-3 max-sm:p-0 max-sm:pt-1 max-sm:empty:hidden">
                  {actionable && inv.id && (
                    <div className="sm:flex sm:justify-end">
                      <RowActionsMenu
                        label={`invitation for ${inv.email}`}
                        actions={[
                          {
                            label: 'Resend',
                            disabled: resend.isPending,
                            onSelect: () =>
                              resend.mutate(inv.id as string, {
                                onSuccess: () => toast.success(`Invitation resent to ${inv.email}.`),
                                onError: (err) => toast.error(getErrorMessage(err, 'Could not resend invitation.')),
                              }),
                          },
                          {
                            label: 'Revoke',
                            danger: true,
                            onSelect: () => {
                              revoke.reset()
                              setRevoking(inv)
                            },
                          },
                        ]}
                      />
                    </div>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      </div>
      {revoking && (
        <ConfirmRevokeModal
          email={revoking.email ?? ''}
          loading={revoke.isPending}
          error={revoke.isError ? getErrorMessage(revoke.error, 'Could not revoke invitation.') : undefined}
          onCancel={() => setRevoking(null)}
          onConfirm={() =>
            revoke.mutate(revoking.id as string, {
              onSuccess: () => {
                toast.success(`Invitation for ${revoking.email} revoked.`)
                setRevoking(null)
              },
              onError: (err) => toast.error(getErrorMessage(err, 'Could not revoke invitation.')),
            })
          }
        />
      )}
    </>
  )
}
