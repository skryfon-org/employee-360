import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert } from '@employee360/ui'
import { useState } from 'react'
import { useTeamLeaveRequestsQuery, type LeaveRequest, type LeaveStatus } from '../queries/leaveQueries'
import { DecisionDialog } from './DecisionDialog'
import { EMPTY_FILTERS, filterKey, hasFilters, type LeaveFilters } from '../schemas/filters'
import { LeaveFilterBar } from './LeaveFilterBar'
import { LeaveRequestsTable } from './LeaveRequestsTable'
import { ROW_BTN } from './styles'
import { Pager } from './Pager'

function Section({ title, heading, filtered, status, excludeStatus, leaveTypeId, from, to, actionable, onClear }: { title: string; heading?: boolean; filtered?: boolean; status?: LeaveStatus; excludeStatus?: LeaveStatus; leaveTypeId?: string; from?: string; to?: string; actionable?: boolean; onClear?: () => void }) {
  const [page, setPage] = useState(1)
  const [target, setTarget] = useState<{ request: LeaveRequest; decision: 'approve' | 'reject' } | null>(null)
  const { data, isPending, isError, error } = useTeamLeaveRequestsQuery({ status, excludeStatus, leaveTypeId, from, to, page })
  const rows = data?.data ?? []

  return (
    <section aria-label={title} className="flex flex-col gap-3">
      {heading && (
        <h2 className="flex items-center gap-2 text-base font-semibold text-slate-900">
          {title}
          {!!data?.meta?.total_items && <span className="rounded-sm bg-slate-900 px-1.5 py-0.5 text-xs font-semibold text-white">{data.meta.total_items}</span>}
        </h2>
      )}
      {isPending ? (
        <p role="status" className="text-sm text-slate-600">
          Loading…
        </p>
      ) : isError ? (
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load team requests.')}</InlineAlert>
      ) : rows.length === 0 ? (
        <div className="rounded-sm border border-slate-200 bg-white p-4 text-sm text-slate-600">
          {filtered ? (
            <>
              <p>No requests match your filters.</p>
              {onClear && (
                <button type="button" onClick={onClear} className="mt-2 h-9 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2">
                  Clear filters
                </button>
              )}
            </>
          ) : (
            <p>{actionable ? 'No pending requests.' : 'No past decisions yet.'}</p>
          )}
        </div>
      ) : (
        <LeaveRequestsTable
          rows={rows}
          variant="team"
          actions={
            actionable
              ? (r) => (
                  <div className="flex gap-2">
                    <button
                      type="button"
                      aria-label={`Approve request from ${r.user_name || r.user_email}`}
                      onClick={() => setTarget({ request: r, decision: 'approve' })}
                      className={`${ROW_BTN} border-slate-300 text-slate-800 hover:bg-slate-100 focus-visible:ring-slate-900`}
                    >
                      Approve
                    </button>
                    <button
                      type="button"
                      aria-label={`Reject request from ${r.user_name || r.user_email}`}
                      onClick={() => setTarget({ request: r, decision: 'reject' })}
                      className={`${ROW_BTN} border-red-300 text-red-600 hover:bg-red-50 focus-visible:ring-red-600`}
                    >
                      Reject
                    </button>
                  </div>
                )
              : undefined
          }
        />
      )}
      <Pager page={page} totalPages={data?.meta?.total_pages ?? 1} onChange={setPage} />
      {target && <DecisionDialog request={target.request} decision={target.decision} onClose={() => setTarget(null)} />}
    </section>
  )
}

const HISTORY_ID = 'team-history'

export function TeamRequests({ expanded, onExpandedChange, filters, onFiltersChange }: { expanded: boolean; onExpandedChange: (open: boolean) => void; filters: LeaveFilters; onFiltersChange: (f: LeaveFilters) => void }) {
  return (
    <div className="flex flex-col gap-8">
      <Section title="Pending approval" heading status="pending" actionable />
      <div className="flex flex-col gap-4">
        <h2 className="text-base font-semibold text-slate-900">
          <button
            type="button"
            aria-expanded={expanded}
            aria-controls={HISTORY_ID}
            onClick={() => onExpandedChange(!expanded)}
            className="inline-flex min-h-11 items-center gap-2 rounded-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            <svg aria-hidden="true" viewBox="0 0 20 20" className={`h-4 w-4 fill-none stroke-current stroke-2 ${expanded ? 'rotate-90' : ''}`}>
              <path d="M7 4l6 6-6 6" />
            </svg>
            History
          </button>
        </h2>
        {expanded && (
          <div id={HISTORY_ID} className="flex flex-col gap-4">
            <LeaveFilterBar value={filters} onChange={onFiltersChange} statuses={['approved', 'rejected', 'cancelled']} />
            <Section
              key={filterKey(filters)}
              title="History"
              filtered={hasFilters(filters)}
              status={filters.status || undefined}
              // History "All" = everything no longer pending; the server excludes pending so pagination totals stay correct.
              excludeStatus={filters.status ? undefined : 'pending'}
              leaveTypeId={filters.leaveTypeId || undefined}
              from={filters.from || undefined}
              to={filters.to || undefined}
              onClear={() => onFiltersChange(EMPTY_FILTERS)}
            />
          </div>
        )}
      </div>
    </div>
  )
}
