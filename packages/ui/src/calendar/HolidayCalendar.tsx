import { useEffect, useRef, type MouseEvent } from 'react'
import FullCalendar from '@fullcalendar/react'
import dayGridPlugin from '@fullcalendar/daygrid'
import interactionPlugin from '@fullcalendar/interaction'
import type { DayCellContentArg, DayCellMountArg, EventClickArg, EventContentArg } from '@fullcalendar/core'
import type { CalendarCursor, CalendarHoliday, CalendarScope } from './types'
import { MONTH_NAMES, toIsoDate } from './dates'
import { HolidayYearGrid } from './HolidayYearGrid'
import './holidayCalendar.css'

const navBtn =
  'h-11 w-11 rounded-sm border border-slate-300 bg-white text-sm font-medium text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9 md:w-9'

const monthIndex = (c: CalendarCursor) => c.year * 12 + (c.month - 1)

function shift(c: CalendarCursor, delta: number): CalendarCursor {
  const i = monthIndex(c) + delta
  return { year: Math.floor(i / 12), month: (i % 12) + 1 }
}

function renderEvent(arg: EventContentArg) {
  if (arg.event.extendedProps.kind === 'label') {
    return (
      <div className={`holiday-day-label${arg.event.extendedProps.muted ? ' holiday-day-label--muted' : ''}${arg.event.extendedProps.tone ? ` holiday-day-label--${arg.event.extendedProps.tone as string}` : ''}`} title={arg.event.extendedProps.ariaLabel as string | undefined}>
        {arg.event.title}
      </div>
    )
  }
  const color = arg.event.extendedProps.color as string | undefined
  return (
    <div className="holiday-event" title={arg.event.title}>
      <span className="holiday-event__dot" style={color ? { backgroundColor: color } : undefined} aria-hidden="true" />
      <span className="holiday-event__name">{arg.event.title}</span>
    </div>
  )
}

function markToday(arg: DayCellMountArg) {
  if (arg.isToday) arg.el.setAttribute('aria-current', 'date')
  else arg.el.removeAttribute('aria-current')
}

/** A small text marker rendered inside a day cell (no category dot), e.g. "6h" or "None". */
export interface CalendarDayLabel {
  /** `YYYY-MM-DD`. */
  date: string
  text: string
  /** Accessible name / tooltip; defaults to the text. */
  ariaLabel?: string
  /** Render in a muted style. */
  muted?: boolean
  /** Semantic tone: `info` (blue, e.g. holiday/leave) or `warning` (amber, needs attention). Overrides `muted` styling. */
  tone?: 'info' | 'warning'
}

export interface HolidayCalendarProps {
  cursor: CalendarCursor
  holidays: CalendarHoliday[]
  onCursorChange: (cursor: CalendarCursor) => void
  scope: CalendarScope
  onScopeChange: (scope: CalendarScope) => void
  /** Omit for a read-only calendar (events are not interactive). */
  onSelect?: (h: CalendarHoliday) => void
  /** Omit to disable date-click (e.g. create) behaviour. */
  onDateClick?: (isoDate: string) => void
  /** 0 = Sunday ... 6 = Saturday (from calendar settings). */
  weekStartDay?: number
  /** Optional inclusive bounds for month navigation (previous/next/Today are disabled outside them). */
  minMonth?: CalendarCursor
  maxMonth?: CalendarCursor
  /** Accessible name of the calendar group. */
  ariaLabel?: string
  /** False = month view only (hides the Year/Month switch). */
  showScopeToggle?: boolean
  /** `YYYY-MM-DD` of the highlighted day. */
  selectedDate?: string
  /** Disabled days are visually muted and never fire onDateClick. */
  isDateDisabled?: (isoDate: string) => boolean
  /** Extra CSS classes for a day cell. */
  dayCellClassNames?: (isoDate: string) => string[]
  /** Per-day text markers (no category dot). */
  dayLabels?: CalendarDayLabel[]
}

