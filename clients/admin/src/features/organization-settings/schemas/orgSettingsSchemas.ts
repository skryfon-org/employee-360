import { z } from 'zod'

export const MAX_PREFIX_LEN = 10
export const MAX_CODE_LEN = 20
export const MAX_PADDING = 10
/** Mirrors backend entity.MaxEmployeeCodeStartNumber. */
export const MAX_START_NUMBER = 9999999999
/** Mirrors the backend limit on the custom welcome email message. */
export const MAX_WELCOME_MESSAGE_LEN = 2000
const PREFIX_RE = /^[A-Z0-9]+$/

export interface OrgSettingsFormValues {
  employee_code_prefix: string
  employee_code_padding: string
  employee_code_start_number: string
}

export interface WelcomeEmailFormValues {
  /** Plain text; blank means "use the default message". */
  welcome_message: string
}

/** Normalises prefix input as typed: uppercase (spaces are kept so they are reported, not silently eaten). */
export const normalizePrefix = (v: string) => v.toUpperCase()

const intString = (label: string, min: number, max?: number) =>
  z
    .string()
    .trim()
    .min(1, `${label} is required`)
    .refine((v) => /^\d+$/.test(v), `${label} must be a whole number`)
    .refine((v) => Number(v) >= min, `${label} must be at least ${min}`)
    .refine((v) => max === undefined || Number(v) <= max, `${label} must be at most ${max}`)

/**
 * Mirrors the backend rules. `minStart` is the lowest allowed starting number: 1, or the server's
 * counter once codes have been issued.
 */
export function createOrgSettingsSchema(minStart = 1) {
  return z
    .object({
      employee_code_prefix: z
        .string()
        .transform((v) => v.trim().toUpperCase())
        .pipe(
          z
            .string()
            .min(1, 'Prefix is required')
            .max(MAX_PREFIX_LEN, `Prefix must be at most ${MAX_PREFIX_LEN} characters`)
            .refine((v) => PREFIX_RE.test(v), 'Prefix may contain only letters A-Z and digits 0-9, no spaces')
            .refine((v) => !/\d$/.test(v), 'Prefix must not end in a digit'),
        ),
      employee_code_padding: intString('Digit padding', 1, MAX_PADDING),
      employee_code_start_number: intString('Starting number', Math.max(1, minStart), MAX_START_NUMBER),
    })
    .superRefine((v, ctx) => {
      const pad = Number(v.employee_code_padding)
      const start = v.employee_code_start_number
      if (!Number.isFinite(pad) || !/^\d+$/.test(start)) return
      const len = v.employee_code_prefix.length + Math.max(pad, start.length)
      if (len > MAX_CODE_LEN) {
        ctx.addIssue({
          code: 'custom',
          path: ['employee_code_padding'],
          message: `Codes would be ${len} characters; the maximum is ${MAX_CODE_LEN}`,
        })
      }
    })
}

export type OrgSettingsSchemaOutput = z.output<ReturnType<typeof createOrgSettingsSchema>>

/** Welcome email tab: a single optional plain-text message, blank clears it back to the default. */
export const welcomeEmailSchema = z.object({
  welcome_message: z
    .string()
    .refine((v) => [...v].length <= MAX_WELCOME_MESSAGE_LEN, `Welcome message must be at most ${MAX_WELCOME_MESSAGE_LEN} characters`),
})

export type WelcomeEmailSchemaOutput = z.output<typeof welcomeEmailSchema>

/** Builds the code the next employee would get; returns '' while inputs are not usable. */
export function formatEmployeeCode(prefix: string, padding: string, number: string): string {
  const p = prefix.trim().toUpperCase()
  if (!p || !/^\d+$/.test(padding) || !/^\d+$/.test(number) || Number(number) < 1 || Number(padding) < 1) return ''
  return `${p}${number.padStart(Math.min(Number(padding), MAX_PADDING), '0')}`
}

/** Regex source for an admin-entered override: the tenant prefix followed by digits only. */
export function overrideCodeError(code: string, prefix: string): string | undefined {
  const c = code.trim()
  if (!c) return 'Employee code is required'
  if (!c.startsWith(prefix) || !/^\d+$/.test(c.slice(prefix.length))) {
    return `Code must be ${prefix} followed by digits (e.g. ${prefix}0001)`
  }
  if (c.length > MAX_CODE_LEN) return `Code must be at most ${MAX_CODE_LEN} characters`
  return undefined
}

export type FieldName = keyof OrgSettingsFormValues | keyof WelcomeEmailFormValues

const SERVER_KEY_RE = /^(employee_code_(?:prefix|padding|start_number)|welcome_message):\s*/

/** Removes the leading `employee_code_xxx:` key the backend puts on validation messages. */
export function stripServerKey(message: string): string {
  return message.replace(SERVER_KEY_RE, '')
}

/** Maps a server validation message onto the field it concerns, or undefined for a form-level error. */
export function fieldForServerError(message: string): FieldName | undefined {
  const key = SERVER_KEY_RE.exec(message.trim())?.[1]
  if (key === 'welcome_message') return key
  if (key === 'employee_code_prefix' || key === 'employee_code_padding' || key === 'employee_code_start_number') {
    return key
  }
  const m = message.toLowerCase()
  if (m.includes('welcome')) return 'welcome_message'
  if (m.includes('prefix')) return 'employee_code_prefix'
  if (m.includes('padding')) return 'employee_code_padding'
  if (m.includes('start') || m.includes('number') || m.includes('issued')) return 'employee_code_start_number'
  return undefined
}

export const WEEKDAYS = [
  { value: 0, short: 'Sun', long: 'Sunday' },
  { value: 1, short: 'Mon', long: 'Monday' },
  { value: 2, short: 'Tue', long: 'Tuesday' },
  { value: 3, short: 'Wed', long: 'Wednesday' },
  { value: 4, short: 'Thu', long: 'Thursday' },
  { value: 5, short: 'Fri', long: 'Friday' },
  { value: 6, short: 'Sat', long: 'Saturday' },
] as const

/** Weekly off days: 0=Sunday..6=Saturday, at least one and at most six distinct days. */
export const weeklyOffDaysSchema = z.object({
  weekly_off_days: z
    .array(z.number().int().min(0).max(6))
    .min(1, 'Select at least one weekly off day')
    .max(6, 'Select at most six weekly off days; at least one day must be a working day'),
})

export type WeeklyOffDaysValues = z.output<typeof weeklyOffDaysSchema>
