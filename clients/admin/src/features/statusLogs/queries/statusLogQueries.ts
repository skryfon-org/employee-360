import { keepPreviousData, useQuery } from '@tanstack/react-query'
import {
  getApiV1Departments,
  getApiV1EmployeesIdStatusLogs,
  getApiV1StatusLogsDaily,
  getApiV1StatusLogsMonthly,
  listProjects,
  unwrapListResponse,
  unwrapSingleEntity,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogDailyEmployeeSummaryResponse as DailyEmployeeSummary,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogDailyStatusLogGroupResponse as DayGroup,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogStatusLogEntryResponse as StatusLogEntry,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogMonthlyStatusLogReportResponse,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogMonthlyEmployeeSummaryResponse,
} from '@employee360/api-client'

type MonthlyReport = GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogMonthlyStatusLogReportResponse
type MonthlyEmployeeSummary = GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogMonthlyEmployeeSummaryResponse
export type { MonthlyReport, MonthlyEmployeeSummary }
export type { DailyEmployeeSummary, DayGroup, StatusLogEntry }

export const STATUS_LOGS_KEY = ['status-logs'] as const

/** Every active employee's summary for a date (logged or not), optionally narrowed by department/project. */
export function useDailySummaryQuery(date: string, department: string, project: string) {
  return useQuery({
    queryKey: [...STATUS_LOGS_KEY, 'daily', date, department, project],
    queryFn: async ({ signal }) =>
      unwrapListResponse(
        await getApiV1StatusLogsDaily({ date, department: department || undefined, project: project || undefined }, signal),
      ).data,
    placeholderData: keepPreviousData,
  })
}

/** Org-wide report for a YYYY-MM month, optionally narrowed by department/project. */
export function useMonthlyReportQuery(month: string, department: string, project: string) {
  return useQuery({
    queryKey: [...STATUS_LOGS_KEY, 'monthly', month, department, project],
    queryFn: async ({ signal }) =>
      unwrapSingleEntity(
        await getApiV1StatusLogsMonthly({ month, department: department || undefined, project: project || undefined }, signal),
      ),
    placeholderData: keepPreviousData,
  })
}

/** One employee's day groups between two ISO dates (inclusive). */
export function useEmployeeStatusLogsQuery(employeeId: string | undefined, from: string, to: string, enabled = true) {
  return useQuery({
    queryKey: [...STATUS_LOGS_KEY, 'employee', employeeId, from, to],
    queryFn: async ({ signal }) =>
      unwrapListResponse(await getApiV1EmployeesIdStatusLogs(employeeId as string, { from, to }, signal)).data,
    enabled: enabled && Boolean(employeeId) && Boolean(from) && Boolean(to) && from <= to,
  })
}

export function useDepartmentFilterOptions() {
  return useQuery({
    queryKey: ['departments', 'status-log-options'],
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1Departments({ page: 1, page_size: 100 }, signal)).data,
  })
}

export function useProjectFilterOptions() {
  return useQuery({
    queryKey: ['projects', 'status-log-options'],
    queryFn: async ({ signal }) => (await listProjects({ page: 1, page_size: 100 }, signal)).data,
  })
}
