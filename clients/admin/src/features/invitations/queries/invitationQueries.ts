import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  inviteUser,
  listInvitations,
  listRoles,
  resendInvitation,
  revokeInvitation,
  type InviteUserRequest,
} from '@employee360/api-client'
import type { InvitationListParams } from '../schemas/invitationListSchema'

export const INVITATIONS_KEY = ['invitations'] as const

export function useInvitationsQuery(params: InvitationListParams) {
  return useQuery({
    queryKey: [...INVITATIONS_KEY, params],
    queryFn: ({ signal }) => listInvitations(params, signal),
    placeholderData: keepPreviousData,
  })
}

export const ROLES_KEY = ['roles'] as const

export function useRolesQuery() {
  return useQuery({
    queryKey: ROLES_KEY,
    queryFn: ({ signal }) => listRoles(signal),
    staleTime: 5 * 60 * 1000,
  })
}

function useInvalidatingMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: INVITATIONS_KEY }),
  })
}

export const useInviteUserMutation = () =>
  useInvalidatingMutation((body: InviteUserRequest) => inviteUser(body))
export const useResendInvitationMutation = () =>
  useInvalidatingMutation((id: string) => resendInvitation(id))
export const useRevokeInvitationMutation = () =>
  useInvalidatingMutation((id: string) => revokeInvitation(id))
