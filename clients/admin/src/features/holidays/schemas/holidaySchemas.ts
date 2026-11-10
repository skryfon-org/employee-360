import { z } from 'zod'

export const NAME_MAX = 100
export const DESCRIPTION_MAX = 500

/** True for a real calendar date written `YYYY-MM-DD` (rejects e.g. 2026-02-30). */
export function isValidIsoDate(value: string): boolean {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (!m) return false
  const [y, mo, d] = [Number(m[1]), Number(m[2]), Number(m[3])]
  const dt = new Date(Date.UTC(y, mo - 1, d))
  return dt.getUTCFullYear() === y && dt.getUTCMonth() === mo - 1 && dt.getUTCDate() === d
}

export const holidayFormSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(NAME_MAX, `Name must be at most ${NAME_MAX} characters`),
  date: z
    .string()
    .min(1, 'Date is required')
    .refine(isValidIsoDate, 'Enter a valid date'),
  category_id: z.string().min(1, 'Category is required'),
  description: z.string().trim().max(DESCRIPTION_MAX, `Description must be at most ${DESCRIPTION_MAX} characters`),
})

export type HolidayFormValues = z.infer<typeof holidayFormSchema>
