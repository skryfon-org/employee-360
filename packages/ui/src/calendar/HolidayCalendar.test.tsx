import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { HolidayCalendar } from './HolidayCalendar'

const base = { cursor: { year: 2026, month: 6 }, holidays: [], onCursorChange: () => {}, scope: 'month' as const, onScopeChange: () => {} }
const cellOf = (date: string) => document.querySelector(`[data-date="${date}"]`) as HTMLElement

describe('HolidayCalendar generic props', () => {
  it('keeps the holiday defaults', () => {
    render(<HolidayCalendar {...base} />)
    expect(screen.getByRole('group', { name: 'Holiday calendar' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Calendar scope' })).toBeInTheDocument()
  })

  it('supports a custom label and hides the scope toggle', () => {
    render(<HolidayCalendar {...base} ariaLabel="Status calendar" showScopeToggle={false} />)
    expect(screen.getByRole('group', { name: 'Status calendar' })).toBeInTheDocument()
    expect(screen.queryByRole('group', { name: 'Calendar scope' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Next month' })).toBeInTheDocument()
  })

  it('renders day labels without a category dot and highlights the selected day', () => {
    const { container } = render(
      <HolidayCalendar {...base} selectedDate="2026-06-10" dayLabels={[{ date: '2026-06-10', text: '6h', ariaLabel: '6 hours' }, { date: '2026-06-11', text: 'None', muted: true }]} dayCellClassNames={(d) => (d === '2026-06-10' ? ['is-editable'] : [])} />,
    )
    expect(screen.getByText('6h')).toHaveAttribute('title', '6 hours')
    expect(screen.getByText('None')).toHaveClass('holiday-day-label--muted')
    expect(container.querySelector('.holiday-event__dot')).toBeNull()
    expect(cellOf('2026-06-10')).toHaveClass('holiday-day--selected', 'is-editable')
    expect(cellOf('2026-06-11')).not.toHaveClass('holiday-day--selected')
  })

  it('does not fire onDateClick for disabled dates', () => {
    const onDateClick = vi.fn()
    render(<HolidayCalendar {...base} onDateClick={onDateClick} isDateDisabled={(d) => d > '2026-06-15'} />)
    expect(cellOf('2026-06-20')).toHaveClass('holiday-day--disabled')
    expect(cellOf('2026-06-20')).toHaveAttribute('aria-disabled', 'true')
    fireEvent.click(cellOf('2026-06-20').querySelector('.fc-daygrid-day-frame') as HTMLElement)
    expect(onDateClick).not.toHaveBeenCalled()
    fireEvent.click(cellOf('2026-06-10').querySelector('.fc-daygrid-day-frame') as HTMLElement)
    expect(onDateClick).toHaveBeenCalledWith('2026-06-10')
  })
})
