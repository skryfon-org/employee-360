import { useQueryClient } from '@tanstack/react-query'
import {
  getErrorCode,
  getErrorMessage,
  getGetApiV1MeStatusLogsDayStatusesQueryKey,
  getGetApiV1MeStatusLogsQueryKey,
  useDeleteApiV1MeStatusLogsId,
  useGetApiV1MeProjects,
  useGetApiV1MeStatusLogsDayStatuses,
  useGetApiV1MeStatusLogs,
  usePatchApiV1MeStatusLogsId,
  usePostApiV1MeStatusLogs,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogCreateStatusLogRequest as CreateStatusLog,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogDailyStatusLogGroupResponse as DayGroup,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogStatusLogEntryResponse as StatusLogEntry,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogUpdateStatusLogRequest as UpdateStatusLog,
} from '@employee360/api-client'
import { EMPLOYEE_DASHBOARD_KEY } from '../../dashboard/queries/dashboardQueries'

export type { CreateStatusLog, DayGroup, StatusLogEntry, UpdateStatusLog }

/** The caller's day groups between two ISO dates (inclusive). */
export function useMyStatusLogsQuery(from: string, to: string, enabled = true) {
  return useGetApiV1MeStatusLogs({ from, to }, { query: { enabled } })
}

/** Why each day in the range has (or lacks) a log: logged, holiday, weekly off, on leave or missing. Days after today are omitted by the server. */
export function useMyDayStatusesQuery(from: string, to: string, enabled = true) {
  return useGetApiV1MeStatusLogsDayStatuses({ from, to }, { query: { enabled } })
}

/** All of the caller's allocations; the server enforces an active allocation on the log date. */
export function useLoggableProjectsQuery() {
  return useGetApiV1MeProjects({ scope: 'all' })
}

function useInvalidate() {
  const qc = useQueryClient()
  return () => Promise.all([qc.invalidateQueries({ queryKey: getGetApiV1MeStatusLogsQueryKey() }), qc.invalidateQueries({ queryKey: getGetApiV1MeStatusLogsDayStatusesQueryKey() }), qc.invalidateQueries({ queryKey: EMPLOYEE_DASHBOARD_KEY })])
}

export function useCreateStatusLogMutation() {
  const invalidate = useInvalidate()
  return usePostApiV1MeStatusLogs({ mutation: { onSuccess: invalidate } })
}

export function useUpdateStatusLogMutation() {
  const invalidate = useInvalidate()
  return usePatchApiV1MeStatusLogsId({ mutation: { onSuccess: invalidate } })
}

export function useDeleteStatusLogMutation() {
  const invalidate = useInvalidate()
  return useDeleteApiV1MeStatusLogsId({ mutation: { onSuccess: invalidate } })
}

const CODE_MESSAGES: Record<string, string> = {
  log_window_closed: 'This day is outside the 7-day logging window, so it can no longer be changed.',
  daily_hours_exceeded: 'The total for this day cannot exceed 24 hours.',
  future_date_not_allowed: 'You cannot log status for a future date.',
  no_active_allocation: 'You need an active allocation on this project for that date.',
}

/** User-facing text for a status-log API error, keyed on the stable error code. */
export function statusLogErrorMessage(err: unknown): string {
  const code = getErrorCode(err)
  if (code && CODE_MESSAGES[code]) return CODE_MESSAGES[code]
  return getErrorMessage(err, 'Could not save your status entry.')
}
