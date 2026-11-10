import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useToast } from '@employee360/ui'
import {
  deleteApiV1MeEmergencyContactsContactId,
  getApiV1Me,
  getApiV1MeProfile,
  patchApiV1MeProfile,
  postApiV1MeEmergencyContacts,
  putApiV1MeEmergencyContactsContactId,
  unwrapSingleEntity,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeEmergencyContactInput as ContactInput,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeEmergencyContactResponse as EmergencyContact,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeMeResponse as Me,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeMyProfileResponse as MyProfile,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeUpdateMyProfileInput as UpdateMyProfileInput,
} from '@employee360/api-client'
import { ME_KEY } from '../../../lib/queryClient'
import { useAuthStore } from '../../../stores/authStore'

export type { ContactInput, EmergencyContact, Me, MyProfile, UpdateMyProfileInput }

export const MY_PROFILE_KEY = ['my-profile'] as const

/** GET /me: user, roles, organization name and the gating status. Keyed per user so sessions never share it. */
export function useMeQuery() {
  const userId = useAuthStore((s) => s.user?.id)
  return useQuery({
    queryKey: [...ME_KEY, userId],
    queryFn: async ({ signal }) => unwrapSingleEntity(await getApiV1Me(signal)),
    enabled: Boolean(userId),
  })
}

export function useMyProfileQuery(enabled = true) {
  return useQuery({
    queryKey: MY_PROFILE_KEY,
    queryFn: async ({ signal }) => unwrapSingleEntity(await getApiV1MeProfile(signal)),
    enabled,
  })
}

/** Refetch /me now (used when the wizard finishes) so the gate releases on fresh data. */
export function refetchMe(qc: QueryClient) {
  return qc.refetchQueries({ queryKey: ME_KEY })
}

/**
 * Mutations refresh the profile view always. `refreshMe` also invalidates /me, which can flip
 * the route gate; the wizard passes false so completing the last step doesn't yank the page
 * away before the user presses Finish.
 */
function useProfileMutation<TVars, TResult>(fn: (v: TVars) => Promise<TResult>, refreshMe: boolean, successMessage?: string) {
  const qc = useQueryClient()
  const toast = useToast()
  return useMutation({
    mutationFn: fn,
    onSuccess: async () => {
      // Raised here, not via mutate() callbacks: those are dropped when the form unmounts during the refetch below.
      if (successMessage) toast.success(successMessage)
      await qc.invalidateQueries({ queryKey: MY_PROFILE_KEY })
      if (refreshMe) await qc.invalidateQueries({ queryKey: ME_KEY })
    },
  })
}

export const useUpdateMyProfileMutation = (refreshMe = true, successMessage?: string) =>
  useProfileMutation(async (body: UpdateMyProfileInput) => unwrapSingleEntity(await patchApiV1MeProfile(body)), refreshMe, successMessage)

export const useCreateContactMutation = (refreshMe = true) =>
  useProfileMutation(async (body: ContactInput) => unwrapSingleEntity(await postApiV1MeEmergencyContacts(body)), refreshMe, 'Emergency contact added.')

export const useUpdateContactMutation = (refreshMe = true) =>
  useProfileMutation(
    async (v: { contactId: string; body: ContactInput }) =>
      unwrapSingleEntity(await putApiV1MeEmergencyContactsContactId(v.contactId, v.body)),
    refreshMe,
    'Emergency contact updated.',
  )

export const useDeleteContactMutation = (refreshMe = true) =>
  useProfileMutation(async (contactId: string) => {
    await deleteApiV1MeEmergencyContactsContactId(contactId)
  }, refreshMe, 'Emergency contact removed.')
