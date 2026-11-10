import { keepPreviousData, useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  applyLeave,
  approveLeaveRequest,
  cancelLeaveRequest,
  getWeeklyOffDays,
  listLeaveApprovers,
  listLeaveTypes,
  listMyLeaveRequests,
  listTeamLeaveRequests,
  rejectLeaveRequest,
  getApiV1Holidays,
  unwrapListResponse,
  type LeaveStatus,
} from '@employee360/api-client'

export type { LeaveRequest, LeaveApprover, LeaveType, LeaveStatus } from '@employee360/api-client'

import { holidaysYearKey } from '../../holidays/queries/holidayQueries'
import { EMPLOYEE_DASHBOARD_KEY } from '../../dashboard/queries/dashboardQueries'
import { EMPTY_FILTERS, type LeaveFilters } from '../schemas/filters'

const KEY = ['leave'] as const

export const useLeaveTypesQuery = () => useQuery({ queryKey: [...KEY, 'types'], queryFn: ({ signal }) => listLeaveTypes(signal) })

export const useLeaveApproversQuery = () => useQuery({ queryKey: [...KEY, 'approvers'], queryFn: ({ signal }) => listLeaveApprovers(signal) })

/** Tenant weekly off days (0=Sunday..6=Saturday); rarely changes. */
export const useWeeklyOffDaysQuery = () => useQuery({ queryKey: [...KEY, 'weekly-off-days'], queryFn: ({ signal }) => getWeeklyOffDays(signal), staleTime: 5 * 60_000 })

export interface RangeHoliday {
  /** `YYYY-MM-DD`. */
  date: string
  name: string
  /** Only public holidays are excluded from leave. */
  isPublic: boolean
}

const ISO = /^\d{4}-\d{2}-\d{2}$/
const MAX_RANGE_YEARS = 10

/** Distinct calendar years the inclusive `YYYY-MM-DD` range touches; empty when incomplete/invalid/too wide. */
export function yearsInRange(start: string, end: string): number[] {
  if (!ISO.test(start) || !ISO.test(end) || end < start) return []
  const a = Number(start.slice(0, 4))
  const b = Number(end.slice(0, 4))
  if (b - a >= MAX_RANGE_YEARS) return []
  return Array.from({ length: b - a + 1 }, (_, i) => a + i)
}

/**
 * Holidays within the range (one cached call per year, shared with the holidays page cache).
 * Returns undefined until every year has loaded, or if any failed, so callers don't block (server is authoritative).
 */
export function useRangeHolidays(start: string, end: string): RangeHoliday[] | undefined {
  const years = yearsInRange(start, end)
  const results = useQueries({
    queries: years.map((year) => ({
      queryKey: holidaysYearKey(year),
      queryFn: async ({ signal }: { signal?: AbortSignal }) => unwrapListResponse(await getApiV1Holidays({ year }, signal)).data,
      staleTime: 5 * 60_000,
    })),
  })
  if (years.length === 0 || results.some((r) => !r.isSuccess)) return undefined
  return results
    .flatMap((r) => r.data ?? [])
    .map((h) => ({ date: (h.date ?? '').slice(0, 10), name: h.name ?? '', isPublic: h.category_is_public === true }))
    .filter((h) => ISO.test(h.date) && h.date >= start && h.date <= end)
    .sort((x, y) => x.date.localeCompare(y.date))
}

export const PAGE_SIZE = 20

const toParams = (f: LeaveFilters) => ({ status: f.status || undefined, leaveTypeId: f.leaveTypeId || undefined, from: f.from || undefined, to: f.to || undefined })

export const useMyLeaveRequestsQuery = (page: number, filters: LeaveFilters = EMPTY_FILTERS) =>
  useQuery({
    queryKey: [...KEY, 'mine', page, filters.status, filters.leaveTypeId, filters.from, filters.to],
    queryFn: ({ signal }) => listMyLeaveRequests({ page, page_size: PAGE_SIZE, ...toParams(filters) }, signal),
    placeholderData: keepPreviousData,
  })

/** `status` undefined = every status (history view). */
export const useTeamLeaveRequestsQuery = (opts: { status?: LeaveStatus; leaveTypeId?: string; from?: string; to?: string; excludeStatus?: LeaveStatus; page?: number; pageSize?: number; enabled?: boolean } = {}) => {
  const { status, excludeStatus, leaveTypeId, from, to, page = 1, pageSize = PAGE_SIZE, enabled = true } = opts
  return useQuery({
    queryKey: [...KEY, 'team', status ?? 'all', excludeStatus ?? '', leaveTypeId ?? '', from ?? '', to ?? '', page, pageSize],
    queryFn: ({ signal }) => listTeamLeaveRequests({ page, page_size: pageSize, status, excludeStatus, leaveTypeId, from, to }, signal),
    enabled,
    placeholderData: keepPreviousData,
  })
}

function useInvalidate() {
  const qc = useQueryClient()
  return () => Promise.all([qc.invalidateQueries({ queryKey: KEY }), qc.invalidateQueries({ queryKey: EMPLOYEE_DASHBOARD_KEY })])
}

export function useApplyLeaveMutation() {
  const invalidate = useInvalidate()
  return useMutation({ mutationFn: applyLeave, onSuccess: invalidate })
}

export function useCancelLeaveMutation() {
  const invalidate = useInvalidate()
  return useMutation({ mutationFn: (id: string) => cancelLeaveRequest(id), onSuccess: invalidate })
}

export function useDecideLeaveMutation() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: ({ id, decision, comment }: { id: string; decision: 'approve' | 'reject'; comment?: string }) =>
      decision === 'approve' ? approveLeaveRequest(id, comment) : rejectLeaveRequest(id, comment),
    onSuccess: invalidate,
  })
}
