import { useGetApiV1MeProjects, type GetApiV1MeProjectsScope, type UserAllocation, type UserAllocations } from '@employee360/api-client'

export type { UserAllocation, UserAllocations }
export type ProjectScope = GetApiV1MeProjectsScope

/** The caller's project allocations for a scope (server state, one cache entry per scope). */
export function useMyProjectsQuery(scope: ProjectScope) {
  return useGetApiV1MeProjects({ scope })
}
