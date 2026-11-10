import { getErrorMessage } from '@employee360/api-client'
import { Card, HolidayCalendar, InlineAlert, PageContainer, PageHeader, toIsoDate, daysInMonth, type CalendarDayLabel } from '@employee360/ui'
import { useEffect, useId, useMemo, useState } from 'react'
import { ViewToggle, type StatusView } from '../components/ViewToggle'
import { DayEntries } from '../components/DayEntries'
import type { ProjectOption } from '../components/EntryForm'
import { WeekStrip } from '../components/WeekStrip'
import { isPlainShortcut } from '../shortcuts'
import { addDays, dayMonth, isEditableDate, lastSevenDays, longDate, monthRange } from '../dates'
import { describeDayStatus, type DayStatusItem } from '../dayStatus'
import { useLoggableProjectsQuery, useMyDayStatusesQuery, useMyStatusLogsQuery } from '../queries/statusLogQueries'

export default function DailyStatusPage() {
  const days = useMemo(() => lastSevenDays(), [])
  const ordered = useMemo(() => [...days].reverse(), [days])
  const [selected, setSelected] = useState(days[0])
  const [view, setView] = useState<StatusView>('week')
  const [cursor, setCursor] = useState(() => ({ year: Number(days[0].slice(0, 4)), month: Number(days[0].slice(5, 7)) }))
  const panelId = useId()
  const calendar = view === 'calendar'
  const range = monthRange(cursor.year, cursor.month)
  const monthEnd = range.to < days[0] ? range.to : days[0]
  const weekLogs = useMyStatusLogsQuery(ordered[0], days[0])
  // The server rejects ranges over 366 days; one month is far below. Never ask past today.
  const monthLogs = useMyStatusLogsQuery(range.from, monthEnd, calendar)
  // Why a day has no log (holiday, weekly off, leave). Best effort: if it fails, days fall back to "nothing logged".
  const weekStatuses = useMyDayStatusesQuery(ordered[0], days[0])
  const monthStatuses = useMyDayStatusesQuery(range.from, monthEnd, calendar)
  const statusList = (calendar ? monthStatuses : weekStatuses).data?.data
  const statusByDate = useMemo(() => new Map<string, DayStatusItem>((statusList ?? []).map((s) => [s.date ?? '', s])), [statusList])
  const logs = calendar ? monthLogs : weekLogs
  // The selected day's entries come from their own request (from = to = selected), never from the range result.
  const dayLogs = useMyStatusLogsQuery(selected, selected)

  const select = (date: string) => {
    setSelected(date)
    setCursor({ year: Number(date.slice(0, 4)), month: Number(date.slice(5, 7)) })
  }
  const changeView = (v: StatusView) => {
    setView(v)
    if (v === 'week' && !days.includes(selected)) select(days[0])
  }
  const changeMonth = ({ year, month }: { year: number; month: number }) => {
    const target = toIsoDate(year, month, Math.min(Number(selected.slice(8)), daysInMonth(year, month)))
    select(target > days[0] ? days[0] : target)
  }
  const projectsQuery = useLoggableProjectsQuery()

  const projects: ProjectOption[] = useMemo(() => {
    const seen = new Set<string>()
    const out: ProjectOption[] = []
    for (const a of projectsQuery.data?.data?.allocations ?? []) {
      if (!a.project_id || seen.has(a.project_id)) continue
      seen.add(a.project_id)
      out.push({ id: a.project_id, label: a.project_name ?? a.project_id, startDate: a.start_date, endDate: a.end_date })
    }
    return out
  }, [projectsQuery.data])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const steps: Record<string, number> = calendar
        ? { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -7, ArrowDown: 7 }
        : { ArrowLeft: -1, ArrowRight: 1 }
      const step = steps[e.key]
      if (step === undefined || !isPlainShortcut(e)) return
      if (e.target instanceof HTMLElement && e.target.closest('[role="tab"]')) return
      e.preventDefault()
      if (calendar) {
        // Any allowed date, never into the future.
        const next = addDays(selected, step)
        if (next <= days[0]) {
          select(next)
        }
      } else {
        setSelected((cur) => ordered[Math.min(ordered.length - 1, Math.max(0, ordered.indexOf(cur) + step))])
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [ordered, calendar, selected, days])

  const groups = logs.data?.data ?? []
  const byDate = new Map(groups.map((g) => [g.date ?? '', g]))
  const week = ordered.map((date) => ({ date, total: byDate.get(date)?.total_hours ?? 0, status: statusByDate.get(date) }))
  const totals = new Map(groups.map((g) => [g.date ?? '', g.total_hours ?? 0]))
  const dayNote = describeDayStatus((weekStatuses.data?.data ?? monthStatuses.data?.data)?.find((x) => x.date === selected) ?? statusByDate.get(selected))
  const dayGroup = dayLogs.data?.data?.[0]
  const monthTotal = Math.round([...totals.values()].reduce((a, b) => a + b, 0) * 100) / 100
  const dayLabels = useMemo(() => {
    const out: CalendarDayLabel[] = []
    for (let d = 1; d <= daysInMonth(cursor.year, cursor.month); d++) {
      const date = toIsoDate(cursor.year, cursor.month, d)
      if (date > days[0]) break
      const total = totals.get(date) ?? 0
      const editable = isEditableDate(date)
      const name = dayMonth(date)
      if (total > 0) out.push({ date, text: `${total}h`, ariaLabel: `${name}: ${total} ${total === 1 ? 'hour' : 'hours'} logged` })
      else {
        const info = describeDayStatus(statusByDate.get(date))
        if (info?.tone === 'info') out.push({ date, text: info.label, tone: 'info', ariaLabel: `${name}: ${info.title}` })
        else if (info) out.push({ date, text: 'Missing', tone: 'warning', ariaLabel: `${name}: ${info.title}` })
        else out.push({ date, text: editable ? 'None' : '–', muted: true, ariaLabel: `${name}: nothing logged` })
      }
      if (editable) out.push({ date, text: 'Edit', muted: true, ariaLabel: `${name}: editable` })
    }
    return out
  }, [cursor, totals, days, statusByDate])

  return (
    <PageContainer>
      <PageHeader title="Daily status" description={longDate(selected)} actions={<ViewToggle view={view} onChange={changeView} panelId={panelId} />} />
      <Card>
        <div role="tabpanel" id={panelId} aria-labelledby={`${panelId}-tab-${view}`}>
          {calendar ? (
            <>
              <HolidayCalendar
                ariaLabel="Daily status calendar"
                showScopeToggle={false}
                scope="month"
                onScopeChange={() => {}}
                cursor={cursor}
                onCursorChange={changeMonth}
                holidays={[]}
                maxMonth={{ year: Number(days[0].slice(0, 4)), month: Number(days[0].slice(5, 7)) }}
                selectedDate={selected}
                isDateDisabled={(d) => d > days[0]}
                dayLabels={dayLabels}
                onDateClick={select}
              />
              <p className="mt-2 text-sm text-slate-900">
                Month: <strong>{monthTotal}h</strong>
              </p>
              <p className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-700" aria-label="Legend">
                <span><span className="font-medium text-blue-700">Holiday</span> public holiday</span>
                <span><span className="font-medium text-blue-700">Weekly off</span> no work expected</span>
                <span><span className="font-medium text-blue-700">On leave</span> approved leave</span>
                <span><span className="font-medium text-amber-700">None</span> / <span className="font-medium text-amber-700">Missing</span> still to log</span>
              </p>
              <p className="mt-1 text-xs text-slate-600">Days marked Edit (last 7 days) can be changed. Older days are read-only; future days are unavailable.</p>
            </>
          ) : (
            weekLogs.isPending ? (
              <p role="status" className="text-sm text-slate-600">
                Loading your status…
              </p>
            ) : weekLogs.isError ? (
              <div className="flex flex-col items-start gap-3">
                <InlineAlert tone="error">{getErrorMessage(weekLogs.error, 'Could not load your status logs.')}</InlineAlert>
                <RetryButton onClick={() => void weekLogs.refetch()} />
              </div>
            ) : (
              <WeekStrip days={week} selected={selected} today={days[0]} onSelect={select} />
            )
          )}
        </div>
        <p className="mt-3 text-xs text-slate-600">
          Shortcuts: ← → days{calendar ? ', ↑ ↓ weeks' : ''}, N new entry, Ctrl+Enter save
        </p>
      </Card>
      {dayLogs.isPending ? (
        <Card>
          <p role="status" className="text-sm text-slate-600">
            Loading your status…
          </p>
        </Card>
      ) : dayLogs.isError ? (
        <Card className="flex flex-col items-start gap-3">
          <InlineAlert tone="error">{getErrorMessage(dayLogs.error, 'Could not load your status logs.')}</InlineAlert>
          <RetryButton onClick={() => void dayLogs.refetch()} />
        </Card>
      ) : (
        <>
        {dayNote && (
          <p role="note" className={`rounded-sm border px-3 py-2 text-sm ${dayNote.tone === 'warning' ? 'border-amber-300 bg-amber-50 text-amber-700' : 'border-blue-300 bg-blue-50 text-blue-700'}`}>
            {dayNote.note}
          </p>
        )}
        <DayEntries key={selected} date={selected} entries={dayGroup?.entries ?? []} total={dayGroup?.total_hours ?? 0} projects={projects} />
        </>
      )}
    </PageContainer>
  )
}

function RetryButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
    >
      Retry
    </button>
  )
}
