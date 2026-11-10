const STYLES: Record<string, { cls: string; dot: string; label: string }> = {
  pending: { cls: 'bg-amber-50 border-amber-300 text-amber-700', dot: 'bg-amber-700', label: 'Pending' },
  accepted: { cls: 'bg-green-50 border-green-300 text-green-700', dot: 'bg-green-700', label: 'Accepted' },
  revoked: { cls: 'bg-red-50 border-red-200 text-red-700', dot: 'bg-red-700', label: 'Revoked' },
  expired: { cls: 'bg-slate-100 border-slate-300 text-slate-600', dot: 'bg-slate-600', label: 'Expired' },
}

export function StatusBadge({ status }: { status?: string }) {
  const s = STYLES[status ?? ''] ?? STYLES.expired
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${s.cls}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${s.dot}`} />
      {status ? s.label : 'Unknown'}
    </span>
  )
}
