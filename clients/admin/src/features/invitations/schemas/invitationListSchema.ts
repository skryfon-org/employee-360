import { z } from 'zod'

export const INVITATION_STATUSES = ['pending', 'accepted', 'expired', 'revoked'] as const
export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
export const DEFAULT_PAGE_SIZE = 20
export const SEARCH_MAX_LENGTH = 100

/** Parses raw URL search params; invalid values fall back to defaults. */
export const invitationListParamsSchema = z.object({
  page: z.coerce.number().int().min(1).catch(1),
  page_size: z.coerce.number().int().min(1).max(100).catch(DEFAULT_PAGE_SIZE),
  status: z.enum(INVITATION_STATUSES).optional().catch(undefined),
  search: z.string().trim().max(SEARCH_MAX_LENGTH).optional().catch(undefined),
})
export type InvitationListParams = z.infer<typeof invitationListParamsSchema>
