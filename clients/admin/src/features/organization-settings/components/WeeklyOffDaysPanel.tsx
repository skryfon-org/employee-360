import { getErrorMessage } from '@employee360/api-client'
import { Card, InlineAlert, Skeleton } from '@employee360/ui'
import { WeeklyOffDaysForm } from './WeeklyOffDaysForm'
import { useOrgSettingsQuery } from '../queries/orgSettingsQueries'

/** Weekly off days; rendered as the Organization > Working days tab. */
export function WeeklyOffDaysPanel() {
  const { data, isPending, isError, error, refetch } = useOrgSettingsQuery()
  return (
    <Card aria-label="Weekly off days settings" role="region" className="flex flex-col gap-6">
      <div className="min-w-0">
        <h2 className="text-base font-semibold text-ink">Weekly off days</h2>
        <p className="mt-1 text-sm text-ink-muted">
          Days excluded from working days in the monthly status report. Hours logged on an off day still count.
        </p>
      </div>
      {isPending && (
        <div role="status" aria-label="Loading settings" className="flex flex-col gap-3">
          <Skeleton className="h-9 w-full" />
        </div>
      )}
      {isError && (
        <div className="flex flex-col gap-3">
          <InlineAlert tone="error">{getErrorMessage(error, 'Could not load organization settings.')}</InlineAlert>
          <button type="button" onClick={() => void refetch()} className="h-9 self-start rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100">
            Retry
          </button>
        </div>
      )}
      {data && <WeeklyOffDaysForm settings={data} />}
    </Card>
  )
}
