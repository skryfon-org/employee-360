import { z } from 'zod'

export const ACTIVITY_CATEGORIES = ['meeting', 'training', 'internal', 'support', 'other'] as const
export const DAILY_HOURS_CAP = 24

/** Select values are `project:<id>` or `category:<name>`. */
export type EntryTarget = { project_id: string } | { activity_category: (typeof ACTIVITY_CATEGORIES)[number] }

export function parseTarget(value: string): EntryTarget | null {
  if (value.startsWith('project:') && value.length > 8) return { project_id: value.slice(8) }
  if (value.startsWith('category:')) {
    const c = value.slice(9)
    if ((ACTIVITY_CATEGORIES as readonly string[]).includes(c)) return { activity_category: c as (typeof ACTIVITY_CATEGORIES)[number] }
  }
  return null
}

/**
 * `otherHours` is the day's total excluding the entry being edited; the 24h
 * check here is a convenience, the server remains authoritative.
 */
export function buildEntrySchema(otherHours: number) {
  return z.object({
    target: z.string().refine((v) => parseTarget(v) !== null, 'Choose a project or category'),
    hours: z
      .string()
      .trim()
      .min(1, 'Enter hours')
      .refine((v) => Number.isFinite(Number(v)), 'Hours must be a number')
      .refine((v) => Number(v) > 0, 'Hours must be greater than 0')
      .refine((v) => Number(v) <= DAILY_HOURS_CAP, `Hours cannot exceed ${DAILY_HOURS_CAP}`)
      .refine((v) => otherHours + Number(v) <= DAILY_HOURS_CAP + 1e-9, `Daily total cannot exceed ${DAILY_HOURS_CAP} hours`),
    notes: z.string(),
  })
}

export type EntryFormValues = z.infer<ReturnType<typeof buildEntrySchema>>
