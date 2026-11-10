import type { Holiday } from '../queries/holidayQueries'
import { formatHolidayDate } from '../dates'
import { CategoryBadge, type CategoryStatus } from './CategoryBadge'

/** Read-only chronological list; the caller supplies holidays already sorted by date. */
export function HolidayList({ holidays, categoryStatus = 'ready' }: { holidays: Holiday[]; categoryStatus?: CategoryStatus }) {
  return (
    <ul aria-label="Holidays" className="divide-y divide-slate-200">
      {holidays.map((h) => (
        <li key={h.id} className="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 flex-col gap-1">
            <p className="text-xs text-slate-600">{formatHolidayDate(h.date)}</p>
            <p className="text-sm font-semibold text-slate-900">{h.name}</p>
            {h.description && <p className="text-xs text-slate-600">{h.description}</p>}
          </div>
          <CategoryBadge category={h.category} status={categoryStatus} />
        </li>
      ))}
    </ul>
  )
}
