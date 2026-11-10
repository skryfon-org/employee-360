import type { ProjectOption } from './components/EntryForm'

/** Calendar-date prefix (YYYY-MM-DD) of a server date or timestamp. */
const day = (s?: string) => (s ? s.slice(0, 10) : undefined)

/**
 * Mirrors the server's `ProjectAllocation.ActiveOn` (the basis of `no_active_allocation`):
 * start_date <= date, and end_date is open-ended or >= date.
 */
export function isAllocationActiveOn(a: { startDate?: string; endDate?: string }, date: string): boolean {
  const start = day(a.startDate)
  const end = day(a.endDate)
  if (start && start > date) return false
  return !end || end >= date
}

export const projectsActiveOn = (projects: ProjectOption[], date: string) => projects.filter((p) => isAllocationActiveOn(p, date))