export function HolidayCalendar({
  cursor,
  holidays,
  onCursorChange,
  scope,
  onScopeChange: setScope,
  onSelect,
  onDateClick,
  weekStartDay = 0,
  minMonth,
  maxMonth,
  ariaLabel = 'Holiday calendar',
  showScopeToggle = true,
  selectedDate,
  isDateDisabled,
  dayCellClassNames,
  dayLabels,
}: HolidayCalendarProps) {
  const ref = useRef<FullCalendar>(null)
  const byId = new Map(holidays.map((h) => [h.id, h]))
  const iso = toIsoDate(cursor.year, cursor.month, 1)

  useEffect(() => {
    ref.current?.getApi().gotoDate(iso)
  }, [iso])

  const events = [
    ...holidays.map((h) => ({
      id: h.id,
      title: h.name,
      start: h.date,
      allDay: true,
      extendedProps: { color: h.category?.color },
    })),
    ...(dayLabels ?? []).map((l, i) => ({
      id: `label-${l.date}-${i}`,
      title: l.text,
      start: l.date,
      allDay: true,
      classNames: ['holiday-day-label-event'],
      extendedProps: { kind: 'label', muted: l.muted, tone: l.tone, ariaLabel: l.ariaLabel ?? l.text },
    })),
  ]
  const cellIso = (d: Date) => toIsoDate(d.getFullYear(), d.getMonth() + 1, d.getDate())
  const cellClasses = (arg: DayCellContentArg) => {
    const date = cellIso(arg.date)
    const out = dayCellClassNames?.(date) ?? []
    if (date === selectedDate) out.push('holiday-day--selected')
    if (isDateDisabled?.(date)) out.push('holiday-day--disabled')
    return out
  }
  // Day clicks are resolved from the DOM (not FullCalendar's pointer-geometry dateClick) so they are testable and honour disabled days.
  const handleDayClick = (e: MouseEvent<HTMLDivElement>) => {
    if (!onDateClick || !(e.target instanceof Element)) return
    if (e.target.closest('.fc-event:not(.holiday-day-label-event), .fc-more-link, .fc-popover')) return
    const date = e.target.closest<HTMLElement>('.fc-daygrid-day')?.dataset.date
    if (date && !isDateDisabled?.(date)) onDateClick(date)
  }
  const handleDayMount = (arg: DayCellMountArg) => {
    markToday(arg)
    if (isDateDisabled?.(cellIso(arg.date))) arg.el.setAttribute('aria-disabled', 'true')
    else arg.el.removeAttribute('aria-disabled')
  }

  const nowCursor = { year: new Date().getFullYear(), month: new Date().getMonth() + 1 }
  const prev = shift(cursor, -1)
  const next = shift(cursor, 1)
  const outOfRange = (c: CalendarCursor) => (minMonth ? monthIndex(c) < monthIndex(minMonth) : false) || (maxMonth ? monthIndex(c) > monthIndex(maxMonth) : false)

  const handleEventClick = (arg: EventClickArg) => {
    const holiday = byId.get(arg.event.id)
    if (holiday) onSelect?.(holiday)
  }

  return (
    <div aria-label={ariaLabel} role="group" className={`holiday-calendar${onDateClick ? ' holiday-calendar--clickable' : ''}${onSelect ? ' holiday-calendar--selectable' : ''} flex flex-col gap-3 p-3 sm:p-4`}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        {showScopeToggle && (
        <div role="group" aria-label="Calendar scope" className="inline-flex rounded-sm bg-slate-100 p-1">
          {(['year', 'month'] as CalendarScope[]).map((value) => (
            <button
              key={value}
              type="button"
              aria-pressed={scope === value}
              onClick={() => setScope(value)}
              className={`h-9 rounded-sm px-3 text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-7 ${
                scope === value ? 'bg-white font-semibold text-slate-900' : 'font-medium text-slate-600 hover:text-slate-900'
              }`}
            >
              {value === 'year' ? 'Year' : 'Month'}
            </button>
          ))}
        </div>
        )}
        {scope === 'month' && (
          <div role="group" aria-label="Month" className="flex items-center gap-2">
            <button type="button" className={navBtn} aria-label="Previous month" disabled={outOfRange(prev)} onClick={() => onCursorChange(prev)}>
              &lsaquo;
            </button>
            <h2 aria-live="polite" className="min-w-36 text-center text-base font-semibold text-slate-900">
              {MONTH_NAMES[cursor.month - 1]} {cursor.year}
            </h2>
            <button type="button" className={navBtn} aria-label="Next month" disabled={outOfRange(next)} onClick={() => onCursorChange(next)}>
              &rsaquo;
            </button>
          </div>
        )}
        <button
          type="button"
          disabled={outOfRange(nowCursor)}
          className="h-11 rounded-sm border border-slate-300 bg-white px-3 text-sm font-medium text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
          onClick={() => {
            onCursorChange(nowCursor)
          }}
        >
          Today
        </button>
      </div>
      {showScopeToggle && scope === 'year' && (
        <HolidayYearGrid
          year={cursor.year}
          holidays={holidays}
          weekStartDay={weekStartDay}
          onSelect={onSelect}
          onOpenMonth={(month) => {
            onCursorChange({ year: cursor.year, month })
            setScope('month')
          }}
        />
      )}
      {scope === 'month' && (
        <div onClick={handleDayClick}>
          <FullCalendar
            ref={ref}
            plugins={[dayGridPlugin, interactionPlugin]}
            initialView="dayGridMonth"
            initialDate={iso}
            headerToolbar={false}
            firstDay={weekStartDay}
            height="auto"
            fixedWeekCount={false}
            dayMaxEvents={3}
            events={events}
            eventContent={renderEvent}
            eventClick={handleEventClick}
            dayCellClassNames={cellClasses}
            dayCellDidMount={handleDayMount}
            eventInteractive={Boolean(onSelect)}
          />
        </div>
      )}
    </div>
  )
}
