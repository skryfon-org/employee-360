import type { HolidayCategory } from '../queries/holidayQueries'
import { CategoryBadge } from './CategoryBadge'

export function CategoryLegend({ categories }: { categories: HolidayCategory[] }) {
  if (categories.length === 0) return null
  return (
    <ul aria-label="Categories" className="flex flex-wrap gap-2">
      {categories.map((c) => (
        <li key={c.id}>
          <CategoryBadge category={c} />
        </li>
      ))}
    </ul>
  )
}
