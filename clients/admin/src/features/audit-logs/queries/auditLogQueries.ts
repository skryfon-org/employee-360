import { keepPreviousData, useQuery } from '@tanstack/react-query'
import {
  getApiV1AuditLogs,
  unwrapListResponse,
  type GetApiV1AuditLogsParams,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesAuditlogAuditLogResponse as AuditLogEntry,
} from '@employee360/api-client'
import { endOfDayRfc3339, startOfDayRfc3339, type AuditLogListParams } from '../schemas/auditLogSchemas'

export type { AuditLogEntry }

export const AUDIT_LOGS_KEY = ['audit-logs'] as const

/** Maps URL/UI params to the API contract (dates become inclusive RFC3339 bounds). */
export function toApiParams(p: AuditLogListParams): GetApiV1AuditLogsParams {
  return {
    page: p.page,
    page_size: p.page_size,
    ...(p.entity_type && { entity_type: p.entity_type }),
    ...(p.action && { action: p.action }),
    ...(p.from && { from: startOfDayRfc3339(p.from) }),
    ...(p.to && { to: endOfDayRfc3339(p.to) }),
  }
}

export function useAuditLogsQuery(params: AuditLogListParams) {
  const api = toApiParams(params)
  return useQuery({
    queryKey: [...AUDIT_LOGS_KEY, api],
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1AuditLogs(api, signal)),
    placeholderData: keepPreviousData,
  })
}
