import { z } from 'zod'

const optionalUuid = z
  .string()
  .trim()
  .refine((v) => v === '' || z.string().uuid().safeParse(v).success, 'Enter a valid ID')

export const inviteUserSchema = z.object({
  email: z.string().trim().min(1, 'Email is required').email('Enter a valid email address'),
  roleId: z.string().min(1, 'Select a role'),
  firstName: z.string().trim(),
  lastName: z.string().trim(),
  departmentId: optionalUuid,
  positionId: optionalUuid,
})
export type InviteUserFormValues = z.infer<typeof inviteUserSchema>
