import { describe, expect, it } from 'vitest'
import { addressOnlySchema, contactSchema, employeeListParamsSchema, employmentSchema, personalAddressSchema } from './employeeSchemas'
import { buildPatch } from './buildPatch'

const personal = {
  first_name: 'Ada', last_name: 'Doe',
  gender: '', date_of_birth: '', blood_group: '', marital_status: '', nationality: '', phone: '', personal_email: '',
  address_line1: '', address_line2: '', city: '', state: '', country: '', pincode: '',
}

describe('pincode by country', () => {
  const msg = 'Enter a valid 6-digit PIN code'
  const issue = (v: Record<string, string>) => {
    const r = personalAddressSchema.safeParse({ ...personal, ...v })
    return r.success ? null : r.error.issues.find((i) => i.path[0] === 'pincode')?.message
  }
  it('accepts a 6-digit India PIN and an empty one', () => {
    expect(issue({ country: 'India', pincode: '682001' })).toBeNull()
    expect(issue({ country: 'India', pincode: '' })).toBeNull()
  })
  it.each(['68200', '6820011', '012345', '68A001'])('rejects India PIN %s', (pincode) => {
    expect(issue({ country: 'India', pincode })).toBe(msg)
  })
  it('re-evaluates against the selected country and also applies on the address-only schema', () => {
    expect(issue({ country: 'United States', pincode: '90210' })).toBeNull()
    expect(issue({ country: 'India', pincode: '90210' })).toBe(msg)
    const { first_name: _f, last_name: _l, ...addr } = { ...personal, country: 'India', pincode: '1234' }
    void _f; void _l
    expect(addressOnlySchema.safeParse(addr).success).toBe(false)
  })
})

describe('personalAddressSchema', () => {
  it('requires trimmed first and last names, max 100', () => {
    expect(personalAddressSchema.safeParse({ ...personal, first_name: '  ' }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, last_name: '' }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, last_name: 'x'.repeat(101) }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, first_name: 'x'.repeat(100) }).success).toBe(true)
  })
  it('accepts an otherwise empty form (admins may correct partially)', () => {
    expect(personalAddressSchema.safeParse(personal).success).toBe(true)
  })
  it('rejects bad phone, email and out-of-range dates of birth', () => {
    expect(personalAddressSchema.safeParse({ ...personal, phone: 'abc' }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, personal_email: 'nope' }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, date_of_birth: '1899-12-31' }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, date_of_birth: '2999-01-01' }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, date_of_birth: '1990-05-01', phone: '+1 (555) 123-4567' }).success).toBe(true)
  })
  it('enforces length limits', () => {
    expect(personalAddressSchema.safeParse({ ...personal, pincode: '1'.repeat(21) }).success).toBe(false)
    expect(personalAddressSchema.safeParse({ ...personal, address_line1: 'a'.repeat(256) }).success).toBe(false)
  })
})

describe('employmentSchema', () => {
  const base = {
    employee_code: 'E-1', employment_type: 'full_time', employment_status: 'active',
    joining_date: '2024-01-10', probation_end_date: '', confirmation_date: '', exit_date: '', work_location: '',
    department_id: '', position_id: '', manager_id: '', effective_date: '', change_reason: '',
  }
  it('validates code format', () => {
    expect(employmentSchema.safeParse(base).success).toBe(true)
    expect(employmentSchema.safeParse({ ...base, employee_code: '-bad code' }).success).toBe(false)
  })
  it('rejects dates before the joining date', () => {
    const r = employmentSchema.safeParse({ ...base, exit_date: '2023-01-01' })
    expect(r.success).toBe(false)
  })
})

describe('contactSchema', () => {
  it('requires name, relationship and a valid phone', () => {
    expect(contactSchema.safeParse({ name: '', relationship: '', phone: '', alternate_phone: '', email: '', is_primary: false }).success).toBe(false)
    expect(contactSchema.safeParse({ name: 'Bo', relationship: 'Sibling', phone: '5551234', alternate_phone: '', email: 'b@x.com', is_primary: true }).success).toBe(true)
  })
})

describe('employeeListParamsSchema', () => {
  it('falls back to defaults for invalid values', () => {
    const p = employeeListParamsSchema.parse({ page: 'x', status: 'weird', employment_status: 'nope' })
    expect(p).toMatchObject({ page: 1, page_size: 20, search: '' })
    expect(p.status).toBeUndefined()
    expect(p.employment_status).toBeUndefined()
  })
})

describe('buildPatch', () => {
  it('sends only dirty fields and nulls cleared nullable ones', () => {
    expect(buildPatch({ a: ' x ', b: '', c: 'keep' }, { a: true, b: true }, ['b'])).toEqual({ a: 'x', b: null })
  })
})
