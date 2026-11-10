import { toIsoDate } from '@employee360/ui'

/** Local calendar date as YYYY-MM-DD. */
export const isoOf = (d: Date) => toIsoDate(d.getFullYear(), d.getMonth() + 1, d.getDate())

export const todayIso = (now: Date = new Date()) => isoOf(now)

/** Adds (or subtracts) whole days to an ISO date. */
export function addDays(iso: string, n: number): string {
  const [y, m, d] = iso.split('-').map(Number)
  return isoOf(new Date(y, m - 1, d + n))
}

/** Long form, e.g. "Tuesday, 27 October 2026". */
export function longDate(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

/** Hours with at most 2 decimals, e.g. 4.5 -> "4.5h". */
export const formatHours = (h: number | undefined) => `${Number((h ?? 0).toFixed(2))}h`

/** Current month as YYYY-MM. */
export const monthOf = (d: Date = new Date()) => isoOf(d).slice(0, 7)

/** Shifts a YYYY-MM month by n months. */
export function addMonths(month: string, n: number): string {
  const [y, m] = month.split('-').map(Number)
  return monthOf(new Date(y, m - 1 + n, 1))
}

/** First and last ISO dates of a YYYY-MM month. */
export function monthRange(month: string): { from: string; to: string } {
  const [y, m] = month.split('-').map(Number)
  return { from: isoOf(new Date(y, m - 1, 1)), to: isoOf(new Date(y, m, 0)) }
}

/** e.g. "October 2026". */
export function monthLabel(month: string): string {
  const [y, m] = month.split('-').map(Number)
  return new Date(y, m - 1, 1).toLocaleDateString('en-GB', { month: 'long', year: 'numeric' })
}
