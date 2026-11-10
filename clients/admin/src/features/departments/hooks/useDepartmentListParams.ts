import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { DEFAULT_PAGE_SIZE, departmentListParamsSchema, type DepartmentListParams } from '../schemas/departmentSchemas'

/** Pagination state lives in the URL search params (UI state only). */
export function useDepartmentListParams() {
  const [sp, setSp] = useSearchParams()
  const params = useMemo<DepartmentListParams>(
    () =>
      departmentListParamsSchema.parse({
        page: sp.get('page') ?? undefined,
        page_size: sp.get('page_size') ?? undefined,
        is_active: sp.get('is_active') ?? undefined,
      }),
    [sp],
  )

  /** Changing page size or the status filter resets to page 1; changing only `page` does not. */
  const update = useCallback(
    (patch: Partial<DepartmentListParams>) => {
      const next = { ...params, ...patch }
      if (!('page' in patch)) next.page = 1
      const out = new URLSearchParams()
      if (next.page > 1) out.set('page', String(next.page))
      if (next.page_size !== DEFAULT_PAGE_SIZE) out.set('page_size', String(next.page_size))
      if (next.is_active !== undefined) out.set('is_active', String(next.is_active))
      setSp(out, { replace: true })
    },
    [params, setSp],
  )
  return { params, update }
}
