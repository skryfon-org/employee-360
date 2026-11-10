import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  deleteApiV1EmployeesIdEmergencyContactsContactId,
  getApiV1Departments,
  getApiV1Employees,
  getApiV1EmployeesId,
  getApiV1EmployeesIdAssignments,
  getApiV1Positions,
  patchApiV1EmployeesId,
  postApiV1EmployeesIdActivate,
  postApiV1EmployeesIdDeactivate,
  postApiV1EmployeesIdEmergencyContacts,
  putApiV1EmployeesIdEmergencyContactsContactId,
  putApiV1EmployeesIdRole,
  unwrapListResponse,
  unwrapSingleEntity,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeAssignmentResponse as Assignment,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeEmergencyContactInput as ContactInput,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeEmergencyContactResponse as EmergencyContact,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeEmployeeDetailResponse as EmployeeDetail,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeEmployeeSummaryResponse as EmployeeSummary,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeUpdateEmployeeInput as UpdateEmployeeInput,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeChangeRoleInputRole as AssignableRole,
} from '@employee360/api-client'
import type { EmployeeListParams } from '../schemas/employeeSchemas'

export type { Assignment, ContactInput, EmergencyContact, EmployeeDetail, EmployeeSummary, UpdateEmployeeInput, AssignableRole }

export const EMPLOYEES_KEY = ['employees'] as const

export function useEmployeesQuery(params: EmployeeListParams) {
  return useQuery({
    queryKey: [...EMPLOYEES_KEY, 'list', params],
    queryFn: async ({ signal }) =>
      unwrapListResponse(await getApiV1Employees({ ...params, search: params.search || undefined }, signal)),
    placeholderData: keepPreviousData,
  })
}

export function useEmployeeQuery(id: string | undefined) {
  return useQuery({
    queryKey: [...EMPLOYEES_KEY, 'detail', id],
    queryFn: async ({ signal }) => unwrapSingleEntity(await getApiV1EmployeesId(id as string, signal)),
    enabled: Boolean(id),
  })
}

export function useAssignmentsQuery(id: string | undefined) {
  return useQuery({
    queryKey: [...EMPLOYEES_KEY, 'assignments', id],
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1EmployeesIdAssignments(id as string, signal)).data,
    enabled: Boolean(id),
  })
}

/** All departments (active and inactive) so names resolve for history; selects filter on `is_active`. */
export function useDepartmentOptionsQuery() {
  return useQuery({
    queryKey: ['departments', 'employee-options'],
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1Departments({ page: 1, page_size: 100 }, signal)).data,
  })
}

export function usePositionOptionsQuery() {
  return useQuery({
    queryKey: ['positions', 'employee-options'],
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1Positions({ page: 1, page_size: 100 }, signal)).data,
  })
}

/** Active employees, used as manager choices. */
export function useManagerOptionsQuery() {
  return useQuery({
    queryKey: [...EMPLOYEES_KEY, 'manager-options'],
    queryFn: async ({ signal }) =>
      unwrapListResponse(await getApiV1Employees({ status: 'active', page: 1, page_size: 100 }, signal)).data,
  })
}

function useInvalidatingMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: EMPLOYEES_KEY }),
  })
}

export const useUpdateEmployeeMutation = () =>
  useInvalidatingMutation(async (v: { id: string; body: UpdateEmployeeInput }) =>
    unwrapSingleEntity(await patchApiV1EmployeesId(v.id, v.body)),
  )

export const useActivateEmployeeMutation = () =>
  useInvalidatingMutation(async (id: string) => unwrapSingleEntity(await postApiV1EmployeesIdActivate(id)))

export const useDeactivateEmployeeMutation = () =>
  useInvalidatingMutation(async (id: string) => unwrapSingleEntity(await postApiV1EmployeesIdDeactivate(id)))

export const useChangeRoleMutation = () =>
  useInvalidatingMutation(async (v: { id: string; role: AssignableRole }) =>
    unwrapSingleEntity(await putApiV1EmployeesIdRole(v.id, { role: v.role })),
  )

export const useCreateContactMutation = () =>
  useInvalidatingMutation(async (v: { id: string; body: ContactInput }) =>
    unwrapSingleEntity(await postApiV1EmployeesIdEmergencyContacts(v.id, v.body)),
  )

export const useUpdateContactMutation = () =>
  useInvalidatingMutation(async (v: { id: string; contactId: string; body: ContactInput }) =>
    unwrapSingleEntity(await putApiV1EmployeesIdEmergencyContactsContactId(v.id, v.contactId, v.body)),
  )

export const useDeleteContactMutation = () =>
  useInvalidatingMutation(async (v: { id: string; contactId: string }) => {
    await deleteApiV1EmployeesIdEmergencyContactsContactId(v.id, v.contactId)
  })
