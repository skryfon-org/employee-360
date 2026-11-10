import { PROJECT_STATUS_LABELS, type ProjectStatus } from '../schemas/projectSchemas'

const STYLES: Record<ProjectStatus, { badge: string; dot: string }> = {
  active: { badge: 'bg-green-50 border-green-300 text-green-700', dot: 'bg-green-700' },
  on_hold: { badge: 'bg-amber-50 border-amber-300 text-amber-700', dot: 'bg-amber-700' },
  completed: { badge: 'bg-slate-100 border-slate-300 text-slate-600', dot: 'bg-slate-600' },
  archived: { badge: 'bg-slate-100 border-slate-300 text-slate-600', dot: 'bg-slate-600' },
}

export function ProjectStatusBadge({ status }: { status?: string }) {
  const key = (status && status in STYLES ? status : 'active') as ProjectStatus
  const s = STYLES[key]
  return (
    <span className={`inline-flex h-6 items-center gap-1.5 rounded-sm border px-2 text-xs font-medium ${s.badge}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${s.dot}`} />
      {PROJECT_STATUS_LABELS[key]}
    </span>
  )
}
