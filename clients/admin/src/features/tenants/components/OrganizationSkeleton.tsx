import { Card, Skeleton, SkeletonRegion } from '@employee360/ui'

function Heading() {
  return (
    <div className="flex flex-col gap-2">
      <Skeleton className="h-5 w-40" />
      <Skeleton className="h-4 w-full max-w-md" />
    </div>
  )
}

/** Mirrors RenameTenantForm: heading + description, then input + button. */
function GeneralSkeleton() {
  return (
    <Card>
      <div className="flex flex-col gap-4">
        <Heading />
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <Skeleton className="h-4 w-12" />
            <Skeleton className="h-9 w-full" />
          </div>
          <Skeleton className="h-9 w-full sm:mt-5 sm:w-24" />
        </div>
      </div>
    </Card>
  )
}

/** Mirrors DomainsManager: heading, a few rows, then the add-domain form. */
function DomainsSkeleton() {
  return (
    <Card>
      <div className="flex flex-col gap-4">
        <Heading />
        <ul className="divide-y divide-slate-200 rounded-sm border border-slate-200">
          {Array.from({ length: 3 }, (_, i) => (
            <li key={i} className="flex items-center justify-between gap-3 p-3">
              <Skeleton className="h-4 w-40" />
              <div className="flex gap-2">
                <Skeleton className="h-8 w-14" />
                <Skeleton className="h-8 w-16" />
              </div>
            </li>
          ))}
        </ul>
        <div className="flex flex-col gap-3 border-t border-line pt-4 sm:flex-row sm:items-start">
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-9 w-full" />
          </div>
          <Skeleton className="h-9 w-full sm:mt-5 sm:w-28" />
        </div>
      </div>
    </Card>
  )
}

export function OrganizationSkeleton({ tab }: { tab: string }) {
  return (
    <SkeletonRegion label="Loading organization…">
      {tab === 'domains' ? <DomainsSkeleton /> : <GeneralSkeleton />}
    </SkeletonRegion>
  )
}
