import { useCallback, useMemo, useState } from 'react'
import { MONTH_NAMES, WEEKDAYS, daysInMonth, firstWeekday, toIsoDate } from '@employee360/ui'
import { useMediaQuery } from '../../../lib/useMediaQuery'
import type { UserAllocation } from '../queries/projectQueries'
import { AllocationPopover } from './AllocationPopover'
import { allocationState, todayIso, type AllocationState } from './projectDates'

interface Segment {
  a: UserAllocation
  startCol: number // 1..7
  endCol: number // 1..7 inclusive
  lane: number
  continuesLeft: boolean
  continuesRight: boolean
}

/** Splits the visible month into week rows and lays out each allocation as a bar per week. */
export function layoutWeeks(year: number, month: number, allocations: UserAllocation[]): Segment[][] {
  const total = daysInMonth(year, month)
  const offset = firstWeekday(year, month)
  const weekCount = Math.ceil((offset + total) / 7)
  const monthStart = toIsoDate(year, month, 1)
  const monthEnd = toIsoDate(year, month, total)
  const visible = allocations
    .filter((a) => a.start_date && a.start_date <= monthEnd && (!a.end_date || a.end_date >= monthStart))
    .sort((x, y) => (x.start_date ?? '').localeCompare(y.start_date ?? ''))

  const weeks: Segment[][] = []
  for (let w = 0; w < weekCount; w++) {
    const firstDay = w * 7 - offset + 1 // may be < 1
    const lastDay = firstDay + 6
    const laneEnds: number[] = []
    const segs: Segment[] = []
    for (const a of visible) {
      const s = a.start_date as string
      const startDay = s < monthStart ? 1 : Number(s.slice(8, 10))
      const endDay = !a.end_date || a.end_date > monthEnd ? total : Number(a.end_date.slice(8, 10))
      if (endDay < Math.max(firstDay, 1) || startDay > Math.min(lastDay, total)) continue
      const from = Math.max(startDay, firstDay, 1)
      const to = Math.min(endDay, lastDay, total)
      const startCol = from - firstDay + 1
      const endCol = to - firstDay + 1
      let lane = laneEnds.findIndex((e) => e < startCol)
      if (lane === -1) lane = laneEnds.length
      laneEnds[lane] = endCol
      segs.push({ a, startCol, endCol, lane, continuesLeft: startDay < from, continuesRight: endDay > to })
    }
    weeks.push(segs)
  }
  return weeks
}

export const STATE_META: Record<AllocationState, { label: string; bar: string; dot: string }> = {
  past: { label: 'Past', bar: 'border-slate-300 bg-slate-100 text-slate-600', dot: 'bg-slate-600' },
  active: { label: 'Active', bar: 'border-green-300 bg-green-50 text-green-700', dot: 'bg-green-700' },
  upcoming: { label: 'Upcoming', bar: 'border-amber-300 bg-amber-50 text-amber-700', dot: 'bg-amber-700' },
}
const SHORT_MONTH = (m: number) => MONTH_NAMES[m - 1].slice(0, 3)

function Agenda({ year, month, weeks, today }: { year: number; month: number; weeks: Segment[][]; today: string }) {
  const offset = firstWeekday(year, month)
  const total = daysInMonth(year, month)
  const groups = weeks
    .map((segs, w) => ({
      segs,
      from: Math.max(1, w * 7 - offset + 1),
      to: Math.min(total, w * 7 - offset + 7),
    }))
    .filter((g) => g.segs.length > 0)
  if (groups.length === 0) return <p className="py-4 text-sm text-slate-600">No allocations this month.</p>
  return (
    <div data-testid="calendar-agenda" className="flex flex-col gap-4">
      {groups.map((g) => (
        <section key={g.from} aria-label={`${SHORT_MONTH(month)} ${g.from} to ${g.to}`}>
          <h3 className="mb-2 text-xs font-semibold text-slate-600">
            {SHORT_MONTH(month)} {g.from} - {g.to}
          </h3>
          <ul className="flex flex-col gap-2">
            {g.segs.map((s) => {
              const state = allocationState(s.a, today)
              const meta = STATE_META[state]
              return (
                <li
                  key={s.a.id}
                  data-testid="agenda-item"
                  data-state={state}
                  className={`rounded-sm border px-3 py-2 text-sm ${meta.bar}`}
                >
                  <p className="flex items-center gap-1.5 font-medium">
                    <span aria-hidden="true" className={`h-1.5 w-1.5 shrink-0 rounded-full ${meta.dot}`} />
                    <span className="truncate">{s.a.project_name}</span>
                    <span className="ml-auto shrink-0">{s.a.allocation_percent}%</span>
                  </p>
                  <p className="text-xs">
                    {s.a.start_date} to {s.a.end_date ?? 'Ongoing'} - {meta.label}
                  </p>
                </li>
              )
            })}
          </ul>
        </section>
      ))}
    </div>
  )
}

const STATE_ORDER: AllocationState[] = ['past', 'active', 'upcoming']

const navBtn =
  'h-9 rounded-sm border border-slate-300 bg-white px-3 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

