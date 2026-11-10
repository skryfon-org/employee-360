import { describe, expect, it } from 'vitest'
import { holidayFormSchema, isValidIsoDate } from './holidaySchemas'

const valid = { name: 'Diwali', date: '2026-11-08', category_id: 'c1', description: '' }

describe('holidayFormSchema', () => {
  it('accepts a valid holiday with empty description', () => {
    expect(holidayFormSchema.safeParse(valid).success).toBe(true)
  })

  it('rejects blank names', () => {
    const r = holidayFormSchema.safeParse({ ...valid, name: '   ' })
    expect(r.success).toBe(false)
  })

  it('requires a category', () => {
    expect(holidayFormSchema.safeParse({ ...valid, category_id: '' }).success).toBe(false)
  })

  it('rejects empty and impossible dates', () => {
    expect(holidayFormSchema.safeParse({ ...valid, date: '' }).success).toBe(false)
    expect(holidayFormSchema.safeParse({ ...valid, date: '2026-02-30' }).success).toBe(false)
    expect(isValidIsoDate('2028-02-29')).toBe(true)
    expect(isValidIsoDate('2026-13-01')).toBe(false)
  })

  it('limits description length', () => {
    expect(holidayFormSchema.safeParse({ ...valid, description: 'x'.repeat(501) }).success).toBe(false)
  })
})
