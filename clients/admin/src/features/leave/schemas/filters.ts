import type { LeaveStatus } from '@employee360/api-client'

/** '' = all statuses. The queue opens on `pending`. */
export interface LeaveFilters {
  status: LeaveStatus | ''
  employeeId: string
  leaveTypeId: string
  from: string
  to: string
  page: number
}

export const DEFAULT_FILTERS: LeaveFilters = { status: 'pending', employeeId: '', leaveTypeId: '', from: '', to: '', page: 1 }

const STATUSES: readonly string[] = ['pending', 'approved', 'rejected', 'cancelled']
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const ISO = /^\d{4}-\d{2}-\d{2}$/

/** URL is the source of truth: absent `status` = pending, `status=all` = every status. */
export function filtersFromParams(p: URLSearchParams): LeaveFilters {
  const status = p.get('status')
  const emp = p.get('employee_id') ?? ''
  const typeId = p.get('leave_type_id') ?? ''
  const from = ISO.test(p.get('from') ?? '') ? (p.get('from') as string) : ''
  const to = ISO.test(p.get('to') ?? '') ? (p.get('to') as string) : ''
  // A hand-edited URL with to < from would be a 400; swap so the range stays valid.
  const [a, b] = from && to && to < from ? [to, from] : [from, to]
  const page = Number.parseInt(p.get('page') ?? '1', 10)
  return {
    status: status === 'all' ? '' : status && STATUSES.includes(status) ? (status as LeaveStatus) : 'pending',
    employeeId: UUID.test(emp) ? emp : '',
    leaveTypeId: UUID.test(typeId) ? typeId : '',
    from: a,
    to: b,
    page: Number.isFinite(page) && page > 0 ? page : 1,
  }
}

export function filtersToParams(f: LeaveFilters): URLSearchParams {
  const out = new URLSearchParams()
  if (f.status === '') out.set('status', 'all')
  else if (f.status !== 'pending') out.set('status', f.status)
  if (f.employeeId) out.set('employee_id', f.employeeId)
  if (f.leaveTypeId) out.set('leave_type_id', f.leaveTypeId)
  if (f.from) out.set('from', f.from)
  if (f.to) out.set('to', f.to)
  if (f.page > 1) out.set('page', String(f.page))
  return out
}

export const isDefaultFilters = (f: LeaveFilters, defaults: LeaveFilters = DEFAULT_FILTERS) =>
  f.status === defaults.status && !f.employeeId && !f.leaveTypeId && !f.from && !f.to
