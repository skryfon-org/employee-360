import type { HolidayCategory } from '../queries/holidayQueries'

/** Category colour comes from tenant data (`#RRGGBB`), so the dot uses an inline style; text carries the meaning. */
export function CategoryDot({ color }: { color?: string }) {
  return <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-slate-600" style={color ? { backgroundColor: color } : undefined} />
}

export function CategoryBadge({ category }: { category?: HolidayCategory }) {
  return (
    <span className="inline-flex h-6 items-center gap-1.5 rounded-sm border border-slate-300 bg-slate-50 px-2 text-xs font-medium text-slate-900">
      <CategoryDot color={category?.color} />
      {category?.name ?? 'Uncategorised'}
    </span>
  )
}
