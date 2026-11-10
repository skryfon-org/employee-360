import { toIsoDate } from '@employee360/ui'
import type { UserAllocation } from '../queries/projectQueries'

export function todayIso(now = new Date()): string {
  return toIsoDate(now.getFullYear(), now.getMonth() + 1, now.getDate())
}

/** Ended = project finished/archived, or the allocation end date is in the past. */
export function isEnded(a: UserAllocation, today = todayIso()): boolean {
  if (a.project_status === 'completed' || a.project_status === 'archived') return true
  return !!a.end_date && a.end_date < today
}

export type AllocationState = 'past' | 'active' | 'upcoming'

/** Past = ended or project completed/archived; Upcoming = starts after today; otherwise Active. */
export function allocationState(a: UserAllocation, today = todayIso()): AllocationState {
  if (isEnded(a, today)) return 'past'
  if (a.start_date && a.start_date > today) return 'upcoming'
  return 'active'
}
