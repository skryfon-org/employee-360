import { unwrapListResponse, unwrapSingleEntity } from './unwrap.ts';
import {
  getApiV1Projects,
  getApiV1ProjectsCodeSuggestion,
  getApiV1ProjectsId,
  getApiV1ProjectsIdAllocations,
  patchApiV1ProjectsId,
  postApiV1Projects,
  postApiV1ProjectsIdAllocations,
} from './generated/hooks/projects/projects.ts';
import { deleteApiV1AllocationsId, patchApiV1AllocationsId } from './generated/hooks/allocations/allocations.ts';
import { getApiV1EmployeesIdAllocations } from './generated/hooks/employees/employees.ts';
import { getApiV1MeProjects } from './generated/hooks/me/me.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectProjectResponse as Project } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectProjectResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectProjectCodeSuggestionResponse as ProjectCodeSuggestion } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectProjectCodeSuggestionResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectCreateProjectRequest as CreateProjectRequest } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectCreateProjectRequest.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectUpdateProjectRequest as UpdateProjectRequest } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectUpdateProjectRequest.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectProjectAllocationResponse as ProjectAllocation } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectProjectAllocationResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectAllocationWriteResponse as AllocationWriteResult } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectAllocationWriteResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectCreateAllocationRequest as CreateAllocationRequest } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectCreateAllocationRequest.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectUpdateAllocationRequest as UpdateAllocationRequest } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectUpdateAllocationRequest.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectUserAllocationResponse as UserAllocation } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectUserAllocationResponse.ts';
import type { GithubComSkryfonOrgEmployee360BackendInternalTypesProjectUserAllocationsResponse as UserAllocations } from './generated/models/githubComSkryfonOrgEmployee360BackendInternalTypesProjectUserAllocationsResponse.ts';
import type { GetApiV1EmployeesIdAllocationsScope as EmployeeAllocationScope } from './generated/models/getApiV1EmployeesIdAllocationsScope.ts';
import type { GetApiV1ProjectsIdAllocationsState as AllocationStateFilter } from './generated/models/getApiV1ProjectsIdAllocationsState.ts';
import type { GetApiV1MeProjectsParams } from './generated/models/getApiV1MeProjectsParams.ts';
import type { GetApiV1ProjectsParams as ListProjectsParams } from './generated/models/getApiV1ProjectsParams.ts';

export type {
  ProjectCodeSuggestion,
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  ProjectAllocation,
  AllocationWriteResult,
  CreateAllocationRequest,
  UpdateAllocationRequest,
  UserAllocation,
  UserAllocations,
  ListProjectsParams,
  EmployeeAllocationScope,
};

/** `GET /api/v1/projects` (admin/super_admin). */
export async function listProjects(params: ListProjectsParams, signal?: AbortSignal) {
  return unwrapListResponse(await getApiV1Projects(params, signal));
}

export async function getProject(id: string, signal?: AbortSignal): Promise<Project> {
  return unwrapSingleEntity(await getApiV1ProjectsId(id, signal));
}

export async function createProject(body: CreateProjectRequest): Promise<Project> {
  return unwrapSingleEntity(await postApiV1Projects(body));
}

export async function updateProject(id: string, body: UpdateProjectRequest): Promise<Project> {
  return unwrapSingleEntity(await patchApiV1ProjectsId(id, body));
}

export async function listProjectAllocations(projectId: string, state?: AllocationStateFilter, signal?: AbortSignal): Promise<ProjectAllocation[]> {
  return unwrapListResponse(await getApiV1ProjectsIdAllocations(projectId, state ? { state } : undefined, signal)).data;
}

/** Response includes `over_allocated` / `user_total_percent`; over-allocation is advisory and never rejected. */
export async function createAllocation(projectId: string, body: CreateAllocationRequest): Promise<AllocationWriteResult> {
  return unwrapSingleEntity(await postApiV1ProjectsIdAllocations(projectId, body));
}

export async function updateAllocation(id: string, body: UpdateAllocationRequest): Promise<AllocationWriteResult> {
  return unwrapSingleEntity(await patchApiV1AllocationsId(id, body));
}

export async function deleteAllocation(id: string): Promise<void> {
  await deleteApiV1AllocationsId(id);
}

/** `GET /api/v1/employees/:id/allocations` — allocations for `scope` (server default: active). `total_percent` always reflects today's live allocations. */
export async function getEmployeeAllocations(employeeId: string, scope?: EmployeeAllocationScope, signal?: AbortSignal): Promise<UserAllocations> {
  return unwrapSingleEntity(await getApiV1EmployeesIdAllocations(employeeId, scope ? { scope } : undefined, signal));
}

/** `GET /api/v1/me/projects` — the caller's own allocations (default scope: active). */
export async function getMyProjects(scope?: GetApiV1MeProjectsParams['scope'], signal?: AbortSignal): Promise<UserAllocations> {
  return unwrapSingleEntity(await getApiV1MeProjects(scope ? { scope } : undefined, signal));
}

/** `GET /api/v1/projects/code-suggestion?name=` — advisory code the server would generate for a name. */
export async function suggestProjectCode(name: string, signal?: AbortSignal): Promise<ProjectCodeSuggestion> {
  return unwrapSingleEntity(await getApiV1ProjectsCodeSuggestion({ name }, signal));
}
