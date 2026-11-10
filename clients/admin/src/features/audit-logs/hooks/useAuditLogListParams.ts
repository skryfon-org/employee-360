import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { auditLogListParamsSchema, DEFAULT_PAGE_SIZE, type AuditLogListParams } from '../schemas/auditLogSchemas'

const FILTER_KEYS = ['entity_type', 'action', 'from', 'to'] as const

/** Pagination and filters live in the URL search params (UI state only). */
export function useAuditLogListParams() {
  const [sp, setSp] = useSearchParams()
  const params = useMemo<AuditLogListParams>(
    () =>
      auditLogListParamsSchema.parse({
        page: sp.get('page') ?? undefined,
        page_size: sp.get('page_size') ?? undefined,
        entity_type: sp.get('entity_type') ?? undefined,
        action: sp.get('action') ?? undefined,
        from: sp.get('from') ?? undefined,
        to: sp.get('to') ?? undefined,
      }),
    [sp],
  )

  /** Any change other than `page` resets to page 1. */
  const update = useCallback(
    (patch: Partial<AuditLogListParams>) => {
      const next = { ...params, ...patch }
      if (!('page' in patch)) next.page = 1
      const out = new URLSearchParams()
      if (next.page > 1) out.set('page', String(next.page))
      if (next.page_size !== DEFAULT_PAGE_SIZE) out.set('page_size', String(next.page_size))
      for (const k of FILTER_KEYS) if (next[k]) out.set(k, next[k])
      setSp(out, { replace: true })
    },
    [params, setSp],
  )
  const clear = useCallback(() => update({ entity_type: undefined, action: undefined, from: undefined, to: undefined }), [update])
  const hasFilters = FILTER_KEYS.some((k) => params[k] !== undefined)
  return { params, update, clear, hasFilters }
}
