import { PageContainer, PageHeader, Tabs } from '@employee360/ui'
import { Link, useSearchParams } from 'react-router-dom'
import { filtersFromParams, hasFilters, type LeaveFilters } from '../schemas/filters'
import { MyRequests } from '../components/MyRequests'
import { TeamRequests } from '../components/TeamRequests'
import { useTeamLeaveRequestsQuery } from '../queries/leaveQueries'

export default function LeavePage() {
  const [params, setParams] = useSearchParams()
  // The Team tab only exists for users that have requests addressed to them as approver.
  const probe = useTeamLeaveRequestsQuery({ pageSize: 1 })
  const hasTeam = (probe.data?.meta?.total_items ?? probe.data?.data.length ?? 0) > 0

  const filters = filtersFromParams(params)
  // Filters live in the URL next to `tab`; any change keeps the tab and drops the page (the lists restart at page 1).
  const setFilters = (f: LeaveFilters) => {
    const next = new URLSearchParams()
    if (params.get('tab') === 'team') {
      next.set('tab', 'team')
      next.set('view', 'history')
    }
    if (f.status) next.set('status', f.status)
    if (f.leaveTypeId) next.set('leave_type_id', f.leaveTypeId)
    if (f.from) next.set('from', f.from)
    if (f.to) next.set('to', f.to)
    setParams(next, { replace: true })
  }

  // History is expanded via `view=history`, or automatically when any filter is in the URL. Collapsing drops filters and page.
  const historyOpen = params.get('view') === 'history' || hasFilters(filters)
  const setHistoryOpen = (open: boolean) => setParams(open ? { tab: 'team', view: 'history' } : { tab: 'team' }, { replace: true })

  const items = [
    { value: 'mine', label: 'My requests' },
    ...(hasTeam ? [{ value: 'team', label: 'Team requests' }] : []),
  ]
  const tab = params.get('tab') === 'team' && hasTeam ? 'team' : 'mine'

  return (
    <PageContainer>
      <PageHeader
        title="Leave"
        description="Track your leave requests."
        actions={
          <Link
            to="/leave/apply"
            className="inline-flex h-9 items-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
          >
            Apply leave
          </Link>
        }
      />
      {hasTeam ? (
        <Tabs items={items} value={tab} label="Leave sections" onValueChange={(v) => setParams(v === 'mine' ? {} : { tab: v }, { replace: true })}>
          {tab === 'team' ? <TeamRequests expanded={historyOpen} onExpandedChange={setHistoryOpen} filters={filters} onFiltersChange={setFilters} /> : <MyRequests filters={filters} onFiltersChange={setFilters} />}
        </Tabs>
      ) : (
        <MyRequests filters={filters} onFiltersChange={setFilters} />
      )}
    </PageContainer>
  )
}
