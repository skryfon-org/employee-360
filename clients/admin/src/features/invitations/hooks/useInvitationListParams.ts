import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  DEFAULT_PAGE_SIZE,
  invitationListParamsSchema,
  type InvitationListParams,
} from '../schemas/invitationListSchema'

/** Filter/search/pagination state lives in the URL search params (UI state only). */
export function useInvitationListParams() {
  const [sp, setSp] = useSearchParams()
  const params = useMemo<InvitationListParams>(
    () =>
      invitationListParamsSchema.parse({
        page: sp.get('page') ?? undefined,
        page_size: sp.get('page_size') ?? undefined,
        status: sp.get('status') || undefined,
        search: sp.get('search') || undefined,
      }),
    [sp],
  )

  /** Changing filter/search/page size resets to page 1; changing only `page` does not. */
  const update = useCallback(
    (patch: Partial<InvitationListParams>) => {
      const next = { ...params, ...patch }
      if (!('page' in patch)) next.page = 1
      const out = new URLSearchParams()
      if (next.page > 1) out.set('page', String(next.page))
      if (next.page_size !== DEFAULT_PAGE_SIZE) out.set('page_size', String(next.page_size))
      if (next.status) out.set('status', next.status)
      if (next.search) out.set('search', next.search)
      setSp(out, { replace: true })
    },
    [params, setSp],
  )
  return { params, update }
}
