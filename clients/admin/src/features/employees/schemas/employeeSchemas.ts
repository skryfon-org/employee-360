import { z } from 'zod'
import { postalCodeRefinement } from '@employee360/ui'

/** Backend counts characters (runes), not UTF-16 units; spread by code point to match. */
const runeLength = (s: string) => [...s].length

export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const
export const DEFAULT_PAGE_SIZE = 20
export const MAX_CONTACTS = 10

export const GENDERS = ['male', 'female', 'other', 'prefer_not_to_say'] as const
export const BLOOD_GROUPS = ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-'] as const
export const MARITAL_STATUSES = ['single', 'married', 'divorced', 'widowed', 'other'] as const
export const EMPLOYMENT_TYPES = ['full_time', 'part_time', 'contract', 'intern', 'consultant'] as const
export const EMPLOYMENT_STATUSES = ['active', 'probation', 'on_leave', 'notice_period', 'resigned', 'terminated'] as const

/** Option label: "full_time" -> "Full time". */
export const humanize = (v: string) => {
  const s = v.replaceAll('_', ' ')
  return s.charAt(0).toUpperCase() + s.slice(1)
}

// ---- list params (URL search params) ----

export const employeeListParamsSchema = z.object({
  page: z.coerce.number().int().min(1).catch(1),
  page_size: z.coerce.number().int().min(1).max(100).catch(DEFAULT_PAGE_SIZE),
  search: z.string().trim().catch('').default(''),
  status: z.enum(['active', 'inactive']).optional().catch(undefined),
  department_id: z.string().optional().catch(undefined),
  position_id: z.string().optional().catch(undefined),
  employment_status: z.enum(EMPLOYMENT_STATUSES).optional().catch(undefined),
})
export type EmployeeListParams = z.infer<typeof employeeListParamsSchema>

// ---- shared validators (mirror backend rules) ----

const PHONE_RE = /^[0-9+()\-\s.]{5,30}$/
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const CODE_RE = /^[A-Za-z0-9][A-Za-z0-9._-]{0,29}$/

const maxLen = (label: string, max: number) =>
  z.string().trim().refine((v) => runeLength(v) <= max, `${label} must be at most ${max} characters`)

const optionalPhone = (label: string) =>
  z
    .string()
    .trim()
    .refine((v) => v === '' || (runeLength(v) <= 30 && PHONE_RE.test(v)), `${label} is not a valid phone number`)

const optionalEnum = <T extends readonly [string, ...string[]]>(values: T, label: string) =>
  z.string().refine((v) => v === '' || (values as readonly string[]).includes(v), `${label} is not valid`)

const optionalDate = z.string().refine((v) => v === '' || /^\d{4}-\d{2}-\d{2}$/.test(v), 'Enter a valid date')

/** DOB must be in the past and after 1900. */
const optionalDob = optionalDate.refine((v) => {
  if (v === '') return true
  const d = new Date(`${v}T00:00:00Z`)
  return d.getUTCFullYear() >= 1900 && d.getTime() < Date.now()
}, 'Date of birth must be in the past and after 1900')

// ---- personal + address (admin may correct these) ----

const personalAddressShape = z.object({
  first_name: z.string().trim().min(1, 'First name is required').pipe(maxLen('First name', 100)),
  last_name: z.string().trim().min(1, 'Last name is required').pipe(maxLen('Last name', 100)),
  gender: optionalEnum(GENDERS, 'Gender'),
  date_of_birth: optionalDob,
  blood_group: optionalEnum(BLOOD_GROUPS, 'Blood group'),
  marital_status: optionalEnum(MARITAL_STATUSES, 'Marital status'),
  nationality: maxLen('Nationality', 100),
  phone: optionalPhone('Phone'),
  personal_email: z
    .string()
    .trim()
    .refine((v) => v === '' || (runeLength(v) <= 200 && EMAIL_RE.test(v)), 'Enter a valid email address (max 200 characters)'),
  address_line1: maxLen('Address line 1', 255),
  address_line2: maxLen('Address line 2', 255),
  city: maxLen('City', 100),
  state: maxLen('State', 100),
  country: maxLen('Country', 100),
  pincode: maxLen('Pincode', 20),
})
/** Pincode is validated against the selected country's postal code rule. */
export const personalAddressSchema = personalAddressShape.superRefine(postalCodeRefinement)
/** The address sub-page does not edit names, so it must not validate them. */
export const addressOnlySchema = personalAddressShape.omit({ first_name: true, last_name: true }).superRefine(postalCodeRefinement)
export type PersonalAddressValues = z.infer<typeof personalAddressSchema>

// ---- employment (admin only) ----

export const employmentSchema = z
  .object({
    employee_code: z
      .string()
      .trim()
      .refine((v) => v === '' || CODE_RE.test(v), "Use letters, digits, '.', '_' or '-' (max 30 characters)"),
    employment_type: optionalEnum(EMPLOYMENT_TYPES, 'Employment type'),
    employment_status: z.enum(EMPLOYMENT_STATUSES, 'Employment status is required'),
    joining_date: optionalDate,
    probation_end_date: optionalDate,
    confirmation_date: optionalDate,
    exit_date: optionalDate,
    work_location: maxLen('Work location', 150),
    department_id: z.string(),
    position_id: z.string(),
    manager_id: z.string(),
    effective_date: optionalDate,
    change_reason: maxLen('Change reason', 200),
  })
  .superRefine((v, ctx) => {
    if (!v.joining_date) return
    for (const f of ['probation_end_date', 'confirmation_date', 'exit_date'] as const) {
      if (v[f] && v[f] < v.joining_date) {
        ctx.addIssue({ code: 'custom', path: [f], message: 'Must not be before the joining date' })
      }
    }
  })
export type EmploymentValues = z.infer<typeof employmentSchema>

// ---- emergency contact ----

export const contactSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').pipe(maxLen('Name', 100)),
  relationship: z.string().trim().min(1, 'Relationship is required').pipe(maxLen('Relationship', 50)),
  phone: z
    .string()
    .trim()
    .min(1, 'Phone is required')
    .refine((v) => PHONE_RE.test(v), 'Enter a valid phone number'),
  alternate_phone: optionalPhone('Alternate phone'),
  email: z
    .string()
    .trim()
    .refine((v) => v === '' || (runeLength(v) <= 200 && EMAIL_RE.test(v)), 'Enter a valid email address (max 200 characters)'),
  is_primary: z.boolean(),
})
export type ContactValues = z.infer<typeof contactSchema>
