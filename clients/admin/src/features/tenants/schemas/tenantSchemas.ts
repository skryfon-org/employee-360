import { z } from 'zod'

export const NAME_MAX_LENGTH = 255

const nameField = z.string().trim().min(1, 'Name is required').max(NAME_MAX_LENGTH, `Name must be at most ${NAME_MAX_LENGTH} characters`)
const domainField = z
  .string()
  .trim()
  .min(1, 'Domain is required')
  .max(253, 'Domain is too long')
  .regex(/^(?!-)[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$/, 'Enter a valid domain, e.g. acme.com')

export const renameTenantSchema = z.object({ name: nameField })
export type RenameTenantFormValues = z.infer<typeof renameTenantSchema>

export const addDomainSchema = z.object({ domain: domainField })
export type AddDomainFormValues = z.infer<typeof addDomainSchema>
