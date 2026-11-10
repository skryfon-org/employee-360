import { useMyProjectsQuery, type ProjectScope } from '../queries/projectQueries'

/** "Total today: N%" from the /me/projects response; amber + text (not color alone) when over-allocated. */
export function TotalAllocationChip({ scope }: { scope: ProjectScope }) {
  const { data } = useMyProjectsQuery(scope)
  const info = data?.data
  if (!info || info.total_percent === undefined) return null
  const over = !!info.over_allocated
  return (
    <p
      data-testid="total-allocation"
      data-over-allocated={over}
      className={`inline-flex min-h-8 items-center gap-2 self-start rounded-sm border px-3 py-1 text-sm font-medium ${
        over ? 'border-amber-300 bg-amber-50 text-amber-700' : 'border-slate-300 bg-slate-100 text-slate-900'
      }`}
    >
      {over && (
        <svg aria-hidden="true" viewBox="0 0 20 20" className="h-4 w-4 shrink-0" fill="currentColor">
          <path d="M10 2 1 18h18L10 2Zm-1 6h2v5H9V8Zm0 6h2v2H9v-2Z" />
        </svg>
      )}
      <span>Total today: {info.total_percent}%</span>
      {over && <span>Over-allocated</span>}
    </p>
  )
}
