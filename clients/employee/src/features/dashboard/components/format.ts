import { MONTH_NAMES, parseIsoDate } from '@employee360/ui'

const WEEKDAYS_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

/** "5 Nov" from an ISO date. */
export function shortDate(iso: string): string {
  const { month, day } = parseIsoDate(iso)
  return `${day} ${MONTH_NAMES[month - 1].slice(0, 3)}`
}

export function weekdayShort(iso: string): string {
  const { year, month, day } = parseIsoDate(iso)
  return WEEKDAYS_SHORT[new Date(Date.UTC(year, month - 1, day)).getUTCDay()]
}

export function dateRange(start?: string, end?: string): string {
  if (!start) return ''
  if (!end || end === start) return shortDate(start)
  return `${shortDate(start)} – ${shortDate(end)}`
}

export function greetingFor(hour: number): string {
  if (hour < 12) return 'Good morning'
  if (hour < 17) return 'Good afternoon'
  return 'Good evening'
}

export function plural(n: number, one: string, many = `${one}s`): string {
  return `${n} ${n === 1 ? one : many}`
}
