import { describe, expect, it } from 'vitest'
import { isAllocationActiveOn, projectsActiveOn } from './allocations'

describe('isAllocationActiveOn (mirrors server ActiveOn)', () => {
  it('is inclusive at both ends and open-ended without end date', () => {
    expect(isAllocationActiveOn({ startDate: '2026-10-10', endDate: '2026-10-20' }, '2026-10-10')).toBe(true)
    expect(isAllocationActiveOn({ startDate: '2026-10-10', endDate: '2026-10-20' }, '2026-10-20')).toBe(true)
    expect(isAllocationActiveOn({ startDate: '2026-10-10' }, '2027-01-01')).toBe(true)
  })
  it('excludes dates before start or after end, and tolerates timestamps', () => {
    expect(isAllocationActiveOn({ startDate: '2026-10-10' }, '2026-10-09')).toBe(false)
    expect(isAllocationActiveOn({ startDate: '2026-10-10', endDate: '2026-10-20T00:00:00Z' }, '2026-10-21')).toBe(false)
  })
  it('filters project options', () => {
    const ps = [
      { id: 'a', label: 'A', startDate: '2026-10-01' },
      { id: 'b', label: 'B', startDate: '2026-10-01', endDate: '2026-10-05' },
    ]
    expect(projectsActiveOn(ps, '2026-10-10').map((p) => p.id)).toEqual(['a'])
  })
})
