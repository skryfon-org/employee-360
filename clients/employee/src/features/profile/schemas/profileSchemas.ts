import { z } from 'zod'
import { postalCodeRefinement } from '@employee360/ui'

/** Backend counts characters (runes), not UTF-16 units; spread by code point to match. */
const runeLength = (s: string) => [...s].length

export const MAX_CONTACTS = 10
export const GENDERS = ['male', 'female', 'other', 'prefer_not_to_say'] as const
export const BLOOD_GROUPS = ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-'] as const
export const MARITAL_STATUSES = ['single', 'married', 'divorced', 'widowed', 'other'] as const

export const humanize = (v: string) => {
  const s = v.replaceAll('_', ' ')
  return s.charAt(0).toUpperCase() + s.slice(1)
}

/** Labels for backend `missing_fields` values. */
export const FIELD_LABELS: Record<string, string> = {
  phone: 'Phone',
  date_of_birth: 'Date of birth',
  gender: 'Gender',
  address_line1: 'Address line 1',
  city: 'City',
  country: 'Country',
  emergency_contact: 'At least one emergency contact',
}

const PHONE_RE = /^[0-9+()\-\s.]{5,30}$/
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const maxLen = (label: string, max: number) =>
  z.string().trim().refine((v) => runeLength(v) <= max, `${label} must be at most ${max} characters`)

const requiredText = (label: string, max: number) =>
  z.string().trim().min(1, `${label} is required`).pipe(maxLen(label, max))

const optionalEnum = (values: readonly string[], label: string) =>
  z.string().refine((v) => v === '' || values.includes(v), `${label} is not valid`)

/** Mandatory: phone, date_of_birth, gender, address_line1, city, country. DOB must be in the past and after 1900. */
const personalShape = {
  phone: z
    .string()
    .trim()
    .min(1, 'Phone is required')
    .refine((v) => PHONE_RE.test(v), 'Enter a valid phone number'),
  date_of_birth: z
    .string()
    .min(1, 'Date of birth is required')
    .refine((v) => /^\d{4}-\d{2}-\d{2}$/.test(v), 'Enter a valid date')
    .refine((v) => {
      const d = new Date(`${v}T00:00:00Z`)
      return Number.isNaN(d.getTime()) || (d.getUTCFullYear() >= 1900 && d.getTime() < Date.now())
    }, 'Date of birth must be in the past and after 1900'),
  gender: z.string().min(1, 'Gender is required').pipe(optionalEnum(GENDERS, 'Gender')),
  blood_group: optionalEnum(BLOOD_GROUPS, 'Blood group'),
  marital_status: optionalEnum(MARITAL_STATUSES, 'Marital status'),
  nationality: maxLen('Nationality', 100),
  personal_email: z
    .string()
    .trim()
    .refine((v) => v === '' || (runeLength(v) <= 200 && EMAIL_RE.test(v)), 'Enter a valid email address (max 200 characters)'),
}

const addressShape = {
  address_line1: requiredText('Address line 1', 255),
  address_line2: maxLen('Address line 2', 255),
  city: requiredText('City', 100),
  state: maxLen('State', 100),
  country: requiredText('Country', 100),
  pincode: maxLen('Pincode', 20),
}

/** First/last name: required, trimmed, max 100. Edited on the profile page only (not in the completion wizard). */
const nameShape = {
  first_name: requiredText('First name', 100),
  last_name: requiredText('Last name', 100),
}

export const personalSchema = z.object(personalShape)
export const personalWithNamesSchema = z.object({ ...nameShape, ...personalShape })
export const addressSchema = z.object(addressShape).superRefine(postalCodeRefinement)
export const profileSchema = z.object({ ...personalShape, ...addressShape }).superRefine(postalCodeRefinement)

export type PersonalValues = z.infer<typeof personalSchema>
export type AddressValues = z.infer<typeof addressSchema>
export type ProfileValues = z.infer<typeof profileSchema> & { first_name: string; last_name: string }

export const contactSchema = z.object({
  name: requiredText('Name', 100),
  relationship: requiredText('Relationship', 50),
  phone: z
    .string()
    .trim()
    .min(1, 'Phone is required')
    .refine((v) => PHONE_RE.test(v), 'Enter a valid phone number'),
  alternate_phone: z
    .string()
    .trim()
    .refine((v) => v === '' || PHONE_RE.test(v), 'Enter a valid phone number'),
  email: z
    .string()
    .trim()
    .refine((v) => v === '' || (runeLength(v) <= 200 && EMAIL_RE.test(v)), 'Enter a valid email address (max 200 characters)'),
  is_primary: z.boolean(),
})
export type ContactValues = z.infer<typeof contactSchema>
