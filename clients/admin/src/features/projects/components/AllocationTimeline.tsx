import {
  ALLOCATION_STATES,
  ALLOCATION_STATE_LABELS,
  ALLOCATION_STATE_STYLES,
  allocationState,
  todayISO,
} from '../allocationState'
import { AllocationStateBadge } from './AllocationStateBadge'

export interface TimelineRow {
  id: string
  name: string
  percent?: number
  start: string
  end?: string | null
}

type Granularity = 'day' | 'week' | 'month'

const DAY_MS = 86_400_000
const UNIT_WIDTH_REM: Record<Granularity, number> = { day: 3, week: 4.5, month: 6 }
const EMPLOYEE_COL_REM = 11
const monthFmt = new Intl.DateTimeFormat('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' })
const weekdayFmt = new Intl.DateTimeFormat('en-US', { weekday: 'short', timeZone: 'UTC' })
const dateFmt = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' })

const dayNum = (iso: string) => Date.UTC(+iso.slice(0, 4), +iso.slice(5, 7) - 1, +iso.slice(8, 10)) / DAY_MS
const toDate = (day: number) => new Date(day * DAY_MS)
const monthStartDay = (day: number, add = 0) => {
  const d = toDate(day)
  return Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + add, 1) / DAY_MS
}
/** Monday on or before the given day (day 0 = Thursday 1970-01-01). */
const mondayOf = (day: number) => day - ((((day + 3) % 7) + 7) % 7)

interface Column {
  key: string
  day: number
  label: string
}

/** Pick a granularity from the span and return the padded, aligned range plus its columns. */
function buildAxis(rows: TimelineRow[], projectEnd?: string | null) {
  const starts = rows.map((r) => dayNum(r.start))
  const ends = rows.filter((r) => r.end).map((r) => dayNum(r.end as string) + 1)
  const min = Math.min(...starts)
  const latestKnown = Math.max(...starts, ...ends)
  const hasOpen = rows.some((r) => !r.end)
  let max = latestKnown
  if (hasOpen) {
    if (projectEnd && dayNum(projectEnd) + 1 >= latestKnown) max = dayNum(projectEnd) + 1
    else max = latestKnown + Math.max(7, Math.round((latestKnown - min) * 0.15))
  }
  const span = Math.max(max - min, 1)
  const granularity: Granularity = span <= 42 ? 'day' : span <= 183 ? 'week' : 'month'

  let start: number
  let end: number
  const columns: Column[] = []
  if (granularity === 'day') {
    start = min - 1
    end = max + 1
    for (let d = start; d < end; d += 1) {
      columns.push({ key: String(d), day: d, label: `${weekdayFmt.format(toDate(d))} ${toDate(d).getUTCDate()}` })
    }
  } else if (granularity === 'week') {
    start = mondayOf(min) - 7
    end = mondayOf(max) + 14
    for (let d = start; d < end; d += 7) columns.push({ key: String(d), day: d, label: dateFmt.format(toDate(d)) })
  } else {
    start = monthStartDay(min)
    end = monthStartDay(max, 2)
    for (let d = start; d < end; d = monthStartDay(d, 1)) {
      columns.push({ key: String(d), day: d, label: monthFmt.format(toDate(d)) })
    }
  }
  return { granularity, start, end, columns }
}

interface Props {
  rows: TimelineRow[]
  projectStart?: string
  projectEnd?: string | null
}

