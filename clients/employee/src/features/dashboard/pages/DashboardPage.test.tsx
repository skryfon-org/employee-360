import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import DashboardPage from './DashboardPage'

const URL = '/api/v1/dashboard/employee'
let mock: MockAdapter

const leaveItem = { id: 'l1', leave_type_name: 'Annual Leave', leave_type_code: 'AL', start_date: '2026-11-10', end_date: '2026-11-12', day_part: 'full', days: 3, status: 'approved' }

const populated = {
  greeting: { first_name: 'Ada', welcome_message: 'Welcome to Acme', today: { date: '2026-10-30', kind: 'holiday', holiday_name: 'Founders Day' } },
  stats: { pending_leave: 2, upcoming_leave_days: 3, allocation_percent: 80, hours_this_week: 31, pending_approvals: 4 },
  leave: { next_approved: leaveItem, recent: [{ ...leaveItem, id: 'l2', status: 'pending' }] },
  holidays: [{ id: 'h1', name: 'Diwali', date: '2026-11-08', category_name: 'National', category_color: '#ff0000', days_until: 9 }],
  projects: [{ project_id: 'p1', project_name: 'Apollo', project_code: 'APL', role_on_project: 'Developer', allocation_percent: 80, end_date: null }],
  week: {
    days: Array.from({ length: 7 }, (_, i) => ({ date: `2026-10-${24 + i}`, hours: i, is_working_day: i % 2 === 0 })),
    missing_days: 2,
  },
}

const empty = {
  greeting: { first_name: 'Ada', welcome_message: '', today: { date: '2026-10-30', kind: 'working_day' } },
  stats: { pending_leave: 0, upcoming_leave_days: 0, allocation_percent: 0, hours_this_week: 0, pending_approvals: null },
  leave: { next_approved: null, recent: [] },
  holidays: [],
  projects: [],
  week: { days: [], missing_days: 0 },
}

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('employee DashboardPage', () => {
  beforeEach(() => {
    mock = new MockAdapter(apiClient)
  })
  afterEach(() => mock.restore())

  it('shows a skeleton while loading', () => {
    mock.onGet(URL).reply(200, { success: true, data: populated })
    renderPage()
    expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'true')
  })

  it('shows an error with retry', async () => {
    mock.onGet(URL).replyOnce(500)
    renderPage()
    expect(await screen.findByRole('alert')).toBeInTheDocument()
    mock.onGet(URL).reply(200, { success: true, data: populated })
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText(/Ada/)).toBeInTheDocument()
  })

  it('renders every section when populated', async () => {
    mock.onGet(URL).reply(200, { success: true, data: populated })
    renderPage()
    expect(await screen.findByRole('heading', { level: 1, name: /Ada/ })).toBeInTheDocument()
    expect(screen.getByTestId('day-chip')).toHaveTextContent('Holiday: Founders Day')
    expect(screen.getByText('Welcome to Acme')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Apply leave' })).toHaveAttribute('href', '/leave')
    expect(screen.getAllByRole('link', { name: 'Log hours' })[0]).toHaveAttribute('href', '/status')
    expect(screen.getByText('80%', { selector: 'p' })).toBeInTheDocument()
    expect(within(screen.getByTestId('next-leave')).getByText('Annual Leave')).toBeInTheDocument()
    expect(screen.getByText('Diwali')).toBeInTheDocument()
    expect(screen.getByText('in 9 days')).toBeInTheDocument()
    expect(screen.getByText('Apollo')).toBeInTheDocument()
    expect(screen.getAllByTestId('week-day')).toHaveLength(7)
    expect(screen.getByText('2 working days not logged')).toBeInTheDocument()
    expect(screen.getByText('Pending team approvals')).toBeInTheDocument()
  })

  it('shows empty states and hides the approver tile when pending_approvals is null', async () => {
    mock.onGet(URL).reply(200, { success: true, data: empty })
    renderPage()
    await waitFor(() => expect(screen.getByText('No leave requests yet')).toBeInTheDocument())
    expect(screen.getByText('No upcoming holidays')).toBeInTheDocument()
    expect(screen.getByText('No active projects')).toBeInTheDocument()
    expect(screen.getByText('No hours to show')).toBeInTheDocument()
    expect(screen.queryByText('Pending team approvals')).not.toBeInTheDocument()
    expect(screen.getByTestId('day-chip')).toHaveTextContent('Working day')
  })

  it('renders leave days as on leave, not missing, and labels today as on leave', async () => {
    const days = Array.from({ length: 7 }, (_, i) => ({
      date: `2026-10-${24 + i}`,
      hours: 0,
      is_working_day: true,
      kind: i === 5 ? 'leave' : 'working_day',
    }))
    const data = {
      ...populated,
      greeting: { ...populated.greeting, today: { date: '2026-10-30', kind: 'leave' } },
      week: { days, missing_days: 0 },
    }
    mock.onGet(URL).reply(200, { success: true, data })
    renderPage()
    expect(await screen.findByTestId('day-chip')).toHaveTextContent('On leave today')
    const items = screen.getAllByTestId('week-day')
    expect(items.filter((li) => li.textContent?.includes('On leave'))).toHaveLength(1)
    expect(items[5]).toHaveAttribute('data-kind', 'leave')
    expect(screen.queryByText(/not logged/)).not.toBeInTheDocument()
  })

  it('labels holiday and weekly-off bars so they are not mistaken for missing logs', async () => {
    const kinds = ['working_day', 'holiday', 'weekly_off', 'weekly_off', 'working_day', 'working_day', 'working_day']
    const days = kinds.map((kind, i) => ({ date: `2026-10-${24 + i}`, hours: 0, is_working_day: kind === 'working_day', kind }))
    mock.onGet(URL).reply(200, { success: true, data: { ...populated, week: { days, missing_days: 0 } } })
    renderPage()
    await screen.findAllByTestId('week-day')
    const items = screen.getAllByTestId('week-day')
    expect(items[1]).toHaveTextContent('Holiday')
    expect(items[2]).toHaveTextContent('Off')
    expect(items[0]).not.toHaveTextContent('Holiday')
  })
})
