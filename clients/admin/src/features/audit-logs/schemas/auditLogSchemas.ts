import { z } from 'zod'

export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
export const DEFAULT_PAGE_SIZE = 20

/** entity_type values written by the backend audit writers. */
export const ENTITY_TYPES = [
  { value: 'department', label: 'Department' },
  { value: 'position', label: 'Position' },
  { value: 'user_invitation', label: 'Invitation' },
  { value: 'tenant', label: 'Organization' },
  { value: 'tenant_domain', label: 'Organization domain' },
] as const

/** Action groups; a trailing "." makes the backend treat the value as a prefix. */
export const ACTION_GROUPS = [
  { value: 'department.', label: 'Department actions' },
  { value: 'position.', label: 'Position actions' },
  { value: 'invitation.', label: 'Invitation actions' },
  { value: 'tenant.', label: 'Organization actions' },
] as const

const ACTION_LABELS: Record<string, string> = {
  'department.create': 'Created department',
  'department.update': 'Updated department',
  'department.delete': 'Deleted department',
  'department.activate': 'Activated department',
  'department.deactivate': 'Deactivated department',
  'position.create': 'Created position',
  'position.update': 'Updated position',
  'position.delete': 'Deleted position',
  'position.activate': 'Activated position',
  'position.deactivate': 'Deactivated position',
  'invitation.invite': 'Invited user',
  'invitation.resend': 'Resent invitation',
  'invitation.revoke': 'Revoked invitation',
  'tenant.rename': 'Renamed organization',
  'tenant.domain.add': 'Added organization domain',
  'tenant.domain.update': 'Updated organization domain',
  'tenant.domain.remove': 'Removed organization domain',
}

const capitalise = (s: string) => (s ? s[0].toUpperCase() + s.slice(1) : s)

/** Human label for a raw action; unknown actions fall back to a readable form of the raw string. */
export function actionLabel(action?: string): string {
  if (!action) return '-'
  return ACTION_LABELS[action] ?? capitalise(action.replace(/[._]/g, ' '))
}

export function entityTypeLabel(type?: string): string {
  if (!type) return '-'
  return ENTITY_TYPES.find((e) => e.value === type)?.label ?? capitalise(type.replace(/_/g, ' '))
}

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/
const isRealDate = (v: string) => {
  if (!DATE_RE.test(v)) return false
  const d = new Date(`${v}T00:00:00`)
  return !Number.isNaN(d.getTime()) && d.toISOString().length > 0 && v === localYmd(d)
}
function localYmd(d: Date) {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

/** Start (local midnight) of a yyyy-mm-dd date as RFC3339 UTC. */
export const startOfDayRfc3339 = (ymd: string) =>
  new Date(`${ymd}T00:00:00`).toISOString().replace(/\.\d{3}Z$/, 'Z')
/** End (local 23:59:59) of a yyyy-mm-dd date as RFC3339 UTC. */
export const endOfDayRfc3339 = (ymd: string) =>
  new Date(`${ymd}T23:59:59`).toISOString().replace(/\.\d{3}Z$/, 'Z')

const optionalDate = z
  .string()
  .trim()
  .refine((v) => v === '' || isRealDate(v), 'Enter a valid date')

/** Date-range form: both optional; from must not be after to. */
export const dateRangeSchema = z
  .object({ from: optionalDate, to: optionalDate })
  .refine((v) => !v.from || !v.to || v.from <= v.to, {
    path: ['to'],
    message: 'End date must be on or after the start date',
  })
export type DateRangeValues = z.infer<typeof dateRangeSchema>

const knownOrUndefined = (values: readonly string[]) =>
  z.string().optional().transform((v) => (v && values.includes(v) ? v : undefined))
const dateParam = z
  .string()
  .optional()
  .transform((v) => (v && isRealDate(v) ? v : undefined))

/** Parses raw URL search params; invalid values fall back to defaults. */
export const auditLogListParamsSchema = z.object({
  page: z.coerce.number().int().min(1).catch(1),
  page_size: z.coerce.number().int().min(1).max(100).catch(DEFAULT_PAGE_SIZE),
  entity_type: knownOrUndefined(ENTITY_TYPES.map((e) => e.value)),
  action: knownOrUndefined(ACTION_GROUPS.map((a) => a.value)),
  from: dateParam,
  to: dateParam,
})
export type AuditLogListParams = z.infer<typeof auditLogListParamsSchema>
