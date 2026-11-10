import type { UserAllocation } from '../queries/projectQueries'

export const PAGE_SIZE = 10

/** Case-insensitive, trimmed match on project name, project code and role on project. */
export function filterAllocations(rows: UserAllocation[], query: string): UserAllocation[] {
  const q = query.trim().toLowerCase()
  if (!q) return rows
  return rows.filter((a) =>
    [a.project_name, a.project_code, a.role_on_project].some((f) => (f ?? '').toLowerCase().includes(q)),
  )
}
