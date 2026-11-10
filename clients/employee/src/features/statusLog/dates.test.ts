import { describe, expect, it } from 'vitest'
import { lastSevenDays } from './dates'

describe('lastSevenDays', () => {
  it('returns today back to today-6 across a month boundary', () => {
    expect(lastSevenDays(new Date(2026, 10, 3))).toEqual([
      '2026-11-03', '2026-11-02', '2026-11-01', '2026-10-31', '2026-10-30', '2026-10-29', '2026-10-28',
    ])
  })
})
