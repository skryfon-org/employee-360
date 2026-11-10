import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createAllocation,
  createProject,
  deleteAllocation,
  getEmployeeAllocations,
  getProject,
  listProjectAllocations,
  listProjects,
  suggestProjectCode,
  updateAllocation,
  updateProject,
  getApiV1Employees,
  GetApiV1EmployeesInclude,
  unwrapListResponse,
  type CreateAllocationRequest,
  type CreateProjectRequest,
  type ProjectAllocation,
  type UpdateAllocationRequest,
  type UpdateProjectRequest,
  type Project,
  type UserAllocations,
  type UserAllocation,
  type EmployeeAllocationScope,
  type GithubComSkryfonOrgEmployee360BackendInternalTypesEmployeeEmployeeSummaryResponse as EmployeeSummary,
} from '@employee360/api-client'
import type { ProjectListParams } from '../schemas/projectSchemas'

export type { EmployeeAllocationScope, Project, ProjectAllocation, UserAllocations, UserAllocation, EmployeeSummary }

export const PROJECTS_KEY = ['projects'] as const
export type AllocationStateParam = 'all' | 'active' | 'upcoming' | 'past'
export const projectAllocationsKey = (projectId: string, state: AllocationStateParam = 'all') =>
  ['project-allocations', projectId, state] as const
export const EMPLOYEE_ALLOCATIONS_KEY = ['employee-allocations'] as const

export function useProjectsQuery(params: ProjectListParams, enabled = true) {
  return useQuery({
    queryKey: [...PROJECTS_KEY, 'list', params],
    queryFn: ({ signal }) =>
      listProjects({ ...params, search: params.search || undefined }, signal),
    placeholderData: keepPreviousData,
    enabled,
  })
}

export function useProjectQuery(id: string | undefined) {
  return useQuery({
    queryKey: [...PROJECTS_KEY, 'detail', id],
    queryFn: ({ signal }) => getProject(id as string, signal),
    enabled: Boolean(id),
  })
}

/** Advisory server-generated code for a project name; callers debounce `name` and gate with `enabled`. */
export function useProjectCodeSuggestionQuery(name: string, enabled: boolean) {
  return useQuery({
    queryKey: [...PROJECTS_KEY, 'code-suggestion', name],
    queryFn: ({ signal }) => suggestProjectCode(name, signal),
    enabled: enabled && name.length > 0,
    staleTime: 60_000,
    retry: false,
  })
}

export function useProjectAllocationsQuery(projectId: string | undefined, state: AllocationStateParam = 'all') {
  return useQuery({
    queryKey: projectAllocationsKey(projectId ?? '', state),
    queryFn: ({ signal }) => listProjectAllocations(projectId as string, state === 'all' ? undefined : state, signal),
    enabled: Boolean(projectId),
    placeholderData: keepPreviousData,
  })
}

/** Scope is part of the key but after the `EMPLOYEE_ALLOCATIONS_KEY` prefix, so prefix invalidation refreshes every scope. */
export function useEmployeeAllocationsQuery(employeeId: string | undefined, scope: EmployeeAllocationScope = 'active') {
  return useQuery({
    queryKey: [...EMPLOYEE_ALLOCATIONS_KEY, employeeId, scope],
    queryFn: ({ signal }) => getEmployeeAllocations(employeeId as string, scope, signal),
    enabled: Boolean(employeeId),
  })
}

/** Employees for the picker. `includeAllocation` adds each person's current `allocated_percent`; only the allocation picker needs it. */
export function useEmployeeOptionsQuery(search = '', activeOnly = false, pageSize = 100, enabled = true, includeAllocation = false) {
  return useQuery({
    queryKey: ['employees', 'project-options', search, activeOnly, pageSize, includeAllocation],
    enabled,
    queryFn: async ({ signal }) =>
      unwrapListResponse(
        await getApiV1Employees(
          { page: 1, page_size: pageSize, search: search || undefined, ...(activeOnly ? { status: 'active' as const } : {}), ...(includeAllocation ? { include: GetApiV1EmployeesInclude.allocation } : {}) },
          signal,
        ),
      ).data,
    placeholderData: keepPreviousData,
  })
}

export function useCreateProjectMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateProjectRequest) => createProject(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: PROJECTS_KEY }),
  })
}

export function useUpdateProjectMutation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: string; body: UpdateProjectRequest }) => updateProject(v.id, v.body),
    // Project status/dates affect which allocations count toward an employee's total.
    onSuccess: () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: PROJECTS_KEY }),
        qc.invalidateQueries({ queryKey: EMPLOYEE_ALLOCATIONS_KEY }),
      ]),
  })
}

function useAllocationInvalidation() {
  const qc = useQueryClient()
  return () =>
    Promise.all([
      qc.invalidateQueries({ queryKey: ['project-allocations'] }),
      qc.invalidateQueries({ queryKey: EMPLOYEE_ALLOCATIONS_KEY }),
    ])
}

export function useCreateAllocationMutation() {
  const invalidate = useAllocationInvalidation()
  return useMutation({
    mutationFn: (v: { projectId: string; body: CreateAllocationRequest }) => createAllocation(v.projectId, v.body),
    onSuccess: invalidate,
  })
}

export function useUpdateAllocationMutation() {
  const invalidate = useAllocationInvalidation()
  return useMutation({
    mutationFn: (v: { id: string; body: UpdateAllocationRequest }) => updateAllocation(v.id, v.body),
    onSuccess: invalidate,
  })
}

export function useDeleteAllocationMutation() {
  const invalidate = useAllocationInvalidation()
  return useMutation({ mutationFn: (id: string) => deleteAllocation(id), onSuccess: invalidate })
}
