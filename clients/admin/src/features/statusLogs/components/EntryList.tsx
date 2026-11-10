import { formatHours } from '../dates'
import type { StatusLogEntry } from '../queries/statusLogQueries'

const humanize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1).replace(/_/g, ' ')

/** Entries of a single day with project/category, hours and notes. */
export function EntryList({ entries }: { entries: StatusLogEntry[] }) {
  if (entries.length === 0) return <p className="text-sm text-slate-600">No entries.</p>
  return (
    <ul className="flex flex-col divide-y divide-slate-200">
      {entries.map((e) => (
        <li key={e.id} className="flex flex-col gap-1 py-2 first:pt-0 last:pb-0">
          <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
            <span className="font-medium text-slate-900">
              {e.project_name ? `${e.project_name}${e.project_code ? ` (${e.project_code})` : ''}` : humanize(e.activity_category ?? 'other')}
            </span>
            <span className="text-slate-900">{formatHours(e.hours)}</span>
          </div>
          {e.notes ? <p className="whitespace-pre-wrap break-words text-sm text-slate-600">{e.notes}</p> : null}
        </li>
      ))}
    </ul>
  )
}
