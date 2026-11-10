import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'

let mock: MockAdapter

const summaries = [
  {
    user_id: 'e1', first_name: 'Ada', last_name: 'Doe', email: 'ada@x.com', department_name: 'Engineering', logged: true, day_status: 'logged',
    total_hours: 7.5, hours_by_project: [{ project_id: 'p1', project_name: 'Apollo', project_code: 'APL', hours: 7.5 }], hours_by_category: [],
  },
  {
    user_id: 'e2', first_name: 'Bob', last_name: 'Roe', email: 'bob@x.com', department_name: 'Sales', logged: false,
    day_status: 'missing', total_hours: 0, hours_by_project: [], hours_by_category: [],
  },
  { user_id: 'e3', first_name: 'Cy', last_name: 'Lee', email: 'cy@x.com', logged: false, day_status: 'on_leave', half_day: false, total_hours: 0 },
  { user_id: 'e4', first_name: 'Di', last_name: 'Ng', email: 'di@x.com', logged: false, day_status: 'missing', half_day: true, total_hours: 0 },
  { user_id: 'e5', first_name: 'Ed', last_name: 'Fox', email: 'ed@x.com', logged: false, day_status: 'holiday', total_hours: 0 },
  { user_id: 'e6', first_name: 'Flo', last_name: 'Gu', email: 'flo@x.com', logged: false, day_status: 'weekly_off', total_hours: 0 },
]

const monthly = {
  month: '2026-10', working_days: 22, total_employees: 2, logged_employees: 1, not_logged_employees: 1, total_hours: 120.5,
  employees: [
    { user_id: 'e1', first_name: 'Ada', last_name: 'Doe', email: 'ada@x.com', department_name: 'Engineering', days_logged: 15, off_days_worked: 2, expected_days: 21, missing_days: 6, leave_days: 1, total_hours: 120.5,
      hours_by_project: [{ project_id: 'p1', project_name: 'Apollo', project_code: 'APL', hours: 120.5 }], hours_by_category: [] },
    { user_id: 'e2', first_name: 'Bob', last_name: 'Roe', email: 'bob@x.com', department_name: 'Sales', days_logged: 21, expected_days: 21, missing_days: 0, leave_days: 0, total_hours: 0,
      hours_by_project: [], hours_by_category: [] },
  ],
}

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function signIn(roles = ['admin']) {
  setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' })
  useAuthStore.setState({ accessToken: 'a', user: { id: 'u1', email: 'admin@x.com', roles } })
}

beforeEach(() => {
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/status-logs/daily').reply(200, { success: true, data: summaries })
  mock.onGet('/api/v1/departments').reply(200, { success: true, data: [{ id: 'd1', name: 'Engineering' }] })
  mock.onGet('/api/v1/projects').reply(200, { success: true, data: [{ id: 'p1', name: 'Apollo', code: 'APL' }] })
  mock.onGet('/api/v1/employees/e1/status-logs').reply(200, {
    success: true,
    data: [{ date: '2026-10-27', total_hours: 7.5, entries: [{ id: 'x1', hours: 7.5, project_name: 'Apollo', project_code: 'APL', notes: 'Built the thing' }] }],
  })
  mock.onGet('/api/v1/employees/e1').reply(200, {
    success: true,
    data: { id: 'e1', first_name: 'Ada', last_name: 'Doe', email: 'ada@x.com', is_active: true, roles: ['employee'], profile: {} },
  })
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
})
afterEach(() => mock.restore())

