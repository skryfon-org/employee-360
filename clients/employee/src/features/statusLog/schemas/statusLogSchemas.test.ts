import { describe, expect, it } from 'vitest'
import { buildEntrySchema, parseTarget } from './statusLogSchemas'

describe('entry schema', () => {
  const ok = { target: 'category:meeting', hours: '4', notes: '' }
  it('accepts a valid entry', () => expect(buildEntrySchema(0).safeParse(ok).success).toBe(true))
  it('rejects hours <= 0 and non-numeric', () => {
    expect(buildEntrySchema(0).safeParse({ ...ok, hours: '0' }).success).toBe(false)
    expect(buildEntrySchema(0).safeParse({ ...ok, hours: 'x' }).success).toBe(false)
  })
  it('rejects when the day total would exceed 24', () => {
    expect(buildEntrySchema(22).safeParse({ ...ok, hours: '3' }).success).toBe(false)
    expect(buildEntrySchema(22).safeParse({ ...ok, hours: '2' }).success).toBe(true)
  })
  it('parses targets', () => {
    expect(parseTarget('project:abc')).toEqual({ project_id: 'abc' })
    expect(parseTarget('category:other')).toEqual({ activity_category: 'other' })
    expect(parseTarget('category:bogus')).toBeNull()
    expect(parseTarget('')).toBeNull()
  })
})
