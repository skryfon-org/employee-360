import { z } from 'zod'

export const PROJECT_STATUSES = ['active', 'on_hold', 'completed', 'archived'] as const
export type ProjectStatus = (typeof PROJECT_STATUSES)[number]

export const PROJECT_STATUS_LABELS: Record<ProjectStatus, string> = {
  active: 'Active',
  on_hold: 'On hold',
  completed: 'Completed',
  archived: 'Archived',
}

const isoDate = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Enter a valid date')

export const projectFormSchema = z
  .object({
    name: z.string().trim().min(1, 'Name is required'),
    /** Optional on create (the server generates it from the name); edit mode enforces presence in the form. */
    code: z.string().trim(),
    description: z.string().trim(),
    client_name: z.string().trim(),
    status: z.enum(PROJECT_STATUSES),
    start_date: z.string().min(1, 'Start date is required').pipe(isoDate),
    end_date: z.string(),
  })
  .superRefine((v, ctx) => {
    if (v.end_date && v.start_date && v.end_date < v.start_date) {
      ctx.addIssue({ code: 'custom', path: ['end_date'], message: 'End date must be on or after the start date' })
    }
  })
export type ProjectFormValues = z.infer<typeof projectFormSchema>

export interface ProjectWindow {
  start_date?: string
  end_date?: string | null
}

/** Mirrors backend validateWithinProject; pass the project's dates to enforce its window. */
export function buildAllocationFormSchema(project?: ProjectWindow) {
  return z
    .object({
      user_id: z.string().min(1, 'Select an employee'),
      allocation_percent: z
        .string()
        .min(1, 'Allocation is required')
        .refine((v) => /^\d+$/.test(v) && Number(v) >= 1 && Number(v) <= 100, 'Enter a whole number from 1 to 100'),
      role_on_project: z.string().trim(),
      start_date: z.string().min(1, 'Start date is required').pipe(isoDate),
      end_date: z.string(),
    })
    .superRefine((v, ctx) => {
      if (v.end_date && v.start_date && v.end_date < v.start_date) {
        ctx.addIssue({ code: 'custom', path: ['end_date'], message: 'End date must be on or after the start date' })
      }
      if (!project || !v.start_date) return
      if (project.start_date && v.start_date < project.start_date) {
        ctx.addIssue({ code: 'custom', path: ['start_date'], message: 'Start date must not be before the project start date' })
      } else if (project.end_date && v.start_date > project.end_date) {
        ctx.addIssue({ code: 'custom', path: ['start_date'], message: 'Start date must not be after the project end date' })
      }
      if (project.end_date && v.end_date && v.end_date > project.end_date) {
        ctx.addIssue({ code: 'custom', path: ['end_date'], message: 'End date must not be after the project end date' })
      }
    })
}
export const allocationFormSchema = buildAllocationFormSchema()
export type AllocationFormValues = z.infer<typeof allocationFormSchema>

export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
export const DEFAULT_PAGE_SIZE = 20

export const projectListParamsSchema = z.object({
  page: z.coerce.number().int().min(1).catch(1),
  page_size: z.coerce.number().int().min(1).max(100).catch(DEFAULT_PAGE_SIZE),
  search: z.string().trim().catch('').default(''),
  status: z.enum(PROJECT_STATUSES).optional().catch(undefined),
})
export type ProjectListParams = z.infer<typeof projectListParamsSchema>