/** Auto-fitting chart: the range covers every row (project dates never stretch it); granularity follows the span. */
export function AllocationTimeline({ rows, projectStart, projectEnd }: Props) {
  const today = todayISO()
  const todayDay = dayNum(today)
  if (rows.length === 0) {
    return <p className="text-sm text-slate-600">No allocations to show.</p>
  }
  const { granularity, start: rangeStart, end: rangeEnd, columns } = buildAxis(rows, projectEnd)
  const total = rangeEnd - rangeStart
  const pct = (day: number) => ((day - rangeStart) / total) * 100
  const trackMin = `${columns.length * UNIT_WIDTH_REM[granularity]}rem`

  // Month labels grouped above day/week columns.
  const groups: { key: string; day: number; end: number; label: string }[] = []
  if (granularity !== 'month') {
    columns.forEach((c, i) => {
      const next = columns[i + 1]?.day ?? rangeEnd
      const label = monthFmt.format(toDate(c.day))
      const last = groups[groups.length - 1]
      if (last && last.label === label) last.end = next
      else groups.push({ key: c.key, day: c.day, end: next, label })
    })
  }

  const markers: { key: string; day: number; text: string }[] = []
  if (projectStart) markers.push({ key: 'ps', day: dayNum(projectStart), text: 'Project start' })
  if (projectEnd) markers.push({ key: 'pe', day: dayNum(projectEnd) + 1, text: 'Project end' })
  const visibleMarkers = markers.filter((m) => m.day >= rangeStart && m.day <= rangeEnd)
  const todayVisible = todayDay >= rangeStart && todayDay < rangeEnd

  const presentStates = new Set(rows.map((r) => allocationState({ start_date: r.start, end_date: r.end }, today)))
  const legendStates = ALLOCATION_STATES.filter((s) => presentStates.has(s))

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-slate-600">
        {projectStart ? `Project starts ${projectStart}` : ''}
        {projectEnd ? `, ends ${projectEnd}.` : projectStart ? ', no end date.' : ''} Dashed lines mark the project start
        and end, the blue line is today, and an arrow means the allocation has no end date.
      </p>
      <div
        className="overflow-x-auto rounded-sm border border-slate-200"
        tabIndex={0}
        aria-label="Allocation timeline, scrollable"
        data-granularity={granularity}
      >
        <div style={{ minWidth: `calc(${EMPLOYEE_COL_REM}rem + ${trackMin})` }}>
          <div className="flex border-b border-slate-200 bg-slate-100 text-xs font-medium text-slate-600">
            <div className="sticky left-0 z-10 flex w-44 shrink-0 items-end bg-slate-100 px-3 py-2">Employee</div>
            <div className="flex min-w-0 flex-1 flex-col" aria-hidden="true">
              {granularity !== 'month' && (
                <div className="relative h-7 border-b border-slate-200">
                  {groups.map((g) => (
                    <div
                      key={g.key}
                      className="absolute inset-y-0 overflow-hidden whitespace-nowrap border-l border-slate-200 px-2 py-1.5"
                      style={{ left: `${pct(g.day)}%`, width: `${((g.end - g.day) * 100) / total}%` }}
                    >
                      {g.label}
                    </div>
                  ))}
                </div>
              )}
              <div className="relative h-8">
                {columns.map((c, i) => {
                  const next = columns[i + 1]?.day ?? rangeEnd
                  return (
                    <div
                      key={c.key}
                      data-testid="axis-label"
                      className="absolute inset-y-0 overflow-hidden whitespace-nowrap border-l border-slate-200 px-1 py-2 text-center"
                      style={{ left: `${pct(c.day)}%`, width: `${((next - c.day) * 100) / total}%` }}
                    >
                      {c.label}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
          <ul aria-label="Allocation timeline rows">
            {rows.map((r) => {
              const state = allocationState({ start_date: r.start, end_date: r.end }, today)
              const openEnded = !r.end
              const left = pct(dayNum(r.start))
              const right = openEnded ? 100 : pct(dayNum(r.end as string) + 1)
              const desc = `${r.name}, ${r.percent ?? '?'}% allocation, ${state}, ${r.start} to ${r.end ?? 'open-ended'}`
              const st = ALLOCATION_STATE_STYLES[state]
              return (
                <li key={r.id} className="flex border-t border-slate-200 text-sm text-slate-900 first:border-t-0">
                  <div className="sticky left-0 z-10 flex w-44 shrink-0 flex-col items-start gap-1 bg-white px-3 py-2">
                    <span className="max-w-full truncate font-medium">{r.name}</span>
                    <AllocationStateBadge state={state} />
                  </div>
                  <div className="relative h-14 min-w-0 flex-1">
                    {columns.map((c) => (
                      <div
                        key={c.key}
                        aria-hidden="true"
                        className="absolute inset-y-0 border-l border-slate-200"
                        style={{ left: `${pct(c.day)}%` }}
                      />
                    ))}
                    {visibleMarkers.map((m) => (
                      <div
                        key={m.key}
                        aria-hidden="true"
                        title={m.text}
                        className="absolute inset-y-0 border-l-2 border-dashed border-slate-900"
                        style={{ left: `${pct(m.day)}%` }}
                      />
                    ))}
                    {todayVisible && (
                      <div
                        aria-hidden="true"
                        data-testid="today-marker"
                        title="Today"
                        className="absolute inset-y-0 z-[1] border-l-2 border-blue-600"
                        style={{ left: `${pct(todayDay)}%` }}
                      />
                    )}
                    <div
                      role="img"
                      aria-label={desc}
                      data-testid="timeline-bar"
                      className={`absolute top-3 flex h-8 items-center gap-1 overflow-hidden border px-2 text-xs font-medium rounded-l-sm ${st.box} ${
                        openEnded ? 'border-r-0' : 'rounded-r-sm'
                      }`}
                      style={{ left: `${left}%`, width: `${Math.max(right - left, 0.5)}%` }}
                    >
                      <span className="whitespace-nowrap">{r.percent ?? '?'}%</span>
                      {openEnded && <span className="whitespace-nowrap" aria-hidden="true">{'→'}</span>}
                    </div>
                  </div>
                </li>
              )
            })}
          </ul>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-ink-muted">
        <span className="font-medium">Legend</span>
        <ul aria-label="Timeline legend" className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
          <li className="flex items-center gap-1.5">
            <span aria-hidden="true" className="inline-block h-4 border-l-2 border-blue-600" />
            <span>Today{!todayVisible && ' (not in view)'}</span>
          </li>
          <li className="flex items-center gap-1.5">
            <span aria-hidden="true" className="inline-block h-4 border-l-2 border-dashed border-slate-900" />
            <span>Project start / end{visibleMarkers.length === 0 && ' (not in view)'}</span>
          </li>
          {rows.some((r) => !r.end) && (
            <li className="flex items-center gap-1.5">
              <span aria-hidden="true">{'→'}</span>
              No end date (continues)
            </li>
          )}
          {legendStates.map((s) => (
            <li key={s} className="flex items-center gap-1.5">
              <span
                aria-hidden="true"
                className={`inline-block h-3 w-5 rounded-sm border ${ALLOCATION_STATE_STYLES[s].box}`}
              />
              {ALLOCATION_STATE_LABELS[s]}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
