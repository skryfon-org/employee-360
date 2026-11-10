import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import { YearSelector } from '../components/YearSelector'
import { holidaysYearKey } from '../queries/holidayQueries'
import { useUiStore } from '../../../stores/uiStore'
import HolidaysPage from './HolidaysPage'

const categories = [
  { id: 'c1', name: 'Public Holiday', color: '#15803D', is_active: true },
  { id: 'c2', name: 'Optional Holiday', color: '#B45309', is_active: true },
]
const rows = [
  { id: 'h2', date: '2026-03-04', name: 'Holi', category_id: 'c2' },
  { id: 'h1', date: '2026-01-01', name: 'New Year', category_id: 'c1' },
]

let mock: MockAdapter

let lastClient: QueryClient

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  lastClient = qc
  return render(
    <QueryClientProvider client={qc}>
      <HolidaysPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  useUiStore.setState({ holidayView: 'list', holidayScope: 'month' })
  vi.useFakeTimers({ toFake: ['Date'], now: new Date('2026-06-15T12:00:00Z') })
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: categories })
  mock.onGet('/api/v1/holidays').reply((config) => {
    const year = String(config.params?.year)
    const cat = config.params?.category_id
    return [200, { success: true, data: rows.filter((r) => r.date.startsWith(`${year}-`) && (!cat || r.category_id === cat)) }]
  })
})
afterEach(() => {
  mock.restore()
  vi.useRealTimers()
})

describe('employee HolidaysPage', () => {
  it('shows a loading skeleton then a chronological read-only list', async () => {
    renderPage()
    expect(screen.getByRole('status')).toHaveTextContent('Loading holidays')
    const items = await screen.findAllByRole('listitem')
    expect(items[0]).toHaveTextContent('New Year')
    expect(items[1]).toHaveTextContent('Holi')
    expect(screen.queryByRole('button', { name: /edit|delete|add/i })).not.toBeInTheDocument()
  })

  it('filters by category', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.selectOptions(screen.getByLabelText('Category'), 'c2')
    expect(screen.queryByText('New Year')).not.toBeInTheDocument()
    expect(await screen.findByText('Holi')).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText('New Year')).not.toBeInTheDocument())
    const calls = mock.history.get.filter((r) => r.url === '/api/v1/holidays')
    expect(calls[0].params).toEqual({ year: 2026 })
    expect(calls[0].params).not.toHaveProperty('category_id')
    expect(calls[calls.length - 1].params).toEqual({ year: 2026, category_id: 'c2' })

    await userEvent.selectOptions(screen.getByLabelText('Category'), '')
    expect(await screen.findByText('New Year')).toBeInTheDocument()
    const last = mock.history.get.filter((r) => r.url === '/api/v1/holidays').at(-1)
    expect(last?.params).toEqual({ year: 2026 })
    expect(last?.params).not.toHaveProperty('category_id')
  })

  it('shows a category-specific empty state when the category has no holidays', async () => {
    renderPage()
    await screen.findByText('New Year')
    mock.onGet('/api/v1/holidays').reply((config) =>
      config.params?.category_id === 'c2' ? [200, { success: true, data: [] }] : [200, { success: true, data: rows }],
    )
    await userEvent.selectOptions(screen.getByLabelText('Category'), 'c2')
    expect(await screen.findByText('No holidays in this category for 2026.')).toBeInTheDocument()
    expect(screen.queryByText(/have been published/)).not.toBeInTheDocument()
  })

  it('requests a different year and shows the empty-year state', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Next year' }))
    expect(await screen.findByText(/No holidays have been published for 2027/)).toBeInTheDocument()
    expect(mock.history.get.some((r) => r.url === '/api/v1/holidays' && r.params?.year === 2027)).toBe(true)
    expect(lastClient.getQueryData(holidaysYearKey(2026))).toHaveLength(2)
    expect(lastClient.getQueryData(holidaysYearKey(2027))).toEqual([])
    expect(holidaysYearKey(2026)).not.toEqual(holidaysYearKey(2027))

    await userEvent.click(screen.getByRole('button', { name: 'Previous year' }))
    expect(await screen.findByText('New Year')).toBeInTheDocument()
    expect(screen.queryByText(/No holidays have been published for 2027/)).not.toBeInTheDocument()
  })

  it('shows a neutral placeholder, not Uncategorised, while categories load', async () => {
    mock.onGet('/api/v1/holiday-categories').reply(() => new Promise(() => {}))
    renderPage()
    await screen.findByText('New Year')
    expect(screen.queryByText('Uncategorised')).not.toBeInTheDocument()
    expect(screen.getAllByLabelText('Loading category')).toHaveLength(2)
  })

  it('shows a non-blocking notice when categories fail to load', async () => {
    mock.onGet('/api/v1/holiday-categories').reply(500)
    renderPage()
    expect(await screen.findByText('New Year')).toBeInTheDocument()
    expect(await screen.findByText(/category filter is unavailable/)).toBeInTheDocument()
    expect(screen.queryByText('Uncategorised')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Category')).toBeDisabled()
  })

  it('shows an error with retry', async () => {
    mock.onGet('/api/v1/holidays').reply(500)
    renderPage()
    expect(await screen.findByRole('button', { name: 'Retry' })).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
  })
})

