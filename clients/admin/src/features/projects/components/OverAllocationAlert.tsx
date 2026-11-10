/** Warn-only: over-allocation (total > 100%) never blocks saving. */
export function OverAllocationAlert({ total }: { total: number }) {
  return (
    <div role="status" className="flex gap-2 rounded-sm border border-amber-300 bg-amber-50 p-3 text-sm text-amber-700">
      <svg className="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path d="M10 2 1 18h18L10 2zm-1 6h2v5H9V8zm0 6h2v2H9v-2z" />
      </svg>
      <p>Over-allocated: total allocation is {total}%, which exceeds 100%. This is a warning only and does not block changes.</p>
    </div>
  )
}

export function TotalPercentBadge({ total }: { total: number }) {
  const over = total > 100
  return (
    <span
      className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${
        over ? 'border-amber-300 bg-amber-50 text-amber-700' : 'border-slate-300 bg-slate-100 text-slate-900'
      }`}
    >
      Total {total}%{over && ' - over-allocated'}
    </span>
  )
}
