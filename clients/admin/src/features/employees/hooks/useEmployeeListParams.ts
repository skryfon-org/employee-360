import { useCallback, useEffect, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useEmployeeUiStore } from '../../../stores/employeeUiStore'
import { DEFAULT_PAGE_SIZE, employeeListParamsSchema, type EmployeeListParams } from '../schemas/employeeSchemas'

/** Filters and pagination live in the URL search params; the last query string is mirrored to Zustand. */
export function useEmployeeListParams() {
  const [sp, setSp] = useSearchParams()
  const setListSearch = useEmployeeUiStore((s) => s.setListSearch)
  const params = useMemo<EmployeeListParams>(
    () =>
      employeeListParamsSchema.parse({
        page: sp.get('page') ?? undefined,
        page_size: sp.get('page_size') ?? undefined,
        search: sp.get('search') ?? undefined,
        status: sp.get('status') ?? undefined,
        department_id: sp.get('department_id') ?? undefined,
        position_id: sp.get('position_id') ?? undefined,
        employment_status: sp.get('employment_status') ?? undefined,
      }),
    [sp],
  )

  const spString = sp.toString()
  useEffect(() => setListSearch(spString), [spString, setListSearch])

  /** Changing any filter or the page size resets to page 1; changing only `page` does not. */
  const update = useCallback(
    (patch: Partial<EmployeeListParams>) => {
      const next = { ...params, ...patch }
      if (!('page' in patch)) next.page = 1
      const out = new URLSearchParams()
      if (next.page > 1) out.set('page', String(next.page))
      if (next.page_size !== DEFAULT_PAGE_SIZE) out.set('page_size', String(next.page_size))
      if (next.search) out.set('search', next.search)
      if (next.status) out.set('status', next.status)
      if (next.department_id) out.set('department_id', next.department_id)
      if (next.position_id) out.set('position_id', next.position_id)
      if (next.employment_status) out.set('employment_status', next.employment_status)
      setSp(out, { replace: true })
    },
    [params, setSp],
  )
  return { params, update }
}
