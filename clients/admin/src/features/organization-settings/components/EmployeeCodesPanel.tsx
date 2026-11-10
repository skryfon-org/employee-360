import { getErrorMessage } from '@employee360/api-client'
import { Card, InlineAlert, Skeleton } from '@employee360/ui'
import { EmployeeCodeForm } from './EmployeeCodeForm'
import { useOrgSettingsQuery } from '../queries/orgSettingsQueries'

/** Employee code format settings; rendered as the Organization > Employee codes tab. */
export function EmployeeCodesPanel() {
  const { data, isPending, isError, error, refetch } = useOrgSettingsQuery()
  const issued = Boolean(data?.employee_codes_issued)

  return (
    <Card aria-label="Employee code settings" role="region" className="flex flex-col gap-6">
      <div className="min-w-0">
        <h2 className="text-base font-semibold text-ink">Employee codes</h2>
        <p className="mt-1 text-sm text-ink-muted">
          Set the format used to generate employee codes for new employees. Changing the prefix only affects employees added afterwards; existing codes are never rewritten.
        </p>
      </div>
      {isPending && (
        <div role="status" aria-label="Loading settings" className="flex flex-col gap-3">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
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
      {data && (
        <>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex min-w-0 flex-col gap-1 rounded-sm border border-slate-200 bg-slate-50 p-4">
              <dt className="text-xs font-medium text-slate-600">Next code</dt>
              <dd className="break-all font-mono text-xl font-semibold text-slate-900" data-testid="next-code">
                {data.employee_code_preview}
              </dd>
              <dd className="text-xs text-slate-600">Issued to the next new employee</dd>
            </div>
            <div className="flex min-w-0 flex-col gap-1 rounded-sm border border-slate-200 bg-slate-50 p-4">
              <dt className="text-xs font-medium text-slate-600">Next number</dt>
              <dd className="text-xl font-semibold text-slate-900" data-testid="counter">
                {data.employee_code_next_number?.toLocaleString('en-US')}
              </dd>
              <dd className="text-xs text-slate-600">Counter for the next new employee</dd>
            </div>
          </dl>
          {issued && (
            <InlineAlert tone="neutral">
              Employee codes have already been issued, so the starting number can only be raised to the current
              counter or higher.
            </InlineAlert>
          )}
          <EmployeeCodeForm settings={data} />
        </>
      )}
    </Card>
  )
}
