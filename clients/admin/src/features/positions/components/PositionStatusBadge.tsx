export function PositionStatusBadge({ active }: { active?: boolean }) {
  const cls = active
    ? 'bg-green-50 border-green-300 text-green-700'
    : 'bg-slate-100 border-slate-300 text-slate-600'
  const dot = active ? 'bg-green-700' : 'bg-slate-600'
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${cls}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${dot}`} />
      {active ? 'Active' : 'Inactive'}
    </span>
  )
}
