import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert } from '@employee360/ui'
import { EntryList } from './EntryList'
import { useEmployeeStatusLogsQuery } from '../queries/statusLogQueries'

/** Drill-down: one employee's entries (with notes) for a single date. */
export function DayEntries({ employeeId, date }: { employeeId: string; date: string }) {
  const { data, isPending, isError, error } = useEmployeeStatusLogsQuery(employeeId, date, date)
  if (isPending) return <p role="status" className="text-sm text-slate-600">Loading entries…</p>
  if (isError) return <InlineAlert tone="error">{getErrorMessage(error, 'Could not load entries.')}</InlineAlert>
  const entries = (data ?? []).flatMap((g) => g.entries ?? [])
  return <EntryList entries={entries} />
}
