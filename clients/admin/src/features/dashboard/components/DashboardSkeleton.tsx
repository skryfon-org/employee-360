import { Card, Skeleton, SkeletonRegion } from '@employee360/ui'

/**
 * Mirrors DashboardBody: optional organisation panel (`withOrganisation`, super-admin only),
 * then four KPI cards and two KPI cards.
 */
export function DashboardSkeleton({ withOrganisation = false }: { withOrganisation?: boolean }) {
  return (
    <SkeletonRegion label="Loading dashboard…" className="flex flex-col gap-6">
      {withOrganisation && (
        <Card className="flex items-center gap-4">
          <Skeleton className="h-12 w-12" />
          <div className="flex flex-col gap-2">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-6 w-40" />
          </div>
        </Card>
      )}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <Card key={i} className="!p-4 sm:!p-4">
            <Skeleton className="h-4 w-4" />
            <Skeleton className="mt-3 h-3 w-20" />
            <Skeleton className="mt-2 h-7 w-12" />
          </Card>
        ))}
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {Array.from({ length: 2 }, (_, i) => (
          <Card key={i} className="!p-4 sm:!p-4">
            <Skeleton className="h-4 w-4" />
            <Skeleton className="mt-3 h-3 w-20" />
            <Skeleton className="mt-2 h-7 w-12" />
          </Card>
        ))}
      </div>
    </SkeletonRegion>
  )
}
