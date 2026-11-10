import { useEffect } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { Card, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import { useEmployeesQuery } from '../queries/employeeQueries'
import { useEmployeeListParams } from '../hooks/useEmployeeListParams'
import { EmployeesTable } from '../components/EmployeesTable'
import { EmployeesTableSkeleton } from '../components/EmployeesTableSkeleton'
import { EmployeesPagination } from '../components/EmployeesPagination'
import { EmployeesFilters } from '../components/EmployeesFilters'

export default function EmployeesPage() {
  const { params, update } = useEmployeeListParams()
  const { data, isPending, isError, error, isPlaceholderData, refetch } = useEmployeesQuery(params)
  const meta = data?.meta
  const totalItems = meta?.total_items ?? data?.data.length ?? 0
  const totalPages = Math.max(meta?.total_pages ?? 1, 1)
  const filtered = Boolean(
    params.search || params.status || params.department_id || params.position_id || params.employment_status,
  )

  // Step back when the last page becomes empty (e.g. after filtering).
  useEffect(() => {
    if (data && !isPlaceholderData && data.data.length === 0 && params.page > 1) update({ page: params.page - 1 })
  }, [data, isPlaceholderData, params.page, update])

  return (
    <PageContainer>
      <PageHeader title="Employees" description="Search, review and manage the people in your organization." />
      <Card padded={false} aria-label="Employees list" role="region" className="overflow-hidden">
        <EmployeesFilters params={params} update={update} />
        {isPending && <EmployeesTableSkeleton />}
        {isError && (
          <div className="flex flex-col items-start gap-3 p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load employees.')}</InlineAlert>
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
            <EmployeesTable employees={data.data} filtered={filtered} />
          </div>
        )}
        {data && data.data.length > 0 && (
          <EmployeesPagination
            page={params.page}
            pageSize={params.page_size}
            totalItems={totalItems}
            totalPages={totalPages}
            onPageChange={(page) => update({ page })}
            onPageSizeChange={(page_size) => update({ page_size })}
          />
        )}
      </Card>
    </PageContainer>
  )
}
