import { Skeleton, SkeletonRegion } from '@employee360/ui'

const COLS = ['w-32', 'w-40', 'w-40', 'w-32', 'w-16']

export function AuditLogsTableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <SkeletonRegion label="Loading audit log…">
      <div aria-hidden="true">
        <div className="flex gap-4 bg-slate-100 px-4 py-3">
          {COLS.map((w, i) => (
            <div key={i} className="flex-1">
              <Skeleton className={`h-3 ${w} max-w-full`} />
            </div>
          ))}
        </div>
        {Array.from({ length: rows }, (_, r) => (
          <div key={r} className="flex items-center gap-4 border-t border-slate-200 px-4 py-3">
            {COLS.map((w, i) => (
              <div key={i} className="flex-1">
                <Skeleton className={`h-4 ${w} max-w-full`} />
              </div>
            ))}
          </div>
        ))}
      </div>
    </SkeletonRegion>
  )
}
