import { isoOf } from '../../statusLog/dates'
import type { LeaveRequest } from '../queries/leaveQueries'

/** Pending requests, and approved ones starting today or later, can be cancelled (server is authoritative). */
export function canCancel(r: LeaveRequest, today: string = isoOf(new Date())) {
  return r.status === 'pending' || (r.status === 'approved' && r.start_date >= today)
}
