import { z } from 'zod'

export const MIN_YEAR = 2000
export const MAX_YEAR = 2100

export const WEEK_DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'] as const

export const settingsFormSchema = z.object({
  default_view: z.enum(['calendar', 'list']),
  /** Empty string means "current year" (the field is omitted from the request). */
  default_year: z
    .string()
    .trim()
    .refine((v) => v === '' || (/^\d{4}$/.test(v) && Number(v) >= MIN_YEAR && Number(v) <= MAX_YEAR), `Enter a year between ${MIN_YEAR} and ${MAX_YEAR}, or leave blank`),
  week_start_day: z.enum(['0', '1', '2', '3', '4', '5', '6']),
})

export type SettingsFormValues = z.infer<typeof settingsFormSchema>