describe('YearSelector bounds', () => {
  it('disables Previous at 2000 and Next at 2100', async () => {
    const onChange = vi.fn()
    const { rerender } = render(<YearSelector year={2000} onChange={onChange} />)
    const prev = screen.getByRole('button', { name: 'Previous year' })
    expect(prev).toBeDisabled()
    expect(prev).toHaveAttribute('aria-disabled', 'true')
    await userEvent.click(prev)
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Next year' })).toBeEnabled()
    rerender(<YearSelector year={2100} onChange={onChange} />)
    expect(screen.getByRole('button', { name: 'Next year' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Previous year' })).toBeEnabled()
  })
})

describe('employee HolidaysPage calendar view', () => {
  it('toggles to the shared month calendar, shows holidays, stays within the year, and is read-only', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    // Fake date is June 2026; go back to March
    expect(screen.getByRole('heading', { name: 'June 2026' })).toBeInTheDocument()
    for (let i = 0; i < 3; i++) await userEvent.click(screen.getByRole('button', { name: 'Previous month' }))
    expect(screen.getByRole('heading', { name: 'March 2026' })).toBeInTheDocument()
    expect(await screen.findByText('Holi')).toBeInTheDocument()
    for (let i = 0; i < 2; i++) await userEvent.click(screen.getByRole('button', { name: 'Previous month' }))
    expect(screen.getByRole('button', { name: 'Previous month' })).toBeDisabled()
    expect(screen.getByRole('heading', { name: 'January 2026' })).toBeInTheDocument()
    expect(await screen.findByText('New Year')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit|delete|add/i })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'List' }))
    expect(screen.queryByRole('group', { name: 'Holiday calendar' })).not.toBeInTheDocument()
  })

  it('disables Next at December and Today outside the selected year', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    for (let i = 0; i < 6; i++) await userEvent.click(screen.getByRole('button', { name: 'Next month' }))
    expect(screen.getByRole('heading', { name: 'December 2026' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Next month' })).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: 'Next year' }))
    expect(screen.getByRole('heading', { name: 'January 2027' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Today' })).toBeDisabled()
  })

  it('highlights today in the month grid', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    const today = document.querySelector('[aria-current="date"]')
    expect(today).not.toBeNull()
    expect(today).toHaveTextContent('15')
  })

  it('shows all 12 months in year scope, highlights today, and opens a month on click', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    await userEvent.click(screen.getByRole('button', { name: 'Year' }))
    expect(screen.getAllByRole('region', { name: /2026$/ })).toHaveLength(12)
    expect(within(screen.getByRole('region', { name: 'June 2026' })).getByText('15')).toHaveAttribute('aria-current', 'date')
    await userEvent.click(screen.getByRole('button', { name: 'March' }))
    expect(screen.getByRole('heading', { name: 'March 2026' })).toBeInTheDocument()
  })

  it('applies the category filter to the calendar', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    await userEvent.click(screen.getByRole('button', { name: 'Year' }))
    await userEvent.selectOptions(screen.getByLabelText('Category'), 'c2')
    expect(await screen.findByRole('button', { name: /Holi, / })).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByRole('button', { name: /New Year/ })).not.toBeInTheDocument())
  })

  it('opens a read-only details dialog from the year grid and closes it with Close, Escape and backdrop', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    await userEvent.click(screen.getByRole('button', { name: 'Year' }))
    await userEvent.click(screen.getByRole('button', { name: /Holi, Wednesday, 4 March 2026/ }))
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByRole('heading', { name: 'Holi' })).toBeInTheDocument()
    expect(dialog).toHaveTextContent('Wednesday, 4 March 2026')
    expect(dialog).toHaveTextContent('Optional Holiday')
    expect(screen.getByRole('button', { name: 'Close' })).toHaveFocus()
    await userEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /Holi, Wednesday, 4 March 2026/ }))
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /Holi, Wednesday, 4 March 2026/ }))
    await userEvent.click(screen.getByRole('dialog').parentElement as HTMLElement)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('opens the details dialog when a month-view event is clicked', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    for (let i = 0; i < 5; i++) await userEvent.click(screen.getByRole('button', { name: 'Previous month' }))
    await userEvent.click(await screen.findByText('New Year'))
    expect(await screen.findByRole('dialog')).toHaveTextContent('Thursday, 1 January 2026')
    await userEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('colours the month-view event dot with the holiday category colour', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    for (let i = 0; i < 3; i++) await userEvent.click(screen.getByRole('button', { name: 'Previous month' }))
    const holi = (await screen.findByText('Holi')).closest('.holiday-event') as HTMLElement
    expect(holi.querySelector('.holiday-event__dot')).toHaveStyle({ backgroundColor: '#B45309' })
    for (let i = 0; i < 2; i++) await userEvent.click(screen.getByRole('button', { name: 'Previous month' }))
    const ny = (await screen.findByText('New Year')).closest('.holiday-event') as HTMLElement
    expect(ny.querySelector('.holiday-event__dot')).toHaveStyle({ backgroundColor: '#15803D' })
  })

  it('colours the year-grid dots with the holiday category colour', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    await userEvent.click(screen.getByRole('button', { name: 'Year' }))
    const dotOf = (name: RegExp) => screen.getByRole('button', { name }).querySelector('span[aria-hidden="true"].rounded-full')
    expect(dotOf(/Holi, /)).toHaveStyle({ backgroundColor: '#B45309' })
    expect(dotOf(/New Year, /)).toHaveStyle({ backgroundColor: '#15803D' })
  })
})