export function ProjectsCalendar({ allocations }: { allocations: UserAllocation[] }) {
  const now = new Date()
  const [cursor, setCursor] = useState({ year: now.getFullYear(), month: now.getMonth() + 1 })
  const { year, month } = cursor
  const today = todayIso()
  const weeks = useMemo(() => layoutWeeks(year, month, allocations), [year, month, allocations])
  const offset = firstWeekday(year, month)
  const total = daysInMonth(year, month)
  const narrow = useMediaQuery('(max-width: 639px)')
  const [selected, setSelected] = useState<{ a: UserAllocation; anchor: DOMRect; trigger: HTMLElement } | null>(null)
  const closePopover = useCallback(() => setSelected(null), [])

  const shift = (d: number) =>
    setCursor(({ year: y, month: m }) => {
      setSelected(null)
      const idx = y * 12 + (m - 1) + d
      return { year: Math.floor(idx / 12), month: (idx % 12) + 1 }
    })

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-base font-semibold text-slate-900" aria-live="polite">
          {MONTH_NAMES[month - 1]} {year}
        </h2>
        <div className="flex gap-2">
          <button type="button" className={navBtn} onClick={() => shift(-1)} aria-label="Previous month">Prev</button>
          <button type="button" className={navBtn} onClick={() => setCursor({ year: now.getFullYear(), month: now.getMonth() + 1 })}>Today</button>
          <button type="button" className={navBtn} onClick={() => shift(1)} aria-label="Next month">Next</button>
        </div>
      </div>
      {narrow ? (
        <Agenda year={year} month={month} weeks={weeks} today={today} />
      ) : (
      <div className="overflow-x-auto">
        <div className="min-w-[40rem] border border-slate-300" role="grid" aria-label={`${MONTH_NAMES[month - 1]} ${year} allocations`}>
          <div className="grid grid-cols-7 bg-slate-100 text-xs font-medium text-slate-600" role="row">
            {WEEKDAYS.map((d) => (
              <div key={d} role="columnheader" className="px-2 py-2">{d}</div>
            ))}
          </div>
          {weeks.map((segs, w) => {
            const lanes = segs.reduce((m, s) => Math.max(m, s.lane + 1), 0)
            return (
              <div key={w} role="row" className="relative border-t border-slate-300">
                <div className="absolute inset-0 grid grid-cols-7" aria-hidden="true">
                  {Array.from({ length: 7 }, (_, i) => {
                    const day = w * 7 + i - offset + 1
                    const inMonth = day >= 1 && day <= total
                    const isToday = inMonth && toIsoDate(year, month, day) === today
                    const weekend = i === 0 || i === 6
                    const bg = isToday ? 'bg-slate-200' : !inMonth ? 'bg-slate-50' : weekend ? 'bg-slate-100' : ''
                    return (
                      <div
                        key={i}
                        data-testid={isToday ? 'today-cell' : undefined}
                        data-today={isToday || undefined}
                        data-weekend={weekend || undefined}
                        className={`border-l border-slate-200 px-2 pt-1 text-xs first:border-l-0 ${bg} ${inMonth ? 'text-slate-900' : 'text-slate-600'}`}
                      >
                        {inMonth && (
                          <span className={isToday ? 'inline-flex h-5 min-w-5 items-center justify-center rounded-sm bg-slate-900 px-1 font-semibold text-white' : ''}>{day}</span>
                        )}
                      </div>
                    )
                  })}
                </div>
                <div
                  className="relative grid grid-cols-7 gap-y-1 px-0 pb-2 pt-7"
                  style={{ gridTemplateRows: `repeat(${Math.max(lanes, 1)}, 1.75rem)` }}
                >
                  {segs.map((s) => {
                    const state = allocationState(s.a, today)
                    const meta = STATE_META[state]
                    const desc = `${s.a.project_name} - ${s.a.allocation_percent}% (${s.a.start_date} to ${s.a.end_date ?? 'Ongoing'}) - ${meta.label}`
                    return (
                      <div
                        key={`${s.a.id}-${w}`}
                        role="gridcell"
                        style={{ gridColumn: `${s.startCol} / ${s.endCol + 1}`, gridRow: s.lane + 1 }}
                        className="min-w-0"
                      >
                        <button
                          type="button"
                          data-testid="allocation-bar"
                          data-state={state}
                          data-open-ended={!s.a.end_date}
                          title={desc}
                          aria-label={desc}
                          aria-haspopup="dialog"
                          onClick={(e) =>
                            setSelected({ a: s.a, anchor: e.currentTarget.getBoundingClientRect(), trigger: e.currentTarget })
                          }
                          className={`mx-0.5 flex h-full w-[calc(100%-0.25rem)] items-center truncate border px-2 text-left text-xs font-medium focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 ${
                            meta.bar
                          } ${s.continuesLeft ? 'rounded-l-none border-l-0' : 'rounded-l-sm'} ${s.continuesRight ? 'rounded-r-none border-r-0' : 'rounded-r-sm'}`}
                        >
                          <span aria-hidden="true" className={`mr-1.5 h-1.5 w-1.5 shrink-0 rounded-full ${meta.dot}`} />
                          <span className="truncate">
                            {s.a.project_name} {s.a.allocation_percent}%
                            {!s.a.end_date && s.endCol === 7 && ' ->'}
                          </span>
                        </button>
                      </div>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      </div>
      )}
      {!narrow && selected && (
        <AllocationPopover
          allocation={selected.a}
          state={allocationState(selected.a, today)}
          anchor={selected.anchor}
          trigger={selected.trigger}
          onClose={closePopover}
        />
      )}
      <ul className="flex flex-wrap items-center gap-4 text-xs text-slate-600" aria-label="Legend" data-testid="calendar-legend">
        {STATE_ORDER.map((st) => (
          <li key={st} data-state={st} className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 font-medium ${STATE_META[st].bar}`}>
            <span aria-hidden="true" className={`h-1.5 w-1.5 rounded-full ${STATE_META[st].dot}`} />
            {STATE_META[st].label}
          </li>
        ))}
      </ul>
      <p className="text-xs text-slate-600">Past includes ended, completed and archived allocations. Open-ended allocations run to the end of the month.</p>
    </div>
  )
}
