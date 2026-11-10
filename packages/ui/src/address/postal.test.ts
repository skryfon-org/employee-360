import { describe, expect, it } from 'vitest'
import { validatePostalCode } from './postal'

const IN_MSG = 'Enter a valid 6-digit PIN code'

describe('validatePostalCode', () => {
  it.each(['682001', '110001', '999999'])('accepts India PIN %s', (v) => {
    expect(validatePostalCode('India', v)).toBeNull()
  })
  it.each([
    ['5 digits', '68200'],
    ['7 digits', '6820011'],
    ['leading zero', '012345'],
    ['letters', '68A001'],
    ['inner space', '682 001'],
    ['all letters', 'ABCDEF'],
  ])('rejects India PIN with %s', (_n, v) => {
    expect(validatePostalCode('India', v)).toBe(IN_MSG)
  })
  it('trims and treats empty as valid (optional field)', () => {
    expect(validatePostalCode('India', ' 682001 ')).toBeNull()
    expect(validatePostalCode('India', '')).toBeNull()
    expect(validatePostalCode('India', '   ')).toBeNull()
  })
  it('is case-insensitive on the country name', () => {
    expect(validatePostalCode('india', '1234')).toBe(IN_MSG)
  })
  it('validates US ZIP 5 and 5+4', () => {
    expect(validatePostalCode('United States', '94105')).toBeNull()
    expect(validatePostalCode('United States', '94105-1234')).toBeNull()
    expect(validatePostalCode('United States', '9410')).toMatch(/ZIP/)
    expect(validatePostalCode('United States', '94105-12')).toMatch(/ZIP/)
  })
  it('validates UK and Canada formats', () => {
    expect(validatePostalCode('United Kingdom', 'SW1A 1AA')).toBeNull()
    expect(validatePostalCode('United Kingdom', 'sw1a1aa')).toBeNull()
    expect(validatePostalCode('United Kingdom', '12345')).toMatch(/UK postcode/)
    expect(validatePostalCode('Canada', 'K1A 0B1')).toBeNull()
    expect(validatePostalCode('Canada', 'D1A 0B1')).toMatch(/Canadian/)
  })
  it('validates other common countries', () => {
    expect(validatePostalCode('Germany', '10115')).toBeNull()
    expect(validatePostalCode('Germany', '1011')).toMatch(/5-digit/)
    expect(validatePostalCode('Australia', '2000')).toBeNull()
    expect(validatePostalCode('Singapore', '018956')).toBeNull()
    expect(validatePostalCode('Japan', '100-0001')).toBeNull()
  })
  it('uses a permissive 3-10 character fallback for other or unset countries', () => {
    expect(validatePostalCode('Narnia', 'AB-12 3')).toBeNull()
    expect(validatePostalCode('', '123')).toBeNull()
    expect(validatePostalCode('Narnia', '12')).toMatch(/3-10/)
    expect(validatePostalCode('Narnia', '12345678901')).toMatch(/3-10/)
    expect(validatePostalCode('Narnia', 'ab#123')).toMatch(/3-10/)
  })
})
