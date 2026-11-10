const DATE_FORMAT: Intl.DateTimeFormatOptions = { weekday: 'short', day: '2-digit', month: 'short', year: 'numeric' }
const TIME_FORMAT: Intl.DateTimeFormatOptions = { hour: '2-digit', minute: '2-digit', second: '2-digit' }

export function formatClockParts(now: Date, locale?: string | string[]): { date: string; time: string } {
  return {
    date: new Intl.DateTimeFormat(locale, DATE_FORMAT).format(now),
    time: new Intl.DateTimeFormat(locale, TIME_FORMAT).format(now),
  }
}
