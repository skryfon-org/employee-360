export const ALLOCATION_STATES = ['past', 'active', 'upcoming'] as const
export type AllocationState = (typeof ALLOCATION_STATES)[number]
export const ALLOCATION_FILTERS = ['all', 'active', 'upcoming', 'past'] as const
export type AllocationFilter = (typeof ALLOCATION_FILTERS)[number]

export const ALLOCATION_STATE_LABELS: Record<AllocationState, string> = {
  past: 'Past',
  active: 'Active',
  upcoming: 'Upcoming',
}

export const ALLOCATION_FILTER_LABELS: Record<AllocationFilter, string> = { all: 'All', ...ALLOCATION_STATE_LABELS }

/** Same palette as the employee calendar. */
export const ALLOCATION_STATE_STYLES: Record<AllocationState, { box: string; dot: string }> = {
  past: { box: 'bg-slate-100 border-slate-300 text-slate-600', dot: 'bg-slate-600' },
  active: { box: 'bg-green-50 border-green-300 text-green-700', dot: 'bg-green-700' },
  upcoming: { box: 'bg-amber-50 border-amber-300 text-amber-700', dot: 'bg-amber-700' },
}

/** Today as YYYY-MM-DD (UTC, matching the backend). */
export const todayISO = () => new Date().toISOString().slice(0, 10)

/** past = end < today, upcoming = start > today, active otherwise (end null = open-ended). */
export function allocationState(
  a: { start_date?: string; end_date?: string | null },
  today: string = todayISO(),
): AllocationState {
  if (a.end_date && a.end_date < today) return 'past'
  if (a.start_date && a.start_date > today) return 'upcoming'
  return 'active'
}

export function parseAllocationFilter(v: string | null): AllocationFilter {
  return (ALLOCATION_FILTERS as readonly string[]).includes(v ?? '') ? (v as AllocationFilter) : 'all'
}
