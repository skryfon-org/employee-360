/** Minimal structural holiday shape the shared calendar needs; app-level Holiday types satisfy it. */
export interface CalendarHoliday {
  id: string
  name: string
  /** `YYYY-MM-DD`. */
  date: string
  category?: { color?: string | null } | null
}

export type CalendarScope = 'year' | 'month'

export interface CalendarCursor {
  year: number
  /** 1-based month. */
  month: number
}
