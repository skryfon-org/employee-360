import { describe, expect, it } from 'vitest'
import {
  createOrgSettingsSchema,
  fieldForServerError,
  formatEmployeeCode,
  overrideCodeError,
  stripServerKey,
  welcomeEmailSchema,
} from './orgSettingsSchemas'

const ok = { employee_code_prefix: 'SKRYFON', employee_code_padding: '4', employee_code_start_number: '40001' }
const msg = (v: Record<string, string>, min = 1) => {
  const r = createOrgSettingsSchema(min).safeParse({ ...ok, ...v })
  return r.success ? undefined : r.error.issues[0].message
}

describe('createOrgSettingsSchema', () => {
  it('accepts a valid format and uppercases/trims the prefix', () => {
    const r = createOrgSettingsSchema().safeParse({ ...ok, employee_code_prefix: ' skryfon ' })
    expect(r.success && r.data.employee_code_prefix).toBe('SKRYFON')
  })
  it('rejects bad prefixes', () => {
    expect(msg({ employee_code_prefix: '' })).toMatch(/required/)
    expect(msg({ employee_code_prefix: 'ABCDEFGHIJK' })).toMatch(/at most 10/)
    expect(msg({ employee_code_prefix: 'AB CD' })).toMatch(/no spaces/)
    expect(msg({ employee_code_prefix: 'AB-C' })).toMatch(/only letters/)
    expect(msg({ employee_code_prefix: 'EMP1' })).toMatch(/end in a digit/)
  })
  it('validates padding 1-10', () => {
    expect(msg({ employee_code_padding: '0' })).toMatch(/at least 1/)
    expect(msg({ employee_code_padding: '11' })).toMatch(/at most 10/)
    expect(msg({ employee_code_padding: '2.5' })).toMatch(/whole number/)
  })
  it('validates start number >= 1 integer and the issued minimum', () => {
    expect(msg({ employee_code_start_number: '0' })).toMatch(/at least 1/)
    expect(msg({ employee_code_start_number: '1.5' })).toMatch(/whole number/)
    expect(msg({ employee_code_start_number: '7' }, 10)).toMatch(/at least 10/)
    expect(msg({ employee_code_start_number: '10' }, 10)).toBeUndefined()
  })
  it('caps total length at 20', () => {
    expect(msg({ employee_code_prefix: 'ABCDEFGHIJ', employee_code_padding: '10' })).toBeUndefined()
    expect(msg({ employee_code_prefix: 'ABCDEFGHIJ', employee_code_padding: '10', employee_code_start_number: '12345678901' })).toMatch(/at most 9999999999/)
  })
})

describe('welcome message', () => {
  const wm = (v?: string) => welcomeEmailSchema.safeParse({ welcome_message: v })
  it('accepts blank and up to 2000 characters', () => {
    expect(wm('').success).toBe(true)
    expect(wm('a'.repeat(2000)).success).toBe(true)
  })
  it('rejects more than 2000 characters', () => {
    const r = wm('a'.repeat(2001))
    expect(!r.success && r.error.issues[0].message).toMatch(/at most 2000/)
  })
  it('maps the server error to the welcome_message field', () => {
    expect(fieldForServerError('welcome_message: must not exceed 2000 characters')).toBe('welcome_message')
    expect(stripServerKey('welcome_message: must not exceed 2000 characters')).toBe('must not exceed 2000 characters')
  })
})

describe('helpers', () => {
  it('formats codes with padding and overflow', () => {
    expect(formatEmployeeCode('skryfon', '4', '40001')).toBe('SKRYFON40001')
    expect(formatEmployeeCode('EMP', '4', '7')).toBe('EMP0007')
    expect(formatEmployeeCode('', '4', '7')).toBe('')
    expect(formatEmployeeCode('EMP', 'x', '7')).toBe('')
  })
  it('validates override codes against the prefix', () => {
    expect(overrideCodeError('EMP0042', 'EMP')).toBeUndefined()
    expect(overrideCodeError('ABC0042', 'EMP')).toMatch(/EMP followed by digits/)
    expect(overrideCodeError('EMP', 'EMP')).toBeDefined()
  })
  it('maps exact backend messages to fields by their key prefix', () => {
    expect(fieldForServerError('employee_code_prefix: must not end in a digit')).toBe('employee_code_prefix')
    expect(fieldForServerError('employee_code_padding: must be between 1 and 10')).toBe('employee_code_padding')
    expect(fieldForServerError('employee_code_start_number: must be a whole number between 1 and 9999999999')).toBe('employee_code_start_number')
    expect(fieldForServerError('employee_code_padding: prefix length plus padding must not exceed 20 characters')).toBe('employee_code_padding')
    expect(
      fieldForServerError('employee_code_start_number: employee codes have already been issued; the start number must be 12 or higher'),
    ).toBe('employee_code_start_number')
  })
  it('falls back to substrings without a key and strips the key for display', () => {
    expect(fieldForServerError('padding must be between 1 and 10')).toBe('employee_code_padding')
    expect(fieldForServerError('boom')).toBeUndefined()
    expect(stripServerKey('employee_code_padding: must be between 1 and 10')).toBe('must be between 1 and 10')
    expect(stripServerKey('boom')).toBe('boom')
  })
  it('caps the starting number and override length', () => {
    expect(msg({ employee_code_start_number: '9999999999', employee_code_padding: '1', employee_code_prefix: 'A' })).toBeUndefined()
    expect(msg({ employee_code_start_number: '10000000000', employee_code_padding: '1', employee_code_prefix: 'A' })).toMatch(/at most 9999999999/)
    expect(overrideCodeError('ABC1', 'A.*')).toBeDefined()
    expect(overrideCodeError('EMP' + '1'.repeat(18), 'EMP')).toMatch(/at most 20/)
  })
})
