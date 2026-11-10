import { Skeleton, SkeletonRegion } from '@employee360/ui'

export function HolidaysSkeleton() {
  return (
    <SkeletonRegion label="Loading holidays" className="flex flex-col gap-3 p-4">
      {Array.from({ length: 5 }, (_, i) => (
        <Skeleton key={i} className="h-12 w-full" />
      ))}
    </SkeletonRegion>
  )
}