// jsdom has no layout engine or media queries, so these only verify the responsive class/CSS contract
// that produces the mobile layout; they do not render at a narrow viewport.
describe('employee HolidaysPage mobile layout contract (class/CSS only, not a viewport check)', () => {
  it('uses 44px (h-11) touch targets on month navigation and Today, shrinking only from md up', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    for (const name of ['Previous month', 'Next month']) {
      const btn = screen.getByRole('button', { name })
      expect(btn).toHaveClass('h-11', 'w-11', 'md:h-9', 'md:w-9')
    }
    expect(screen.getByRole('button', { name: 'Today' })).toHaveClass('h-11', 'md:h-9')
  })

  it('lays the year grid out as one column by default, widening only at sm/xl breakpoints', async () => {
    renderPage()
    await screen.findByText('New Year')
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    await userEvent.click(screen.getByRole('button', { name: 'Year' }))
    const grid = screen.getByRole('region', { name: 'June 2026' }).parentElement as HTMLElement
    expect(grid).toHaveClass('grid', 'sm:grid-cols-2', 'xl:grid-cols-3')
    expect(grid.className).not.toMatch(/(^|\s)grid-cols-/)
  })

  it('ships a max-width 639px stylesheet rule that hides event names so month cells show dots only', () => {
    // Resolve through the workspace package (node_modules symlink), not a deep relative path.
    const cssPath = createRequire(import.meta.url).resolve('@employee360/ui/src/calendar/holidayCalendar.css')
    const css = readFileSync(cssPath, 'utf8')
    const media = css.slice(css.indexOf('@media (max-width: 639px)'))
    expect(media).toMatch(/\.holiday-event__name\s*\{\s*display:\s*none/)
    expect(css.slice(0, css.indexOf('@media'))).not.toMatch(/holiday-event__name\s*\{[^}]*display:\s*none/)
  })
})
