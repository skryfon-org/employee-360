import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { DEFAULT_PAGE_SIZE, projectListParamsSchema, type ProjectListParams } from '../schemas/projectSchemas'

/** Search, status filter and pagination live in the URL search params (UI state only). */
export function useProjectListParams() {
  const [sp, setSp] = useSearchParams()
  const params = useMemo<ProjectListParams>(
    () =>
      projectListParamsSchema.parse({
        page: sp.get('page') ?? undefined,
        page_size: sp.get('page_size') ?? undefined,
        search: sp.get('search') ?? undefined,
        status: sp.get('status') ?? undefined,
      }),
    [sp],
  )

  /** Changing any filter or the page size resets to page 1; changing only `page` does not. */
  const update = useCallback(
    (patch: Partial<ProjectListParams>) => {
      const next = { ...params, ...patch }
      if (!('page' in patch)) next.page = 1
      const out = new URLSearchParams()
      if (next.page > 1) out.set('page', String(next.page))
      if (next.page_size !== DEFAULT_PAGE_SIZE) out.set('page_size', String(next.page_size))
      if (next.search) out.set('search', next.search)
      if (next.status) out.set('status', next.status)
      setSp(out, { replace: true })
    },
    [params, setSp],
  )
  return { params, update }
}
