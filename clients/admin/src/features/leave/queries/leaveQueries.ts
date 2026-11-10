import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  approveLeaveRequest,
  createLeaveType,
  getApiV1EmployeesId,
  listAllLeaveRequests,
  listAllLeaveTypes,
  rejectLeaveRequest,
  unwrapSingleEntity,
  updateLeaveType,
  type LeaveStatus,
  type LeaveTypeInput,
  type LeaveTypePatch,
} from '@employee360/api-client'

export type { LeaveRequest, LeaveType, LeaveStatus } from '@employee360/api-client'

const KEY = ['leave'] as const
export const PAGE_SIZE = 20

/** Every leave type, active and inactive (admin). */
export const useLeaveTypesQuery = () => useQuery({ queryKey: [...KEY, 'types'], queryFn: ({ signal }) => listAllLeaveTypes(signal) })

/** One employee, to label an `employee_id` filter that arrived via URL. */
export const useLeaveEmployeeQuery = (id: string) =>
  useQuery({
    queryKey: [...KEY, 'employee', id],
    enabled: !!id,
    retry: false,
    queryFn: async ({ signal }) => unwrapSingleEntity(await getApiV1EmployeesId(id, signal)),
  })

export interface LeaveRequestsQueryOpts {
  status?: LeaveStatus
  employeeId?: string
  leaveTypeId?: string
  from?: string
  to?: string
  page?: number
}

export const useAllLeaveRequestsQuery = (opts: LeaveRequestsQueryOpts, enabled = true) => {
  const { status, employeeId, leaveTypeId, from, to, page = 1 } = opts
  return useQuery({
    queryKey: [...KEY, 'all', status ?? '', employeeId ?? '', leaveTypeId ?? '', from ?? '', to ?? '', page],
    queryFn: ({ signal }) => listAllLeaveRequests({ page, page_size: PAGE_SIZE, status, employeeId, leaveTypeId, from, to }, signal),
    placeholderData: keepPreviousData,
    enabled,
  })
}

function useInvalidate() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: KEY })
}

export function useDecideLeaveMutation() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: ({ id, decision, comment }: { id: string; decision: 'approve' | 'reject'; comment?: string }) =>
      decision === 'approve' ? approveLeaveRequest(id, comment) : rejectLeaveRequest(id, comment),
    onSuccess: invalidate,
  })
}

export function useCreateLeaveTypeMutation() {
  const invalidate = useInvalidate()
  return useMutation({ mutationFn: (body: LeaveTypeInput) => createLeaveType(body), onSuccess: invalidate })
}

export function useUpdateLeaveTypeMutation() {
  const invalidate = useInvalidate()
  return useMutation({ mutationFn: ({ id, body }: { id: string; body: LeaveTypePatch }) => updateLeaveType(id, body), onSuccess: invalidate })
}
