import type { LeaveStatus } from '@employee360/api-client'

export interface LeaveFilters {
  status: LeaveStatus | ''
  /** Leave type UUID, '' = all types. */
  leaveTypeId: string
  from: string
  to: string
}

export const EMPTY_FILTERS: LeaveFilters = { status: '', leaveTypeId: '', from: '', to: '' }

const STATUSES: readonly string[] = ['pending', 'approved', 'rejected', 'cancelled']
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const ISO = /^\d{4}-\d{2}-\d{2}$/

export function filtersFromParams(p: URLSearchParams): LeaveFilters {
  const status = p.get('status') ?? ''
  const typeId = p.get('leave_type_id') ?? ''
  const from = p.get('from') ?? ''
  const to = p.get('to') ?? ''
  const f = ISO.test(from) ? from : ''
  const t = ISO.test(to) ? to : ''
  // A hand-edited URL with to < from would be a 400; swap so the range stays valid.
  const [a, b] = f && t && t < f ? [t, f] : [f, t]
  return { status: STATUSES.includes(status) ? (status as LeaveStatus) : '', leaveTypeId: UUID.test(typeId) ? typeId : '', from: a, to: b }
}

export const hasFilters = (f: LeaveFilters) => !!(f.status || f.leaveTypeId || f.from || f.to)
export const filterKey = (f: LeaveFilters) => `${f.status}|${f.leaveTypeId}|${f.from}|${f.to}`
