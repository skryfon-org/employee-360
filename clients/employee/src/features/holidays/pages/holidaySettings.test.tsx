import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import { useUiStore } from '../../../stores/uiStore'
import HolidaysPage from './HolidaysPage'

const rows = [
  { id: 'h1', date: '2026-06-10', name: 'Founders Day', category_id: 'c1' },
  { id: 'h2', date: '2027-02-01', name: 'Next Year Day', category_id: 'c1' },
]
let settings: Record<string, unknown>
let mock: MockAdapter

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <HolidaysPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'], now: new Date('2026-06-15T12:00:00Z') })
  settings = { default_view: 'calendar', week_start_day: 0, updated_at: '2026-01-01T00:00:00Z' }
  useUiStore.setState({ holidayView: 'calendar', holidayScope: 'month' })
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: [{ id: 'c1', name: 'Public', color: '#15803D', is_active: true }] })
  mock.onGet('/api/v1/holidays').reply((c) => [200, { success: true, data: rows.filter((r) => r.date.startsWith(`${c.params?.year}-`) && (!c.params?.category_id || r.category_id === c.params.category_id)) }])
  mock.onGet('/api/v1/holiday-calendar/settings').reply(() => [200, { success: true, data: settings }])
})
afterEach(() => {
  mock.restore()
  vi.useRealTimers()
})

describe('employee HolidaysPage applies calendar settings', () => {
  it('rotates month headers and the year grid for week_start_day 1', async () => {
    settings.week_start_day = 1
    renderPage()
    await screen.findByText('Founders Day')
    const headers = () => screen.getAllByRole('columnheader').map((h) => h.textContent)
    await vi.waitFor(() => expect(headers()[0]).toMatch(/^Mon/))
    expect(headers()[6]).toMatch(/^Sun/)
    await userEvent.click(screen.getByRole('button', { name: 'Year' }))
    const june = screen.getByRole('region', { name: 'June 2026' })
    expect((june.querySelector('.grid') as HTMLElement).children[0].textContent).toBe('Mo')
  })

  it('opens in list view on default_year and does not override later manual changes', async () => {
    settings.default_view = 'list'
    settings.default_year = 2027
    renderPage()
    expect(await screen.findByText('Next Year Day')).toBeInTheDocument()
    expect(screen.getByText('2027')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Previous year' }))
    expect(await screen.findByText('Founders Day')).toBeInTheDocument()
    expect(screen.getByText('2026')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    expect(within(screen.getByRole('region', { name: 'Holidays' })).getByRole('group', { name: 'Holiday calendar' })).toBeInTheDocument()
  })

  it('falls back to defaults silently when settings fail to load', async () => {
    mock.onGet('/api/v1/holiday-calendar/settings').reply(500)
    renderPage()
    expect(await screen.findByRole('group', { name: 'Holiday calendar' })).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
