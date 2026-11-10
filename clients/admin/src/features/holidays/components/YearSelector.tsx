export function YearSelector({ year, onChange }: { year: number; onChange: (year: number) => void }) {
  const btn =
    'h-11 w-11 rounded-sm border border-slate-300 bg-white text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9 md:w-9'
  return (
    <div role="group" aria-label="Year" className="flex items-center gap-2">
      <button type="button" className={btn} aria-label="Previous year" onClick={() => onChange(year - 1)}>
        &lsaquo;
      </button>
      <span aria-live="polite" className="min-w-16 text-center text-base font-semibold text-slate-900">
        {year}
      </span>
      <button type="button" className={btn} aria-label="Next year" onClick={() => onChange(year + 1)}>
        &rsaquo;
      </button>
    </div>
  )
}
