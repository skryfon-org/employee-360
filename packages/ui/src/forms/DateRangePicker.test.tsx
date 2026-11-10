import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DateRangePicker } from './DateRangePicker'

describe('DateRangePicker', () => {
  it('shows the placeholder and a formatted range', () => {
    const { rerender } = render(<DateRangePicker value={{ from: '', to: '' }} onChange={() => {}} label="Dates" />)
    expect(screen.getByLabelText('Dates')).toHaveAttribute('placeholder', 'Select date range')
    rerender(<DateRangePicker value={{ from: '2026-12-01', to: '2026-12-31' }} onChange={() => {}} label="Dates" />)
    expect(screen.getByLabelText('Dates')).toHaveValue('01 Dec 2026 - 31 Dec 2026')
  })

  it('emits local YYYY-MM-DD only once both ends are picked, and "" on clear', async () => {
    const onChange = vi.fn()
    render(<DateRangePicker value={{ from: '2026-12-01', to: '2026-12-01' }} onChange={onChange} label="Dates" />)
    await userEvent.click(screen.getByLabelText('Dates'))
    await userEvent.click(await screen.findByRole('gridcell', { name: /December 10th, 2026/ }))
    expect(onChange).not.toHaveBeenCalled()
    await userEvent.click(await screen.findByRole('gridcell', { name: /December 12th, 2026/ }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '2026-12-10', to: '2026-12-12' })
    await userEvent.click(screen.getByRole('button', { name: 'Clear date range' }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '', to: '' })
  })
})
