import { z } from 'zod'
import { DESCRIPTION_MAX, NAME_MAX } from './holidaySchemas'

export const HEX_COLOR = /^#[0-9A-Fa-f]{6}$/

export const categoryFormSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(NAME_MAX, `Name must be at most ${NAME_MAX} characters`),
  color: z.string().regex(HEX_COLOR, 'Enter a colour as #RRGGBB'),
  description: z.string().trim().max(DESCRIPTION_MAX, `Description must be at most ${DESCRIPTION_MAX} characters`),
  is_active: z.boolean(),
  is_public: z.boolean(),
})

export type CategoryFormValues = z.infer<typeof categoryFormSchema>
