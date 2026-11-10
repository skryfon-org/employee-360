import { z } from 'zod'

export const NAME_MAX = 100
export const CODE_MAX = 20
export const DESCRIPTION_MAX = 500

export const leaveTypeFormSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(NAME_MAX, `Name must be at most ${NAME_MAX} characters`),
  code: z.string().trim().min(1, 'Code is required').max(CODE_MAX, `Code must be at most ${CODE_MAX} characters`),
  description: z.string().trim().max(DESCRIPTION_MAX, `Description must be at most ${DESCRIPTION_MAX} characters`),
  is_active: z.boolean(),
})

export type LeaveTypeFormValues = z.infer<typeof leaveTypeFormSchema>
