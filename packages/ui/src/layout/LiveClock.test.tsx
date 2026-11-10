import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { LiveClock } from './LiveClock'
import { formatClockParts } from './clock-utils'

describe('LiveClock', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 9, 20, 14, 32, 5))
  })
  afterEach(() => vi.useRealTimers())

  it('renders and ticks every second', () => {
    render(<LiveClock />)
    const initial = formatClockParts(new Date())
    expect(screen.getByTestId('live-clock').textContent).toContain(initial.time)
    act(() => {
      vi.advanceTimersByTime(1000)
    })
    expect(screen.getByTestId('live-clock').textContent).toContain(formatClockParts(new Date()).time)
    expect(screen.getByTestId('live-clock').textContent).not.toContain(initial.time)
  })

  it('clears the interval on unmount', () => {
    const { unmount } = render(<LiveClock />)
    unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})
