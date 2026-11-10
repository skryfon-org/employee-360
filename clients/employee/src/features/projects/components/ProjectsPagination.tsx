const BTN =
  'h-11 md:h-8 min-w-[5rem] rounded-sm border border-slate-300 bg-white px-3 text-xs font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500'

interface Props {
  page: number
  pageSize: number
  totalItems: number
  onPageChange: (page: number) => void
}

/** Prev/next pager with a "Showing X–Y of Z" summary. Hidden when everything fits on one page. */
export function ProjectsPagination({ page, pageSize, totalItems, onPageChange }: Props) {
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize))
  if (totalPages <= 1) return null
  const from = (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)
  return (
    <div className="flex flex-col gap-3 border-t border-slate-200 bg-slate-100 px-4 py-2 text-xs text-slate-600 sm:flex-row sm:items-center sm:justify-between">
      <p>
        Showing {from}–{to} of {totalItems}
      </p>
      <nav aria-label="Pagination" className="flex items-center gap-2">
        <button type="button" className={BTN} disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
          Previous
        </button>
        <span aria-live="polite">
          Page {page} of {totalPages}
        </span>
        <button type="button" className={BTN} disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
          Next
        </button>
      </nav>
    </div>
  )
}
