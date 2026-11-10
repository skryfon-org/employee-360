import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'
import { HolidayYearGrid } from './HolidayYearGrid'

const DAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']

describe('HolidayYearGrid week start', () => {
  // 1 June 2026 is a Monday (weekday 1).
  it.each([0, 1, 2, 3, 4, 5, 6])('rotates headers and leading blanks for week start %i', (start) => {
    render(<HolidayYearGrid year={2026} holidays={[]} onSelect={() => {}} onOpenMonth={() => {}} weekStartDay={start} />)
    const june = screen.getByRole('region', { name: 'June 2026' })
    const grid = june.querySelector('.grid') as HTMLElement
    const cells = Array.from(grid.children)
    expect(cells.slice(0, 7).map((c) => c.textContent)).toEqual([...DAYS.slice(start), ...DAYS.slice(0, start)])
    const firstDayIndex = cells.findIndex((c, i) => i >= 7 && c.textContent === '1')
    expect(firstDayIndex - 7).toBe((1 - start + 7) % 7)
    expect(within(june).getAllByText('30')).toHaveLength(1)
  })

  it('defaults to Sunday-first', () => {
    render(<HolidayYearGrid year={2026} holidays={[]} onSelect={() => {}} onOpenMonth={() => {}} />)
    const grid = screen.getByRole('region', { name: 'June 2026' }).querySelector('.grid') as HTMLElement
    expect(grid.children[0].textContent).toBe('Su')
  })
})

describe('HolidayYearGrid today and read-only', () => {
  it('marks today with aria-current and renders holidays without buttons when onSelect is omitted', () => {
    const now = new Date()
    const pad = (n: number) => String(n).padStart(2, '0')
    const date = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
    render(<HolidayYearGrid year={now.getFullYear()} holidays={[{ id: 'h', name: 'Today Fest', date }]} onOpenMonth={() => {}} />)
    const current = document.querySelector('[aria-current="date"]') as HTMLElement
    expect(current).toHaveTextContent(String(now.getDate()))
    expect(screen.getByText(/Today Fest, /)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Today Fest/ })).not.toBeInTheDocument()
  })
})

describe('HolidayYearGrid multiple holidays on one day', () => {
  const two = [
    { id: 'a', name: 'Alpha Day', date: '2026-03-04', category: { color: '#15803D' } },
    { id: 'b', name: 'Beta Day', date: '2026-03-04', category: { color: '#B45309' } },
  ]

  it('renders one button per holiday and selects the clicked one', () => {
    const onSelect = vi.fn()
    render(<HolidayYearGrid year={2026} holidays={two} onSelect={onSelect} onOpenMonth={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /Beta Day, / }))
    expect(onSelect).toHaveBeenLastCalledWith(two[1])
    fireEvent.click(screen.getByRole('button', { name: /Alpha Day, / }))
    expect(onSelect).toHaveBeenLastCalledWith(two[0])
    expect(screen.getByRole('button', { name: /Beta Day, / })).toHaveClass('h-8', 'w-8', 'md:h-4', 'md:w-4')
    expect(screen.getByRole('button', { name: /Beta Day, / }).querySelector('span')).toHaveStyle({ backgroundColor: '#B45309' })
  })

  it('stays non-interactive without onSelect', () => {
    render(<HolidayYearGrid year={2026} holidays={two} onOpenMonth={() => {}} />)
    expect(screen.queryByRole('button', { name: /Alpha Day|Beta Day/ })).not.toBeInTheDocument()
    expect(screen.getByText(/Alpha Day, .*Beta Day, /)).toBeInTheDocument()
  })
})

describe('HolidayYearGrid category colour', () => {
  it('paints each day dot with its category colour', () => {
    render(
      <HolidayYearGrid
        year={2026}
        holidays={[{ id: 'h', name: 'Holi', date: '2026-03-04', category: { color: '#B45309' } }]}
        onSelect={() => {}}
        onOpenMonth={() => {}}
      />,
    )
    expect(screen.getByRole('button', { name: /Holi, / }).querySelector('span[aria-hidden="true"].rounded-full')).toHaveStyle({ backgroundColor: '#B45309' })
  })
})
