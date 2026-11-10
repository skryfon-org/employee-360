import type { HolidayCategory } from '../queries/holidayQueries'

/** Category colour comes from tenant data (`#RRGGBB`), so the dot uses an inline style; text carries the meaning. */
export function CategoryDot({ color }: { color?: string }) {
  return <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-slate-600" style={color ? { backgroundColor: color } : undefined} />
}

export type CategoryStatus = 'loading' | 'error' | 'ready'

export function CategoryBadge({ category, status = 'ready' }: { category?: HolidayCategory; status?: CategoryStatus }) {
  if (!category && status !== 'ready') {
    return (
      <span
        aria-label={status === 'loading' ? 'Loading category' : 'Category unavailable'}
        className="inline-flex h-6 items-center rounded-sm border border-slate-200 bg-slate-50 px-2 text-xs text-slate-600"
      >
        {status === 'loading' ? '\u2026' : '\u2014'}
      </span>
    )
  }
  return (
    <span className="inline-flex h-6 items-center gap-1.5 rounded-sm border border-slate-300 bg-slate-50 px-2 text-xs font-medium text-slate-900">
      <CategoryDot color={category?.color} />
      {category?.name ?? 'Uncategorised'}
    </span>
  )
}
