import { Skeleton, SkeletonRegion } from '@employee360/ui'

const COLS = ['w-32', 'w-64', 'w-24', 'w-24']

export function DepartmentsTableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <SkeletonRegion label="Loading departments…">
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
