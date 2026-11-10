import { PAGE_SIZE_OPTIONS } from '../schemas/invitationListSchema'

const BTN =
  'h-11 md:h-8 min-w-8 rounded-sm border border-slate-300 bg-white px-3 text-xs font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500'

/** Compact page list: 1 … c-1 c c+1 … n */
function pageItems(page: number, total: number): (number | 'gap')[] {
  const set = new Set([1, total, page - 1, page, page + 1].filter((n) => n >= 1 && n <= total))
  const sorted = [...set].sort((a, b) => a - b)
  const out: (number | 'gap')[] = []
  sorted.forEach((n, i) => {
    if (i > 0 && n - sorted[i - 1] > 1) out.push('gap')
    out.push(n)
  })
  return out
}

interface Props {
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
}

export function InvitationsPagination({ page, pageSize, totalItems, totalPages, onPageChange, onPageSizeChange }: Props) {
  const from = totalItems === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(page * pageSize, totalItems)
  return (
    <div className="flex flex-col gap-3 border-t border-slate-200 bg-slate-100 px-4 py-2 text-xs text-slate-600 md:flex-row md:items-center md:justify-between">
      <p>
        Showing {from}–{to} of {totalItems}
      </p>
      <div className="flex flex-wrap items-center gap-2">
        <label className="flex items-center gap-2">
          Rows per page
          <select
            aria-label="Rows per page"
            value={pageSize}
            onChange={(e) => onPageSizeChange(Number(e.target.value))}
            className="h-11 md:h-8 rounded-sm border border-slate-300 bg-white px-2 text-xs text-slate-900 focus:outline-none focus:border-slate-900"
          >
            {PAGE_SIZE_OPTIONS.map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </label>
        <nav aria-label="Pagination" className="flex items-center gap-1">
          <button type="button" className={BTN} disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
            Previous
          </button>
          {pageItems(page, totalPages).map((it, i) =>
            it === 'gap' ? (
              <span key={`g${i}`} aria-hidden="true" className="px-1">…</span>
            ) : (
              <button
                key={it}
                type="button"
                aria-label={`Page ${it}`}
                aria-current={it === page ? 'page' : undefined}
                className={it === page ? `${BTN} !border-slate-900 !bg-slate-900 !text-white` : BTN}
                onClick={() => onPageChange(it)}
              >
                {it}
              </button>
            ),
          )}
          <button type="button" className={BTN} disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
            Next
          </button>
        </nav>
      </div>
    </div>
  )
}
