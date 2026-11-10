import type { CalendarHoliday } from './types'
import { daysInMonth, firstWeekday, formatHolidayDate, MONTH_NAMES, toIsoDate, WEEKDAYS } from './dates'

function CategoryDot({ color }: { color?: string | null }) {
  return <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-slate-600" style={color ? { backgroundColor: color } : undefined} />
}

function todayIso(): string {
  const now = new Date()
  return toIsoDate(now.getFullYear(), now.getMonth() + 1, now.getDate())
}

function MonthGrid({
  year,
  month,
  holidays,
  onSelect,
  onOpenMonth,
  weekStartDay,
  today,
}: {
  year: number
  month: number
  holidays: Map<string, CalendarHoliday[]>
  onSelect?: (h: CalendarHoliday) => void
  today: string
  onOpenMonth: (month: number) => void
  weekStartDay: number
}) {
  const lead = (firstWeekday(year, month) - weekStartDay + 7) % 7
  const total = daysInMonth(year, month)
  const cells: Array<number | null> = [...Array<null>(lead).fill(null), ...Array.from({ length: total }, (_, i) => i + 1)]
  return (
    <section aria-label={`${MONTH_NAMES[month - 1]} ${year}`} className="rounded-sm border border-slate-200 bg-white p-3">
      <h3 className="mb-2 text-sm font-semibold">
        <button
          type="button"
          onClick={() => onOpenMonth(month)}
          className="text-slate-900 hover:text-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
        >
          {MONTH_NAMES[month - 1]}
        </button>
      </h3>
      <div className="grid grid-cols-7 gap-px text-center text-xs text-slate-600">
        {[...WEEKDAYS.slice(weekStartDay), ...WEEKDAYS.slice(0, weekStartDay)].map((d) => (
          <span key={d} aria-hidden="true">{d.slice(0, 2)}</span>
        ))}
        {cells.map((day, i) => {
          if (day === null) return <span key={`b${i}`} />
          const iso = toIsoDate(year, month, day)
          const items = holidays.get(iso)
          const isToday = iso === today
          const todayProps = isToday ? { 'aria-current': 'date' as const } : {}
          const todayRing = isToday ? ' ring-2 ring-slate-900 font-bold' : ''
          if (!items) {
            return (
              <span key={day} {...todayProps} className={`flex h-8 items-center justify-center rounded-sm text-slate-900${isToday ? ' bg-slate-200' + todayRing : ''}`}>
                {day}
              </span>
            )
          }
          const label = items.map((h) => `${h.name}, ${formatHolidayDate(h.date)}`).join('; ')
          const dots = (
            <>
              <span>{day}</span>
              <span className="flex flex-wrap justify-end gap-0.5">
                {items.map((h) => (
                  <CategoryDot key={h.id} color={h.category?.color} />
                ))}
              </span>
            </>
          )
          const cls = `flex h-8 flex-col items-center justify-center rounded-sm border border-slate-300 bg-slate-100 text-slate-900${todayRing}`
          if (!onSelect) {
            return (
              <span key={day} {...todayProps} title={items.map((h) => h.name).join(', ')} className={cls}>
                {dots}
                <span className="sr-only">{label}</span>
              </span>
            )
          }
          if (items.length > 1) {
            // Several holidays on one day: one button per holiday so every one is reachable.
            return (
              <div key={day} {...todayProps} className={cls}>
                <span>{day}</span>
                <span className="flex flex-wrap justify-end gap-0.5">
                  {items.map((h) => (
                    <button
                      key={h.id}
                      type="button"
                      onClick={() => onSelect(h)}
                      aria-label={`${h.name}, ${formatHolidayDate(h.date)}`}
                      title={h.name}
                      className="flex h-8 w-8 items-center justify-center rounded-sm hover:bg-slate-300 md:h-4 md:w-4 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900"
                    >
                      <CategoryDot color={h.category?.color} />
                    </button>
                  ))}
                </span>
              </div>
            )
          }
          return (
            <button
              key={day}
              type="button"
              {...todayProps}
              onClick={() => onSelect(items[0])}
              aria-label={label}
              title={items.map((h) => h.name).join(', ')}
              className={`${cls} hover:bg-slate-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900`}
            >
              {dots}
            </button>
          )
        })}
      </div>
    </section>
  )
}

export function HolidayYearGrid({
  year,
  holidays,
  onSelect,
  onOpenMonth,
  weekStartDay = 0,
}: {
  year: number
  holidays: CalendarHoliday[]
  /** Omit for a read-only grid. */
  onSelect?: (h: CalendarHoliday) => void
  onOpenMonth: (month: number) => void
  /** 0 = Sunday ... 6 = Saturday (from calendar settings). */
  weekStartDay?: number
}) {
  const byDate = new Map<string, CalendarHoliday[]>()
  for (const h of holidays) byDate.set(h.date, [...(byDate.get(h.date) ?? []), h])
  const today = todayIso()
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 12 }, (_, i) => (
        <MonthGrid key={i} year={year} month={i + 1} holidays={byDate} onSelect={onSelect} onOpenMonth={onOpenMonth} weekStartDay={weekStartDay} today={today} />
      ))}
    </div>
  )
}
