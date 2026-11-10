import { RowActionsMenu } from '@employee360/ui'
import type { Holiday } from '../queries/holidayQueries'
import { formatHolidayDate } from '../dates'
import { CategoryBadge } from './CategoryBadge'

export function HolidayList({
  holidays,
  onEdit,
  onDelete,
}: {
  holidays: Holiday[]
  onEdit: (h: Holiday) => void
  onDelete: (h: Holiday) => void
}) {
  const sorted = [...holidays].sort((a, b) => a.date.localeCompare(b.date) || a.name.localeCompare(b.name))
  return (
    <ul aria-label="Holidays" className="divide-y divide-slate-200">
      {sorted.map((h) => (
        <li key={h.id} className="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 flex-col gap-1">
            <p className="text-xs text-slate-600">{formatHolidayDate(h.date)}</p>
            <p className="text-sm font-semibold text-slate-900">{h.name}</p>
            {h.description && <p className="text-xs text-slate-600">{h.description}</p>}
          </div>
          <div className="flex items-center gap-2">
            <CategoryBadge category={h.category} />
            <RowActionsMenu
              label={h.name}
              actions={[
                { label: 'Edit', onSelect: () => onEdit(h) },
                { label: 'Delete', danger: true, onSelect: () => onDelete(h) },
              ]}
            />
          </div>
        </li>
      ))}
    </ul>
  )
}
