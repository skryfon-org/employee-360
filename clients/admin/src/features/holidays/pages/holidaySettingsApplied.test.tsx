import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import HolidaysPage from './HolidaysPage'
import HolidaySettingsPage from './HolidaySettingsPage'
import { HOLIDAY_SETTINGS_KEY } from '../queries/holidayQueries'
import { useHolidayViewStore } from '../../../stores/holidayViewStore'

type Settings = { default_view: string; default_year?: number; week_start_day: number; updated_at: string }
let settings: Settings
let mock: MockAdapter
let qc: QueryClient

function renderPage() {
  qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <HolidaysPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const holidays = [
  { id: 'h1', date: '2026-06-10', name: 'Founders Day', category_id: 'c1' },
  { id: 'h2', date: '2027-02-01', name: 'Next Year Day', category_id: 'c1' },
]

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'], now: new Date('2026-06-15T12:00:00Z') })
  settings = { default_view: 'calendar', week_start_day: 0, updated_at: '2026-01-01T00:00:00Z' }
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: [{ id: 'c1', name: 'Public', color: '#15803D', is_active: true }] })
  mock.onGet('/api/v1/holidays').reply((config) => [200, { success: true, data: holidays.filter((h) => h.date.startsWith(`${config.params?.year}-`)) }])
  mock.onGet('/api/v1/holiday-calendar/settings').reply(() => [200, { success: true, data: settings }])
  useHolidayViewStore.setState({ viewMode: 'calendar', calendarScope: 'month' })
})
afterEach(() => {
  vi.useRealTimers()
  mock.restore()
})

const monthHeaders = () => screen.getAllByRole('columnheader').map((h) => h.textContent)

describe('HolidaysPage applies calendar settings', () => {
  it('opens in list view when default_view is list', async () => {
    settings.default_view = 'list'
    renderPage()
    expect(await screen.findByRole('list', { name: 'Holidays' })).toBeInTheDocument()
    expect(useHolidayViewStore.getState().viewMode).toBe('list')
  })

  it('opens in calendar view when default_view is calendar (overriding a stale list mode)', async () => {
    useHolidayViewStore.setState({ viewMode: 'list' })
    renderPage()
    expect(await screen.findByRole('group', { name: 'Holiday calendar' })).toBeInTheDocument()
  })

  it('opens on default_year', async () => {
    settings.default_year = 2027
    settings.default_view = 'list'
    renderPage()
    expect(await screen.findByText('Next Year Day')).toBeInTheDocument()
    expect(screen.getByText('2027')).toBeInTheDocument()
  })

  it('uses Monday-first headers in the month calendar and the year grid when week_start_day is 1', async () => {
    settings.week_start_day = 1
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    await waitFor(() => expect(monthHeaders()[0]).toMatch(/^Mon/))
    expect(monthHeaders()[6]).toMatch(/^Sun/)
    // Year grid renders in the DOM only in year scope.
    useHolidayViewStore.setState({ calendarScope: 'year' })
    const june = await screen.findByRole('region', { name: 'June 2026' })
    const first = (june.querySelector('.grid') as HTMLElement).children[0]
    expect(first.textContent).toBe('Mo')
  })

  it('defaults to Sunday-first, calendar view and current year', async () => {
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    expect(monthHeaders()[0]).toMatch(/^Sun/)
    expect(screen.getByText('2026')).toBeInTheDocument()
  })

  it('re-applies when settings change (updated_at changes) while the page is mounted', async () => {
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    settings = { default_view: 'list', default_year: 2027, week_start_day: 1, updated_at: '2026-06-15T12:00:00Z' }
    await qc.invalidateQueries({ queryKey: HOLIDAY_SETTINGS_KEY })
    expect(await screen.findByRole('list', { name: 'Holidays' })).toBeInTheDocument()
    expect(await screen.findByText('Next Year Day')).toBeInTheDocument()
  })

  it('falls back to defaults when settings return 404', async () => {
    mock.onGet('/api/v1/holiday-calendar/settings').reply(404, { success: false, error: { code: 'NOT_FOUND', message: 'holiday calendar settings not found' } })
    renderPage()
    await screen.findByRole('group', { name: 'Holiday calendar' })
    expect(monthHeaders()[0]).toMatch(/^Sun/)
    expect(screen.getByText('2026')).toBeInTheDocument()
  })
})

describe('HolidaySettingsPage when settings are missing', () => {
  it('shows an error with a retry button on 404', async () => {
    mock.onGet('/api/v1/holiday-calendar/settings').reply(404, { success: false, error: { code: 'NOT_FOUND', message: 'holiday calendar settings not found' } })
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <HolidaySettingsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    )
    const region = await screen.findByRole('region', { name: 'Calendar settings' })
    expect(await within(region).findByText(/settings not found/)).toBeInTheDocument()
    expect(within(region).getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})
