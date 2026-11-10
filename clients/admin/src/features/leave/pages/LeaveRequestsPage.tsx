import { useMemo } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { FilterBar, Card, PageContainer, PageHeader } from '@employee360/ui'
import { LeaveFilterBar } from '../components/LeaveFilterBar'
import { LeaveRequestsList } from '../components/LeaveRequestsList'
import { SECONDARY_BTN } from '../components/styles'
import { DEFAULT_FILTERS, filtersFromParams, filtersToParams, type LeaveFilters } from '../schemas/filters'

export default function LeaveRequestsPage() {
  const [sp, setSp] = useSearchParams()
  const filters = useMemo(() => filtersFromParams(sp), [sp])
  const update = (patch: Partial<LeaveFilters>) => setSp(filtersToParams({ ...filters, ...patch }), { replace: true })

  return (
    <PageContainer>
      <PageHeader
        title="Leave requests"
        description="Review and decide leave requests across the organization."
        actions={
          <Link to="/leave/types" className={SECONDARY_BTN}>
            Leave types
          </Link>
        }
      />
      <Card padded={false} aria-label="Leave requests" role="region" className="overflow-hidden">
        <FilterBar>
          <LeaveFilterBar value={filters} onChange={update} />
        </FilterBar>
        <LeaveRequestsList filters={filters} onPage={(page) => update({ page })} onReset={() => update({ ...DEFAULT_FILTERS })} />
      </Card>
    </PageContainer>
  )
}
