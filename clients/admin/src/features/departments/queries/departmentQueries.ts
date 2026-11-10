import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  deleteApiV1DepartmentsId,
  getApiV1Departments,
  postApiV1Departments,
  putApiV1DepartmentsId,
  unwrapListResponse,
  unwrapSingleEntity,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesDepartmentCreateDepartmentRequest as CreateDepartmentRequest,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesDepartmentDepartmentResponse as Department,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesDepartmentUpdateDepartmentRequest as UpdateDepartmentRequest,
} from '@employee360/api-client'
import type { DepartmentListParams } from '../schemas/departmentSchemas'

export type { Department }

export const DEPARTMENTS_KEY = ['departments'] as const

export function useDepartmentsQuery(params: DepartmentListParams) {
  return useQuery({
    queryKey: [...DEPARTMENTS_KEY, params],
    queryFn: async ({ signal }) => unwrapListResponse(await getApiV1Departments(params, signal)),
    placeholderData: keepPreviousData,
  })
}

export const ACTIVE_DEPARTMENTS_KEY = [...DEPARTMENTS_KEY, 'active-options'] as const

/** Active departments for selects (single page of up to 100). */
export function useActiveDepartmentsQuery() {
  return useQuery({
    queryKey: ACTIVE_DEPARTMENTS_KEY,
    queryFn: async ({ signal }) =>
      unwrapListResponse(await getApiV1Departments({ is_active: true, page: 1, page_size: 100 }, signal)).data,
  })
}

function useInvalidatingMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: DEPARTMENTS_KEY }),
  })
}

export const useCreateDepartmentMutation = () =>
  useInvalidatingMutation(async (body: CreateDepartmentRequest) => unwrapSingleEntity(await postApiV1Departments(body)))

export const useUpdateDepartmentMutation = () =>
  useInvalidatingMutation(async (v: { id: string; body: UpdateDepartmentRequest }) =>
    unwrapSingleEntity(await putApiV1DepartmentsId(v.id, v.body)),
  )

export const useDeleteDepartmentMutation = () =>
  useInvalidatingMutation(async (id: string) => {
    await deleteApiV1DepartmentsId(id)
  })
