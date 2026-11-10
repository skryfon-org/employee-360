import { useEffect } from 'react'
import { getErrorMessage } from '@employee360/api-client'
import { Card, InlineAlert, PageContainer, PageHeader } from '@employee360/ui'
import { useAuditLogsQuery } from '../queries/auditLogQueries'
import { useAuditLogListParams } from '../hooks/useAuditLogListParams'
import { AuditLogsFilters } from '../components/AuditLogsFilters'
import { AuditLogsTable } from '../components/AuditLogsTable'
import { AuditLogsTableSkeleton } from '../components/AuditLogsTableSkeleton'
import { AuditLogsPagination } from '../components/AuditLogsPagination'

export default function AuditLogsPage() {
  const { params, update, clear, hasFilters } = useAuditLogListParams()
  const { data, isPending, isError, error, isPlaceholderData, refetch } = useAuditLogsQuery(params)
  const meta = data?.meta as { total_items?: number; total_pages?: number } | undefined
  const totalItems = meta?.total_items ?? data?.data.length ?? 0
  const totalPages = Math.max(meta?.total_pages ?? 1, 1)

  // A page beyond the end (stale URL) steps back instead of showing an empty page.
  useEffect(() => {
    if (data && !isPlaceholderData && data.data.length === 0 && params.page > 1) update({ page: params.page - 1 })
  }, [data, isPlaceholderData, params.page, update])

  return (
    <PageContainer>
      <PageHeader title="Audit log" description="A record of administrative changes in your organization." />
      <Card padded={false} aria-label="Audit log entries" role="region" className="overflow-hidden">
        <AuditLogsFilters params={params} hasFilters={hasFilters} onChange={update} onClear={clear} />
        {isPending && <AuditLogsTableSkeleton />}
        {isError && (
          <div className="flex flex-col items-start gap-3 p-4">
            <InlineAlert tone="error">{getErrorMessage(error, 'Could not load the audit log.')}</InlineAlert>
            <button
              type="button"
              onClick={() => void refetch()}
              className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
            >
              Retry
            </button>
          </div>
        )}
        {data && !isError && (
          <div className={isPlaceholderData ? 'opacity-60 transition-opacity' : undefined} aria-busy={isPlaceholderData}>
            <AuditLogsTable entries={data.data} filtered={hasFilters} />
          </div>
        )}
        {data && !isError && data.data.length > 0 && (
          <AuditLogsPagination
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
