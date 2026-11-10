import { Link } from 'react-router-dom'
import type { Crumb } from './breadcrumbs-utils'

const LINK =
  'rounded-sm text-slate-900 hover:text-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2'

/**
 * Presentational breadcrumb trail. Ancestors are links; the last item is plain
 * text with aria-current="page". On small screens only the last two crumbs
 * show, preceded by an ellipsis; labels truncate instead of wrapping.
 */
export function Breadcrumbs({ items, className = '' }: { items: Crumb[]; className?: string }) {
  if (items.length === 0) return null
  const lastIndex = items.length - 1
  return (
    <nav aria-label="Breadcrumb" className={className}>
      <ol className="flex min-w-0 items-center gap-2 text-xs text-slate-600 sm:text-sm">
        {items.length > 2 && (
          <li aria-hidden="true" className="flex shrink-0 items-center gap-2 sm:hidden">
            <span>…</span>
            <span>/</span>
          </li>
        )}
        {items.map((c, i) => {
          const isLast = i === lastIndex
          const hiddenOnMobile = i < lastIndex - 1
          return (
            <li key={c.to} className={`min-w-0 items-center gap-2 ${hiddenOnMobile ? 'hidden sm:flex' : 'flex'}`}>
              {isLast ? (
                <span aria-current="page" title={c.label} className="truncate font-medium text-slate-900">
                  {c.label}
                </span>
              ) : (
                <>
                  <Link to={c.to} title={c.label} className={`${LINK} truncate`}>
                    {c.label}
                  </Link>
                  <span aria-hidden="true" className="shrink-0">/</span>
                </>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
