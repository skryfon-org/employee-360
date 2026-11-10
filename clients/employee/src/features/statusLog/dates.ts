import { toIsoDate } from '@employee360/ui'

export const WINDOW_DAYS = 7

/** Local calendar date as YYYY-MM-DD. */
export const isoOf = (d: Date) => toIsoDate(d.getFullYear(), d.getMonth() + 1, d.getDate())

/** Today first, then back to today-6 (the loggable window). */
export function lastSevenDays(now: Date = new Date()): string[] {
  return Array.from({ length: WINDOW_DAYS }, (_, i) => {
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i)
    return isoOf(d)
  })
}

/** Short weekday + day label, e.g. "Tue 27". */
export function dayLabel(iso: string): { weekday: string; day: string } {
  const [y, m, d] = iso.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  return { weekday: date.toLocaleDateString('en-US', { weekday: 'short' }), day: String(d) }
}

/** Long form, e.g. "Tuesday, 27 October 2026". */
export function longDate(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

/** True when the date is inside the editable window (today back to today-6). */
export const isEditableDate = (iso: string, now: Date = new Date()) => lastSevenDays(now).includes(iso)

/** Adds (or subtracts) whole days to an ISO date. */
export function addDays(iso: string, n: number): string {
  const [y, m, d] = iso.split('-').map(Number)
  return isoOf(new Date(y, m - 1, d + n))
}

/** First and last day of a 1-based month as ISO dates. The server caps ranges at 366 days; a month is far below. */
export function monthRange(year: number, month: number): { from: string; to: string } {
  return { from: toIsoDate(year, month, 1), to: isoOf(new Date(year, month, 0)) }
}

/** "27 October" for an ISO date. */
export function dayMonth(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('en-GB', { day: 'numeric', month: 'long' })
}
