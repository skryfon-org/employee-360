import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { Card, EmptyState, FilterBar, InlineAlert } from '@employee360/ui'
import { addDays, formatHours, longDate, todayIso } from '../dates'
import { EntriesModal } from './EntriesModal'
import { EntryList } from './EntryList'
import { useEmployeeStatusLogsQuery } from '../queries/statusLogQueries'

const INPUT_CLS =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9'

/** "Status logs" tab on the employee detail page: entries grouped by day over a date range (default last 7 days). */
export function EmployeeStatusRange({ employeeId }: { employeeId: string }) {
  const [params] = useSearchParams()
  const iso = /^\d{4}-\d{2}-\d{2}$/
  const qFrom = params.get('from') ?? ''
  const qTo = params.get('to') ?? ''
  const prefilled = iso.test(qFrom) && iso.test(qTo) && qFrom <= qTo
  const [from, setFrom] = useState(prefilled ? qFrom : addDays(todayIso(), -6))
  const [to, setTo] = useState(prefilled ? qTo : todayIso())
  const invalid = from > to
  const { data, isPending, isError, error } = useEmployeeStatusLogsQuery(employeeId, from, to)
  const groups = [...(data ?? [])].sort((a, b) => (b.date ?? '').localeCompare(a.date ?? ''))

  return (
    <Card padded={false} className="overflow-hidden">
      <FilterBar className="grid grid-cols-2 gap-4 sm:max-w-md">
        <div className="flex flex-col gap-1">
          <label htmlFor="sl-from" className="text-xs font-medium text-slate-900">From</label>
          <input id="sl-from" type="date" className={INPUT_CLS} value={from} max={to} onChange={(e) => e.target.value && setFrom(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="sl-to" className="text-xs font-medium text-slate-900">To</label>
          <input id="sl-to" type="date" className={INPUT_CLS} value={to} min={from} onChange={(e) => e.target.value && setTo(e.target.value)} />
        </div>
      </FilterBar>
      {invalid ? (
        <div className="p-4">
          <InlineAlert tone="error">The start date must not be after the end date.</InlineAlert>
        </div>
      ) : isPending ? (
        <p role="status" className="p-4 text-sm text-slate-600">Loading status logs…</p>
      ) : isError ? (
        <div className="p-4">
          <InlineAlert tone="error">{getErrorMessage(error, 'Could not load status logs.')}</InlineAlert>
        </div>
      ) : groups.length === 0 ? (
        <EmptyState title="No status entries" description="Nothing was logged in this date range." />
      ) : (
        groups.map((g) => <DayCard key={g.date} group={g} />)
      )}
    </Card>
  )
}

type DayGroup = NonNullable<ReturnType<typeof useEmployeeStatusLogsQuery>['data']>[number]

/** One day's row; the View button opens the entries in a modal. */
function DayCard({ group: g }: { group: DayGroup }) {
  const [open, setOpen] = useState(false)
  const date = longDate(g.date as string)
  return (
    <div className="border-b border-slate-200 p-4 last:border-b-0">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-base font-semibold text-slate-900">{date}</h3>
        <div className="flex shrink-0 items-center gap-3">
          <span className="text-sm font-medium text-slate-900">{formatHours(g.total_hours)}</span>
          <button
            type="button"
            aria-haspopup="dialog"
            aria-label={`View entries for ${date}`}
            onClick={() => setOpen(true)}
            className="h-8 rounded-sm border border-slate-300 bg-white px-3 text-xs font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            View
          </button>
        </div>
      </div>
      {open && (
        <EntriesModal title={date} subtitle={`Total ${formatHours(g.total_hours)}`} onClose={() => setOpen(false)}>
          <EntryList entries={g.entries ?? []} />
        </EntriesModal>
      )}
    </div>
  )
}
