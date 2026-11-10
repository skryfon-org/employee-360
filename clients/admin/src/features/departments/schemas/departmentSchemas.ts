import { z } from 'zod'

export const NAME_MAX = 100
export const DESCRIPTION_MAX = 500

/** Backend counts characters (runes), not UTF-16 units; spread by code point to match. */
const runeLength = (s: string) => [...s].length

export const departmentFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Name is required')
    .refine((v) => runeLength(v) <= NAME_MAX, `Name must be at most ${NAME_MAX} characters`),
  description: z
    .string()
    .trim()
    .refine((v) => runeLength(v) <= DESCRIPTION_MAX, `Description must be at most ${DESCRIPTION_MAX} characters`),
  is_active: z.boolean(),
})
export type DepartmentFormValues = z.infer<typeof departmentFormSchema>

export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
export const DEFAULT_PAGE_SIZE = 20

/** Parses raw URL search params; invalid values fall back to defaults. */
export const departmentListParamsSchema = z.object({
  page: z.coerce.number().int().min(1).catch(1),
  page_size: z.coerce.number().int().min(1).max(100).catch(DEFAULT_PAGE_SIZE),
  /** Status filter: URL carries `is_active=true|false`; absent/invalid means All. */
  is_active: z
    .enum(['true', 'false'])
    .optional()
    .catch(undefined)
    .transform((v) => (v === undefined ? undefined : v === 'true')),
})
export type DepartmentListParams = z.infer<typeof departmentListParamsSchema>
