import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getApiV1OrganizationSettings,
  putApiV1OrganizationSettings,
  unwrapSingleEntity,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesOrganizationSettingsResponse as OrgSettings,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesOrganizationUpdateSettingsRequest as UpdateOrgSettingsInput,
} from '@employee360/api-client'

export type { OrgSettings, UpdateOrgSettingsInput }

export const ORG_SETTINGS_KEY = ['organization-settings'] as const

export function useOrgSettingsQuery() {
  return useQuery({
    queryKey: ORG_SETTINGS_KEY,
    queryFn: async ({ signal }) => unwrapSingleEntity(await getApiV1OrganizationSettings(signal)),
  })
}

export function useUpdateOrgSettingsMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: UpdateOrgSettingsInput) => unwrapSingleEntity(await putApiV1OrganizationSettings(body)),
    onSuccess: (data) => {
      qc.setQueryData(ORG_SETTINGS_KEY, data)
      // Weekly off days change the monthly report's working days.
      void qc.invalidateQueries({ queryKey: ['status-logs'] })
      return qc.invalidateQueries({ queryKey: ORG_SETTINGS_KEY })
    },
  })
}
