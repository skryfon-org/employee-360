import { z } from 'zod'
import { isoOf } from '../../statusLog/dates'

export const LEAVE_DAY_PARTS = ['full', 'first_half', 'second_half'] as const
/**
 * Earliest selectable start date: today in the user's local time. There is no upper bound.
 * The server compares against UTC today with a one-day grace, so users behind UTC are not rejected
 * for picking their local today; any remaining server error is shown under the start date.
 */
export const minStartDate = (now: Date = new Date()) => isoOf(now)

const ISO = /^\d{4}-\d{2}-\d{2}$/

const NO_WORKING_DAY = 'Selected dates fall only on weekly off days or public holidays. Choose at least one working day.'

/**
 * Working days in the inclusive range, excluding weekly off days (0=Sunday..6=Saturday) and public holiday
 * dates (`YYYY-MM-DD`, compared as plain strings); half day = 0.5.
 * Returns null when the dates are incomplete/invalid.
 */
export function countWorkingDays(start: string, end: string, dayPart: string, offDays: readonly number[], holidays: readonly string[] = []): number | null {
  if (!ISO.test(start) || !ISO.test(end) || end < start) return null
  const off = new Set(offDays)
  const hol = new Set(holidays)
  const cur = new Date(`${start}T00:00:00Z`)
  const last = new Date(`${end}T00:00:00Z`).getTime()
  let n = 0
  for (let i = 0; cur.getTime() <= last && i < 3660; i++) {
    if (!off.has(cur.getUTCDay()) && !hol.has(cur.toISOString().slice(0, 10))) n++
    cur.setUTCDate(cur.getUTCDate() + 1)
  }
  return dayPart !== 'full' && n > 0 ? 0.5 : n
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

/** `2026-11-12` -> `Nov 12` (string-based, no timezone shifts). */
export const formatShortDate = (iso: string) => `${MONTHS[Number(iso.slice(5, 7)) - 1] ?? ''} ${Number(iso.slice(8, 10))}`

/** The server stays authoritative for the date rules; these checks give instant feedback. */
export function buildApplySchema(now: Date = new Date(), offDays?: readonly number[], holidays?: readonly string[]) {
  const min = minStartDate(now)
  return z
    .object({
      leave_type_id: z.string().min(1, 'Choose a leave type'),
      start_date: z.string().regex(ISO, 'Choose a start date'),
      end_date: z.string().regex(ISO, 'Choose an end date'),
      day_part: z.enum(LEAVE_DAY_PARTS),
      reason: z.string().trim().max(1000, 'Reason must be at most 1000 characters'),
      approver_id: z.string().min(1, 'Choose an approver'),
    })
    .superRefine((v, ctx) => {
      if (ISO.test(v.start_date) && v.start_date < min) {
        ctx.addIssue({ code: 'custom', path: ['start_date'], message: 'Start date cannot be in the past' })
      } else if (ISO.test(v.end_date) && v.end_date < min) {
        ctx.addIssue({ code: 'custom', path: ['end_date'], message: 'End date cannot be in the past' })
      }
      if (ISO.test(v.start_date) && ISO.test(v.end_date) && v.end_date < v.start_date) {
        ctx.addIssue({ code: 'custom', path: ['end_date'], message: 'End date cannot be before the start date' })
      }
      if (v.day_part !== 'full' && v.start_date !== v.end_date) {
        ctx.addIssue({ code: 'custom', path: ['day_part'], message: 'Half day is only allowed for a single day' })
      }
      // Skipped while off days are unknown (not loaded / failed); holidays unknown = none excluded: the server is authoritative.
      if (offDays && countWorkingDays(v.start_date, v.end_date, v.day_part, offDays, holidays) === 0) {
        ctx.addIssue({ code: 'custom', path: ['end_date'], message: NO_WORKING_DAY })
      }
    })
}

export type ApplyFormValues = z.infer<ReturnType<typeof buildApplySchema>>