describe('admin status logs', () => {
  it('lists daily statuses, summary counts, and the nav entry', async () => {
    signIn()
    renderAt('/status-logs')
    const table = await screen.findByRole('table')
    expect(within(table).getByText('Bob Roe')).toBeInTheDocument()
    expect(within(table).getByText('Missing', { exact: true })).toBeInTheDocument()
    expect(within(table).getByText('On leave')).toBeInTheDocument()
    expect(within(table).getByText('Missing (half-day leave)').className).toContain('text-amber-700')
    expect(within(table).getByText('Holiday')).toBeInTheDocument()
    expect(within(table).getByText('Weekly off')).toBeInTheDocument()
    expect(within(table).queryByText('Not logged')).not.toBeInTheDocument()
    // only the missing badge uses the warning tone
    expect(within(table).getByText('Missing', { exact: true }).className).toContain('text-amber-700')
    expect(within(table).getByText('On leave').className).not.toContain('amber')
    expect(within(table).getByText('APL: 7.5h')).toBeInTheDocument()
    expect(screen.getByLabelText('Summary')).toHaveTextContent('Logged1')
    expect(screen.getByLabelText('Summary')).toHaveTextContent('Missing2')
    expect(screen.getByLabelText('Summary')).toHaveTextContent('Leave / off3')
    expect(screen.getByRole('link', { name: 'Status logs' })).toHaveAttribute('href', '/status-logs')
  })

  it('filters to missing only and sends department/project filters', async () => {
    signIn()
    renderAt('/status-logs')
    await screen.findByRole('table')
    await userEvent.click(screen.getByLabelText('Missing only'))
    expect(screen.queryByText('Ada Doe')).not.toBeInTheDocument()
    expect(screen.queryByText('Cy Lee')).not.toBeInTheDocument()
    expect(screen.getByText('Bob Roe')).toBeInTheDocument()
    await screen.findByRole('option', { name: 'Engineering' })
    await userEvent.selectOptions(screen.getByLabelText('Department'), 'd1')
    await waitFor(() =>
      expect(mock.history.get.some((r) => r.url === '/api/v1/status-logs/daily' && r.params?.department === 'd1')).toBe(true),
    )
  })

  it('drills down to an employee\'s entries with notes', async () => {
    signIn()
    renderAt('/status-logs')
    await userEvent.click(await screen.findByRole('button', { name: 'View entries for Ada Doe' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Built the thing')).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows a Status logs tab on the employee detail page', async () => {
    signIn()
    renderAt('/employees/e1/status-logs')
    const toggle = await screen.findByRole('button', { name: /27/ })
    expect(screen.queryByText('Built the thing')).not.toBeInTheDocument()
    await userEvent.click(toggle)
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Built the thing')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Status logs', current: 'page' })).toBeInTheDocument()
  })

  describe('monthly view', () => {
    beforeEach(() => {
      mock.onGet('/api/v1/status-logs/monthly').reply(200, { success: true, data: monthly })
    })

    it('shows summary cards, logged vs expected, leave and missing', async () => {
      signIn()
      renderAt('/status-logs')
      await userEvent.click(screen.getByRole('button', { name: 'Monthly' }))
      const table = await screen.findByRole('table')
      expect(within(table).getByText('15/21 logged')).toBeInTheDocument()
      expect(within(table).getByText('21/21 logged')).toBeInTheDocument()
      expect(within(table).queryByText('6 missing')).not.toBeInTheDocument()
      expect(within(table).getByText('Missing logs')).toBeInTheDocument()
      expect(within(table).getByText('15/21 logged').className).toContain('text-amber-700')
      expect(within(table).getByText('21/21 logged').className).not.toContain('amber')
      expect(within(table).getByText('APL: 120.5h')).toBeInTheDocument()
      const summary = screen.getByLabelText('Monthly summary')
      expect(summary).toHaveTextContent('Working days22')
      expect(summary).toHaveTextContent('Employees with missing logs1')
      expect(summary).toHaveTextContent('Total hours120.5h')
    })

    it('opens a details modal from the info icon with the breakdown', async () => {
      signIn()
      renderAt('/status-logs')
      await userEvent.click(screen.getByRole('button', { name: 'Monthly' }))
      const table = await screen.findByRole('table')
      await userEvent.click(within(table).getAllByTitle('View details')[0])
      const dialog = await screen.findByRole('dialog')
      expect(within(dialog).getByText('Missing').nextSibling).toHaveTextContent('6')
      expect(within(dialog).getByText('On leave').nextSibling).toHaveTextContent('1')
      expect(within(dialog).getByText('Logged on off-days').nextSibling).toHaveTextContent('2')
      await userEvent.click(within(dialog).getByRole('button', { name: 'Close' }))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('filters missing logs only, sends filters, and links to the prefilled employee range', async () => {
      signIn()
      renderAt('/status-logs')
      await userEvent.click(screen.getByRole('button', { name: 'Monthly' }))
      const link = await screen.findByRole('link', { name: 'Ada Doe' })
      expect(link.getAttribute('href')).toMatch(/^\/employees\/e1\/status-logs\?from=\d{4}-\d{2}-01&to=\d{4}-\d{2}-\d{2}$/)
      await userEvent.click(screen.getByLabelText('Missing logs only'))
      expect(screen.queryByText('Bob Roe')).not.toBeInTheDocument()
      expect(screen.getByText('Ada Doe')).toBeInTheDocument()
      await screen.findByRole('option', { name: 'Engineering' })
      await userEvent.selectOptions(screen.getByLabelText('Department'), 'd1')
      await waitFor(() =>
        expect(mock.history.get.some((r) => r.url === '/api/v1/status-logs/monthly' && r.params?.department === 'd1')).toBe(true),
      )
    })

    it('navigates to the previous month', async () => {
      signIn()
      renderAt('/status-logs')
      await userEvent.click(screen.getByRole('button', { name: 'Monthly' }))
      await screen.findByRole('table')
      await userEvent.click(screen.getByRole('button', { name: 'Previous month' }))
      await waitFor(() => expect(mock.history.get.filter((r) => r.url === '/api/v1/status-logs/monthly')).toHaveLength(2))
      expect(screen.getByRole('button', { name: 'Next month' })).toBeEnabled()
    })
  })

  it('prefills the employee tab range from from/to search params', async () => {
    signIn()
    renderAt('/employees/e1/status-logs?from=2026-09-01&to=2026-09-30')
    await screen.findByRole('button', { name: /27/ })
    expect(screen.getByLabelText('From')).toHaveValue('2026-09-01')
    expect(screen.getByLabelText('To')).toHaveValue('2026-09-30')
  })
})
