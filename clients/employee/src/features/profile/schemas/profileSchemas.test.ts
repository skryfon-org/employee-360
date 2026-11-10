import { describe, expect, it } from 'vitest'
import { addressSchema, contactSchema, personalSchema, personalWithNamesSchema, profileSchema } from './profileSchemas'

const valid = {
  phone: '+1 (555) 123-4567',
  date_of_birth: '1990-05-01',
  gender: 'female',
  blood_group: '',
  marital_status: '',
  nationality: '',
  personal_email: '',
  address_line1: '1 Main St',
  address_line2: '',
  city: 'Pune',
  state: '',
  country: 'India',
  pincode: '',
}

describe('profileSchema', () => {
  it('accepts a valid profile with optional fields blank', () => {
    expect(profileSchema.safeParse(valid).success).toBe(true)
  })

  it.each(['phone', 'date_of_birth', 'gender', 'address_line1', 'city', 'country'] as const)('requires %s', (field) => {
    const r = profileSchema.safeParse({ ...valid, [field]: '' })
    expect(r.success).toBe(false)
  })

  it('validates phone and email formats', () => {
    expect(personalSchema.safeParse({ ...valid, phone: 'abc' }).success).toBe(false)
    expect(personalSchema.safeParse({ ...valid, phone: '1234' }).success).toBe(false)
    expect(personalSchema.safeParse({ ...valid, personal_email: 'not-an-email' }).success).toBe(false)
    expect(personalSchema.safeParse({ ...valid, personal_email: 'me@home.org' }).success).toBe(true)
  })

  it('requires a date of birth in the past and after 1900', () => {
    expect(personalSchema.safeParse({ ...valid, date_of_birth: '1899-12-31' }).success).toBe(false)
    expect(personalSchema.safeParse({ ...valid, date_of_birth: '2999-01-01' }).success).toBe(false)
    expect(personalSchema.safeParse({ ...valid, date_of_birth: '1900-01-01' }).success).toBe(true)
  })

  it('rejects unknown enum values and over-long text', () => {
    expect(personalSchema.safeParse({ ...valid, gender: 'robot' }).success).toBe(false)
    expect(personalSchema.safeParse({ ...valid, blood_group: 'Z+' }).success).toBe(false)
    expect(personalSchema.safeParse({ ...valid, nationality: 'x'.repeat(101) }).success).toBe(false)
    expect(addressSchema.safeParse({ ...valid, address_line1: 'x'.repeat(256) }).success).toBe(false)
    expect(addressSchema.safeParse({ ...valid, pincode: 'x'.repeat(21) }).success).toBe(false)
  })
})

describe('pincode by country', () => {
  const msg = 'Enter a valid 6-digit PIN code'
  const issue = (v: Record<string, string>) => {
    const r = addressSchema.safeParse({ ...valid, ...v })
    return r.success ? null : r.error.issues.find((i) => i.path[0] === 'pincode')?.message
  }
  it('accepts a 6-digit India PIN and an empty one', () => {
    expect(issue({ pincode: '682001' })).toBeNull()
    expect(issue({ pincode: '' })).toBeNull()
  })
  it.each(['68200', '6820011', '012345', '68A001'])('rejects India PIN %s', (pincode) => {
    expect(issue({ pincode })).toBe(msg)
    expect(profileSchema.safeParse({ ...valid, pincode }).success).toBe(false)
  })
  it('applies the rule of the selected country', () => {
    expect(issue({ country: 'United States', pincode: '90210-1234' })).toBeNull()
    expect(issue({ country: 'United States', pincode: '682001' })).toMatch(/ZIP/)
    expect(issue({ country: 'Narnia', pincode: 'AB1 2CD' })).toBeNull()
  })
})

describe('contactSchema', () => {
  const c = { name: 'Bo', relationship: 'Sibling', phone: '5551234', alternate_phone: '', email: '', is_primary: false }
  it('requires name, relationship and phone; limits lengths', () => {
    expect(contactSchema.safeParse(c).success).toBe(true)
    expect(contactSchema.safeParse({ ...c, name: '' }).success).toBe(false)
    expect(contactSchema.safeParse({ ...c, relationship: 'x'.repeat(51) }).success).toBe(false)
    expect(contactSchema.safeParse({ ...c, name: 'x'.repeat(101) }).success).toBe(false)
    expect(contactSchema.safeParse({ ...c, phone: 'nope' }).success).toBe(false)
    expect(contactSchema.safeParse({ ...c, alternate_phone: 'nope' }).success).toBe(false)
    expect(contactSchema.safeParse({ ...c, email: 'bad' }).success).toBe(false)
  })
})

describe('personalWithNamesSchema', () => {
  const v = { ...valid, first_name: 'Jane', last_name: 'Doe' }
  it('accepts names and rejects blank or over-long ones', () => {
    expect(personalWithNamesSchema.safeParse(v).success).toBe(true)
    expect(personalWithNamesSchema.safeParse({ ...v, first_name: '   ' }).success).toBe(false)
    expect(personalWithNamesSchema.safeParse({ ...v, last_name: 'x'.repeat(101) }).success).toBe(false)
    expect(personalWithNamesSchema.safeParse({ ...v, last_name: 'x'.repeat(100) }).success).toBe(true)
  })
})
