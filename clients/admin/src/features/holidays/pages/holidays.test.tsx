import { pickRowAction } from '../../../test/rowActions'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, renderHook, screen, waitFor, within } from '@testing-library/react'
import { useHolidayCategoriesQuery } from '../queries/holidayQueries'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import { MemoryRouter } from 'react-router-dom'
import HolidaysPage from './HolidaysPage'
import { useHolidayViewStore } from '../../../stores/holidayViewStore'

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <HolidaysPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const PUBLIC_ID = 'cat-public'
const OPTIONAL_ID = 'cat-optional'
const categories = [
  { id: PUBLIC_ID, name: 'Public Holiday', color: '#15803D', is_active: true },
  { id: OPTIONAL_ID, name: 'Optional Holiday', color: '#B45309', is_active: true },
]
const seed: Array<[string, string, string]> = [
  ['2026-01-01', 'New Year', PUBLIC_ID],
  ['2026-03-04', 'Holi', OPTIONAL_ID],
  ['2026-05-01', 'Labour Day', PUBLIC_ID],
  ['2026-06-10', 'Founders Day', PUBLIC_ID],
  ['2026-08-15', 'Independence Day', PUBLIC_ID],
  ['2026-10-20', 'Dussehra', OPTIONAL_ID],
  ['2026-11-08', 'Diwali', PUBLIC_ID],
  ['2026-12-25', 'Christmas', PUBLIC_ID],
]
let rows: Array<{ id: string; date: string; name: string; category_id: string }> = []
let seq = 0
let mock: MockAdapter
const idOf = (url?: string) => url?.split('/').pop() ?? ''

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'], now: new Date('2026-06-15T12:00:00Z') })
  seq = 0
  rows = seed.map(([date, name, category_id]) => ({ id: `h${++seq}`, date, name, category_id }))
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: categories })
  mock.onGet('/api/v1/holidays').reply((config) => {
    const year = String(config.params?.year)
    return [200, { success: true, data: rows.filter((r) => r.date.startsWith(`${year}-`)) }]
  })
  mock.onPost('/api/v1/holidays').reply((config) => {
    const body = JSON.parse(config.data)
    const row = { id: `h${++seq}`, ...body }
    rows.push(row)
    return [201, { success: true, data: row }]
  })
  mock.onDelete(/\/api\/v1\/holidays\/.+/).reply((config) => {
    rows = rows.filter((r) => r.id !== idOf(config.url))
    return [204]
  })
  useHolidayViewStore.setState({ viewMode: 'calendar', calendarScope: 'month' })
})
afterEach(() => {
  vi.useRealTimers()
  mock.restore()
})

describe('HolidaysPage (API backed)', () => {
  it('shows a loading state then the calendar for the current year', async () => {
    const user = userEvent.setup()
    renderPage()
    expect(screen.getByText('Loading holidays')).toBeInTheDocument()
    expect(await screen.findByRole('group', { name: 'Holiday calendar' })).toBeInTheDocument()
    expect(screen.getByText('2026')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Next month' }))
    await user.click(screen.getByRole('button', { name: 'Next month' }))
    await user.click(screen.getByRole('button', { name: 'Next month' }))
    await user.click(screen.getByRole('button', { name: 'Next month' }))
    await user.click(screen.getByRole('button', { name: 'Next month' }))
    expect(await screen.findByText(/Diwali/)).toBeInTheDocument()
  })

  it('switches between the full-year calendar and a single month', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    expect(screen.getByRole('heading', { name: 'June 2026' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Year' }))
    expect(screen.getByRole('region', { name: 'December 2026' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Diwali/ })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'March' }))
    expect(screen.getByRole('heading', { name: 'March 2026' })).toBeInTheDocument()
  })

  it('toggles to the list view and back', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    await user.click(screen.getByRole('button', { name: 'List' }))
    const list = await screen.findByRole('list', { name: 'Holidays' })
    expect(within(list).getAllByRole('listitem').length).toBeGreaterThan(5)
    expect(useHolidayViewStore.getState().viewMode).toBe('list')
    await user.click(screen.getByRole('button', { name: 'Calendar' }))
    expect(await screen.findByRole('group', { name: 'Holiday calendar' })).toBeInTheDocument()
  })

  it('shows the empty-year state', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    await user.click(screen.getByRole('button', { name: 'Next year' }))
    expect(await screen.findByText(/No holidays have been added for 2027/)).toBeInTheDocument()
  })

  it('adds a holiday and it appears in the list', async () => {
    const user = userEvent.setup()
    useHolidayViewStore.setState({ viewMode: 'list' })
    renderPage()
    await screen.findByRole('list', { name: 'Holidays' })
    await user.click(screen.getByRole('button', { name: 'Add holiday' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Name'), 'Team Day')
    await user.selectOptions(within(dialog).getByLabelText('Category'), 'Optional Holiday')
    await user.click(within(dialog).getByRole('button', { name: 'Add holiday' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(await screen.findByText('Team Day')).toBeInTheDocument()
  })

  it('validates required fields', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    await user.click(screen.getByRole('button', { name: 'Add holiday' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Add holiday' }))
    expect(await within(dialog).findByText('Name is required')).toBeInTheDocument()
    expect(within(dialog).getByText('Category is required')).toBeInTheDocument()
  })

  it('confirms before deleting', async () => {
    const user = userEvent.setup()
    useHolidayViewStore.setState({ viewMode: 'list' })
    renderPage()
    await pickRowAction('Actions for Labour Day', 'Delete')
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Labour Day')).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(screen.queryByText('Labour Day')).toBeNull())
  })

  it('shows an error state with retry when loading fails', async () => {
    mock.onGet('/api/v1/holidays').reply(500, { success: false, error: { code: 'INTERNAL', message: 'boom' } })
    renderPage()
    expect(await screen.findByText('boom')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})

describe('useHolidayCategoriesQuery pagination', () => {
  it('fetches every page of categories', async () => {
    const mkCat = (n: number) => ({ id: `c${n}`, name: `Cat ${n}`, color: '#000000', is_active: true })
    const calls: number[] = []
    mock.onGet('/api/v1/holiday-categories').reply((config) => {
      const page = Number(config.params?.page)
      calls.push(page)
      const start = (page - 1) * 100
      const count = page === 3 ? 20 : 100
      return [
        200,
        {
          success: true,
          data: Array.from({ length: count }, (_, i) => mkCat(start + i)),
          meta: { page, page_size: 100, total_items: 220, total_pages: 3 },
        },
      ]
    })
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useHolidayCategoriesQuery(), {
      wrapper: ({ children }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>,
    })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(calls).toEqual([1, 2, 3])
    expect(result.current.data).toHaveLength(220)
    expect(result.current.data?.[219].id).toBe('c219')
  })
})

describe('Add holiday without categories', () => {
  it('still opens the form and explains why no category can be chosen', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: [] })
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    const btn = screen.getByRole('button', { name: 'Add holiday' })
    expect(btn).toBeEnabled()
    await user.click(btn)
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText(/No active holiday categories/)).toBeInTheDocument()
  })
})
