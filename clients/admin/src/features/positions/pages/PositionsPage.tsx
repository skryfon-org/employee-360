import { useEffect, useState } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { FilterBar, Card, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import { usePositionsQuery } from '../queries/positionQueries'
import { usePositionListParams } from '../hooks/usePositionListParams'
import { PositionsTable } from '../components/PositionsTable'
import { PositionsTableSkeleton } from '../components/PositionsTableSkeleton'
import { PositionsPagination } from '../components/PositionsPagination'
import { PositionFormModal } from '../components/PositionFormModal'

export default function PositionsPage() {
  const { params, update } = usePositionListParams()
  const { data, isPending, isError, error, isPlaceholderData, refetch } = usePositionsQuery(params)
  const [creating, setCreating] = useState(false)
  const meta = data?.meta
  const totalItems = meta?.total_items ?? data?.data.length ?? 0
  const totalPages = Math.max(meta?.total_pages ?? 1, 1)

  // After deleting the last row of the last page, step back instead of showing an empty page.
  useEffect(() => {
    if (data && !isPlaceholderData && data.data.length === 0 && params.page > 1) update({ page: params.page - 1 })
  }, [data, isPlaceholderData, params.page, update])

  return (
    <PageContainer>
      <PageHeader
        title="Positions"
        description="Organise your people into positions."
        actions={
          <button
            type="button"
            onClick={() => setCreating(true)}
            className="inline-flex h-11 items-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
          >
            Create position
          </button>
        }
      />
      <Card padded={false} aria-label="Positions list" role="region" className="overflow-hidden">
        <FilterBar>
          <div className="flex flex-col gap-1 sm:w-48">
            <label htmlFor="position-status" className="text-xs font-medium text-slate-900">
              Status
            </label>
            <select
              id="position-status"
              value={params.is_active === undefined ? '' : String(params.is_active)}
              onChange={(e) => update({ is_active: e.target.value === '' ? undefined : e.target.value === 'true' })}
              className="h-11 rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 sm:text-sm md:h-9"
            >
              <option value="">All</option>
              <option value="true">Active</option>
              <option value="false">Inactive</option>
            </select>
          </div>
        </FilterBar>
        {isPending && <PositionsTableSkeleton />}
        {isError && (
          <div className="flex flex-col items-start gap-3 p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load positions.')}</InlineAlert>
            <button
              type="button"
              onClick={() => void refetch()}
              className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
            >
              Retry
            </button>
          </div>
        )}
        {data && (
          <div className={isPlaceholderData ? 'opacity-60 transition-opacity' : undefined} aria-busy={isPlaceholderData}>
            <PositionsTable positions={data.data} filtered={params.is_active !== undefined} />
          </div>
        )}
        {data && data.data.length > 0 && (
          <PositionsPagination
            page={params.page}
            pageSize={params.page_size}
            totalItems={totalItems}
            totalPages={totalPages}
            onPageChange={(page) => update({ page })}
            onPageSizeChange={(page_size) => update({ page_size })}
          />
        )}
      </Card>
      {creating && <PositionFormModal onClose={() => setCreating(false)} />}
    </PageContainer>
  )
}
