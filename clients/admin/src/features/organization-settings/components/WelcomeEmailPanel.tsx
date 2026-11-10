import { getErrorMessage } from '@employee360/api-client'
import { Card, InlineAlert, Skeleton } from '@employee360/ui'
import { WelcomeEmailForm } from './WelcomeEmailForm'
import { useOrgSettingsQuery } from '../queries/orgSettingsQueries'

/** Welcome email settings; rendered as the Organization > Welcome email tab. */
export function WelcomeEmailPanel() {
  const { data, isPending, isError, error, refetch } = useOrgSettingsQuery()

  return (
    <Card aria-label="Welcome email settings" role="region" className="flex flex-col gap-6">
      <div className="min-w-0">
        <h2 className="text-base font-semibold text-ink">Welcome email</h2>
        <p className="mt-1 text-sm text-ink-muted">
          Customise the message sent to new users after they accept their invitation.
        </p>
      </div>
      {isPending && (
        <div role="status" aria-label="Loading settings" className="flex flex-col gap-3">
          <Skeleton className="h-24 w-full" />
        </div>
      )}
      {isError && (
        <div className="flex flex-col gap-3">
          <InlineAlert tone="error">{getErrorMessage(error, 'Could not load organization settings.')}</InlineAlert>
          <button
            type="button"
            onClick={() => void refetch()}
            className="h-9 self-start rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100"
          >
            Retry
          </button>
        </div>
      )}
      {data && <WelcomeEmailForm settings={data} />}
    </Card>
  )
}
