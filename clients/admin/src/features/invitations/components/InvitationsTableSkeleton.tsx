import { Skeleton, SkeletonRegion } from '@employee360/ui'

const COLS = ['w-44', 'w-16', 'w-24', 'w-20', 'w-20', 'w-16', 'w-24']

/** Mirrors InvitationsTable: header + rows at sm and up, stacked cards below. */
export function InvitationsTableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <SkeletonRegion label="Loading invitations…">
      <div className="overflow-x-auto max-sm:hidden">
        <div className="min-w-[64rem]">
          <div className="flex gap-4 bg-slate-100 px-4 py-3" aria-hidden="true">
            {COLS.map((w, i) => (
              <div key={i} className="flex-1"><Skeleton className={`h-3 ${w}`} /></div>
            ))}
          </div>
          {Array.from({ length: rows }, (_, r) => (
            <div key={r} className="flex items-center gap-4 border-t border-slate-200 px-4 py-3" aria-hidden="true">
              {COLS.map((w, i) => (
                <div key={i} className="flex-1"><Skeleton className={`h-4 ${w}`} /></div>
              ))}
            </div>
          ))}
        </div>
      </div>
      <div className="sm:hidden" aria-hidden="true">
        {Array.from({ length: Math.min(rows, 3) }, (_, r) => (
          <div key={r} className="flex flex-col gap-2 border-t border-slate-200 p-4 first:border-t-0">
            <Skeleton className="h-4 w-48" />
            {['w-16', 'w-24', 'w-20'].map((w, i) => (
              <div key={i} className="flex items-center justify-between">
                <Skeleton className="h-3 w-16" />
                <Skeleton className={`h-4 ${w}`} />
              </div>
            ))}
          </div>
        ))}
      </div>
    </SkeletonRegion>
  )
}
