import { dayLabel } from '../dates'
import { describeDayStatus, type DayStatusItem } from '../dayStatus'

export interface WeekDay {
  date: string
  total: number
  /** Why the day may have no log; absent while loading or if unavailable. */
  status?: DayStatusItem
}

/** The 7 loggable days (oldest to newest) with totals; empty days are flagged. Scrolls horizontally on narrow screens. */
export function WeekStrip({
  days,
  selected,
  today,
  onSelect,
}: {
  days: WeekDay[]
  selected: string
  today: string
  onSelect: (d: string) => void
}) {
  const weekTotal = Math.round(days.reduce((sum, d) => sum + d.total, 0) * 100) / 100
  return (
    <div>
      <p className="mb-2 text-sm text-slate-900">
        Week: <strong>{weekTotal}h</strong>
      </p>
    <div className="-mx-1 overflow-x-auto px-1 pb-1">
      <div role="group" aria-label="Last 7 days" title="Use the left and right arrow keys to change day" aria-keyshortcuts="ArrowLeft ArrowRight" className="flex min-w-max gap-2 sm:min-w-0">
        {days.map(({ date, total, status }) => {
          const { weekday, day } = dayLabel(date)
          const active = date === selected
          const missed = total === 0
          const isToday = date === today
          const info = missed ? describeDayStatus(status) : undefined
          const halfDayMissing = info?.tone === 'warning'
          const stateText = missed ? (info ? (halfDayMissing ? 'half-day leave, nothing logged' : info.title.replace(': ', ', ')) : 'nothing logged') : `${total} hours logged`
          return (
            <button
              key={date}
              type="button"
              aria-pressed={active}
              aria-label={`${weekday} ${day}, ${stateText}${isToday ? ', today' : ''}`}
              onClick={() => onSelect(date)}
              className={`flex min-h-24 w-20 flex-col items-center justify-center gap-1 rounded-sm border-2 px-2 py-3 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 sm:w-auto sm:min-w-0 sm:flex-1 ${
                active ? 'border-slate-900 bg-slate-200' : 'border-slate-200 bg-white hover:bg-slate-100'
              }`}
            >
              <span className="font-medium uppercase text-slate-600">{weekday}</span>
              <span className="text-xl font-semibold text-slate-900">{day}</span>
              {missed ? (
                info && !halfDayMissing ? (
                  <span title={info.title} className="inline-flex max-w-full items-center gap-1 whitespace-nowrap rounded-sm border border-blue-300 bg-blue-50 px-1 font-medium text-blue-700">
                    {info.label}
                  </span>
                ) : (
                  <span title={info?.title} className="inline-flex items-center gap-1 font-medium text-amber-700">
                    <span className="h-1.5 w-1.5 rounded-full bg-amber-700" aria-hidden="true" />
                    {halfDayMissing ? 'Missing' : 'None'}
                  </span>
                )
              ) : (
                <span className="font-medium text-slate-900">{total}h</span>
              )}
              {halfDayMissing && <span className="text-[10px] font-medium text-slate-700">Half-day leave</span>}
              <span className={`h-4 text-xs font-semibold text-slate-900 ${isToday ? '' : 'invisible'}`} aria-hidden="true">
                Today
              </span>
            </button>
          )
        })}
      </div>
    </div>
    </div>
  )
}
