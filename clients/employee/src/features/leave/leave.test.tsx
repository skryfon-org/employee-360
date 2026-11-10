import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import { leaveRoutes } from './routes'
import { canCancel } from './components/utils'
import { filtersFromParams } from './schemas/filters'
import { buildApplySchema, countWorkingDays, minStartDate } from './schemas/leaveSchemas'

const TYPE_ID = '9a1c2b3d-4e5f-4a6b-8c7d-0e1f2a3b4c5d'
let mock: MockAdapter

const req = (o: Record<string, unknown> = {}) => ({
  id: 'r1', user_id: 'u1', user_name: 'Asha', user_email: 'a@x.com', leave_type_id: TYPE_ID, leave_type_name: 'Casual',
  leave_type_code: 'CL', start_date: '2099-01-05', end_date: '2099-01-06', day_part: 'full', days: 2, reason: 'trip',
  approver_id: 'm1', approver_name: 'Maya', status: 'pending', decision_comment: '', ...o,
})
const list = (data: unknown[], total = data.length) => ({ success: true, data, meta: { page: 1, page_size: 20, total_items: total, total_pages: 1 } })

function setup({ team = list([]), mine = list([]) }: { team?: unknown; mine?: unknown } = {}) {
  mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: [{ id: TYPE_ID, name: 'Casual', code: 'CL', is_active: true }] })
  mock.onGet('/api/v1/me/leave-approvers').reply(200, { success: true, data: [{ user_id: 'm1', name: 'Maya', email: 'm@x.com', kind: 'manager' }] })
  mock.onGet('/api/v1/me/leave-requests').reply(200, mine)
  mock.onGet('/api/v1/leave-requests/team').reply(200, team)
}

function renderPage(url = '/leave') {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[url]}>
        <Routes>{leaveRoutes}</Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  mock = new MockAdapter(apiClient)
})
afterEach(() => mock.restore())

describe('leave schema', () => {
  const base = { leave_type_id: 't', start_date: '2026-10-29', end_date: '2026-10-29', day_part: 'full', reason: '', approver_id: 'a' }
  const now = new Date(2026, 9, 29)
  it('computes min start as today (local date)', () => expect(minStartDate(now)).toBe('2026-10-29'))
  it('rejects a past start or end and has no max', () => {
    const s = buildApplySchema(now)
    expect(s.safeParse({ ...base, start_date: '2026-10-28', end_date: '2026-10-28' }).success).toBe(false)
    expect(s.safeParse({ ...base, start_date: '2026-10-29', end_date: '2026-10-29' }).success).toBe(true)
    expect(s.safeParse({ ...base, start_date: '2026-10-29', end_date: '2026-10-28' }).success).toBe(false)
    expect(s.safeParse({ ...base, start_date: '2031-01-01', end_date: '2031-02-01' }).success).toBe(true)
  })
  it('allows half day only for a single day', () => {
    const s = buildApplySchema(now)
    expect(s.safeParse({ ...base, day_part: 'first_half' }).success).toBe(true)
    expect(s.safeParse({ ...base, day_part: 'first_half', end_date: '2026-10-30' }).success).toBe(false)
  })
})

describe('working days', () => {
  const base = { leave_type_id: 't', start_date: '2026-11-28', end_date: '2026-11-28', day_part: 'full', reason: '', approver_id: 'a' }
  const now = new Date(2026, 9, 29)
  it('skips holiday dates in the count', () => {
    expect(countWorkingDays('2026-11-23', '2026-11-27', 'full', [0, 6], ['2026-11-25'])).toBe(4)
    expect(countWorkingDays('2026-11-27', '2026-11-29', 'full', [0, 6], ['2026-11-27'])).toBe(0)
    expect(countWorkingDays('2026-11-27', '2026-11-27', 'first_half', [0, 6], ['2026-11-27'])).toBe(0)
  })
  it('blocks a half day on a holiday in the schema', () => {
    const b = { leave_type_id: 't', start_date: '2026-11-27', end_date: '2026-11-27', day_part: 'first_half', reason: '', approver_id: 'a' }
    expect(buildApplySchema(new Date(2026, 9, 29), [0, 6], ['2026-11-27']).safeParse(b).success).toBe(false)
    expect(buildApplySchema(new Date(2026, 9, 29), [0, 6], []).safeParse(b).success).toBe(true)
  })
  it('counts 2026-11-26..30 as 3 with Sat/Sun off', () => expect(countWorkingDays('2026-11-26', '2026-11-30', 'full', [0, 6])).toBe(3))
  it('counts a half day as 0.5 and 0 on an off day', () => {
    expect(countWorkingDays('2026-11-27', '2026-11-27', 'first_half', [0, 6])).toBe(0.5)
    expect(countWorkingDays('2026-11-28', '2026-11-28', 'first_half', [0, 6])).toBe(0)
  })
  it('blocks off-day-only selections only when off days are known', () => {
    expect(buildApplySchema(now, [0, 6]).safeParse(base).success).toBe(false)
    expect(buildApplySchema(now, [0, 6]).safeParse({ ...base, end_date: '2026-11-29' }).success).toBe(false)
    expect(buildApplySchema(now, [0, 6]).safeParse({ ...base, day_part: 'second_half' }).success).toBe(false)
    expect(buildApplySchema(now, [0, 6]).safeParse({ ...base, end_date: '2026-11-30' }).success).toBe(true)
    expect(buildApplySchema(now).safeParse(base).success).toBe(true)
  })
})

describe('canCancel', () => {
  const r = (o: Record<string, unknown>) => ({ ...req(o) }) as unknown as Parameters<typeof canCancel>[0]
  it('allows an approved request starting today', () => expect(canCancel(r({ status: 'approved', start_date: '2026-10-30' }), '2026-10-30')).toBe(true))
  it('blocks an approved request that already started', () => expect(canCancel(r({ status: 'approved', start_date: '2026-10-29' }), '2026-10-30')).toBe(false))
})

describe('Leave page', () => {
  it('lists my requests with status and decision comment, and cancels a pending one', async () => {
    setup({ mine: list([req(), req({ id: 'r2', status: 'rejected', decision_comment: 'Busy period', start_date: '2020-01-01', end_date: '2020-01-01' })]) })
    mock.onPost('/api/v1/me/leave-requests/r1/cancel').reply(200, { success: true, data: req({ status: 'cancelled' }) })
    renderPage()
    const row = (await screen.findByText('Busy period')).closest('tr') as HTMLElement
    expect(within(row).getByText('Rejected')).toBeInTheDocument()
    expect(within(row).queryByRole('button', { name: /Cancel/ })).toBeNull()
    await userEvent.click(await screen.findByRole('button', { name: /Cancel Casual request/ }))
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Cancel request' }))
    await waitFor(() => expect(mock.history.post.some((p) => p.url === '/api/v1/me/leave-requests/r1/cancel')).toBe(true))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it('asks for confirmation before cancelling; Keep request closes without calling the API', async () => {
    setup({ mine: list([req()]) })
    mock.onPost('/api/v1/me/leave-requests/r1/cancel').reply(200, { success: true, data: req({ status: 'cancelled' }) })
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: /Cancel Casual request/ }))
    const dlg = await screen.findByRole('dialog')
    expect(within(dlg).getByText('Cancel this leave request?')).toBeInTheDocument()
    expect(within(dlg).getByText('Casual')).toBeInTheDocument()
    expect(within(dlg).getByText('2099-01-05 to 2099-01-06')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
    await userEvent.click(within(dlg).getByRole('button', { name: 'Keep request' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(mock.history.post).toHaveLength(0)
  })

  it('calls cancel exactly once on confirm', async () => {
    setup({ mine: list([req()]) })
    mock.onPost('/api/v1/me/leave-requests/r1/cancel').reply(200, { success: true, data: req({ status: 'cancelled' }) })
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: /Cancel Casual request/ }))
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Cancel request' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(mock.history.post.filter((p) => p.url === '/api/v1/me/leave-requests/r1/cancel')).toHaveLength(1)
  })

  it('renders the server-computed days, including half days', async () => {
    setup({ mine: list([req({ id: 'r3', days: 0.5, day_part: 'first_half', decision_comment: 'Half', start_date: '2099-02-01', end_date: '2099-02-01' })]) })
    renderPage()
    const row = (await screen.findByText('Half')).closest('tr') as HTMLElement
    expect(within(row).getByText('0.5')).toBeInTheDocument()
  })

  it('hides the Team tab when nothing is assigned to the user', async () => {
    setup()
    renderPage()
    await screen.findByText('No leave requests yet')
    expect(screen.queryByRole('tab', { name: 'Team requests' })).toBeNull()
  })

  it('submits an application and surfaces leave_overlap inline', async () => {
    setup()
    mock.onPost('/api/v1/me/leave-requests').reply(422, { success: false, error: { code: 'leave_overlap', message: 'You already have leave on these dates' } })
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await screen.findByRole('option', { name: /Maya/ })
    await userEvent.selectOptions(screen.getByLabelText(/Leave type/), TYPE_ID)
    await userEvent.selectOptions(screen.getByLabelText(/Approver/), 'm1')
    await userEvent.type(screen.getByLabelText(/Start date/), '2099-01-05')
    await userEvent.type(screen.getByLabelText(/End date/), '2099-01-06')
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /Confirm/ }))
    expect(await screen.findByText('You already have leave on these dates')).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ leave_type_id: TYPE_ID, approver_id: 'm1', start_date: '2099-01-05', end_date: '2099-01-06', day_part: 'full' })
  })

  it('navigates back to the list after a successful submit', async () => {
    setup()
    mock.onPost('/api/v1/me/leave-requests').reply(201, { success: true, data: req() })
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await screen.findByRole('option', { name: /Maya/ })
    await userEvent.selectOptions(screen.getByLabelText(/Leave type/), TYPE_ID)
    await userEvent.selectOptions(screen.getByLabelText(/Approver/), 'm1')
    await userEvent.type(screen.getByLabelText(/Start date/), '2099-01-05')
    await userEvent.type(screen.getByLabelText(/End date/), '2099-01-05')
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /Confirm/ }))
    expect(await screen.findByRole('heading', { name: 'Leave' })).toBeInTheDocument()
  })

  it('shows a confirmation summary and only calls the API on confirm', async () => {
    setup()
    mock.onPost('/api/v1/me/leave-requests').reply(201, { success: true, data: req() })
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await screen.findByRole('option', { name: /Maya/ })
    await userEvent.selectOptions(screen.getByLabelText(/Leave type/), TYPE_ID)
    await userEvent.selectOptions(screen.getByLabelText(/Approver/), 'm1')
    await userEvent.type(screen.getByLabelText(/Start date/), '2099-01-05')
    await userEvent.type(screen.getByLabelText(/End date/), '2099-01-06')
    await userEvent.type(screen.getByLabelText(/Reason/), 'Family trip')
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    const dlg = await screen.findByRole('dialog')
    for (const t of ['Casual', '2099-01-05 to 2099-01-06', 'Full day', 'Maya', 'Family trip', /cannot be edited/]) {
      expect(within(dlg).getByText(t)).toBeInTheDocument()
    }
    await userEvent.click(within(dlg).getByRole('button', { name: 'Go back' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(mock.history.post).toHaveLength(0)
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /Confirm/ }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
  })

  it('opens the apply page from the Apply leave button', async () => {
    setup()
    renderPage()
    await userEvent.click(await screen.findByRole('link', { name: 'Apply leave' }))
    expect(await screen.findByRole('heading', { name: 'Apply for leave' })).toBeInTheDocument()
  })

  it('shows client validation for a past start date', async () => {
    setup()
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await userEvent.type(screen.getByLabelText(/Start date/), '2000-01-01')
    await userEvent.type(screen.getByLabelText(/End date/), '2000-01-01')
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    expect(await screen.findByText('Start date cannot be in the past')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })

  it('shows Working days preview and blocks a Saturday-only selection', async () => {
    setup()
    mock.onGet('/api/v1/organization/weekly-off-days').reply(200, { success: true, data: { weekly_off_days: [0, 6] } })
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await screen.findByRole('option', { name: /Maya/ })
    await userEvent.selectOptions(screen.getByLabelText(/Leave type/), TYPE_ID)
    await userEvent.selectOptions(screen.getByLabelText(/Approver/), 'm1')
    await userEvent.type(screen.getByLabelText(/Start date/), '2026-11-26')
    await userEvent.type(screen.getByLabelText(/End date/), '2026-11-30')
    await waitFor(() => expect(screen.getByTestId('working-days-summary')).toHaveTextContent(/Working days:\s*3/))
    await userEvent.clear(screen.getByLabelText(/Start date/))
    await userEvent.clear(screen.getByLabelText(/End date/))
    await userEvent.type(screen.getByLabelText(/Start date/), '2026-11-28')
    await userEvent.type(screen.getByLabelText(/End date/), '2026-11-28')
    await waitFor(() => expect(screen.getByTestId('working-days-summary')).toHaveTextContent(/Working days:\s*0/))
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    expect(await screen.findByText(/only on weekly off days/)).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(mock.history.post).toHaveLength(0)
  })

  it('excludes a mid-range holiday, blocks holiday-only ranges and half days on a holiday', async () => {
    setup()
    mock.onGet('/api/v1/organization/weekly-off-days').reply(200, { success: true, data: { weekly_off_days: [0, 6] } })
    mock.onGet('/api/v1/holidays').reply(200, list([{ id: 'h1', name: 'Diwali', date: '2026-11-27', category_id: 'c', category_is_public: true }]))
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await screen.findByRole('option', { name: /Maya/ })
    await userEvent.selectOptions(screen.getByLabelText(/Leave type/), TYPE_ID)
    await userEvent.selectOptions(screen.getByLabelText(/Approver/), 'm1')
    await userEvent.type(screen.getByLabelText(/Start date/), '2026-11-26')
    await userEvent.type(screen.getByLabelText(/End date/), '2026-11-30')
    await waitFor(() => expect(screen.getByTestId('working-days-summary')).toHaveTextContent(/Working days:\s*2/))
    expect(screen.queryByText('Diwali')).toBeNull()
    const info = screen.getByRole('button', { name: 'Show details' })
    expect(info).toHaveAttribute('aria-expanded', 'false')
    await userEvent.click(info)
    expect(screen.getByRole('button', { name: 'Hide details' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Excluded')).toBeInTheDocument()
    expect(screen.getByText('Diwali')).toBeInTheDocument()
    expect(screen.getByText('Nov 27')).toBeInTheDocument()
    expect(screen.getByText('Weekly off (Sat, Sun)')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hide details' }))
    expect(screen.queryByText('Diwali')).toBeNull()
    expect(screen.queryByText(/Weekly off/)).toBeNull()
    // Holiday (Fri) + weekend only.
    await userEvent.clear(screen.getByLabelText(/Start date/))
    await userEvent.type(screen.getByLabelText(/Start date/), '2026-11-27')
    await userEvent.clear(screen.getByLabelText(/End date/))
    await userEvent.type(screen.getByLabelText(/End date/), '2026-11-29')
    await waitFor(() => expect(screen.getByTestId('working-days-summary')).toHaveTextContent(/Working days:\s*0/))
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    expect(await screen.findByText(/weekly off days or public holidays/)).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })

  it('hides the weekly off row when no off day falls in the range', async () => {
    setup()
    mock.onGet('/api/v1/organization/weekly-off-days').reply(200, { success: true, data: { weekly_off_days: [0, 6] } })
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await userEvent.type(screen.getByLabelText(/Start date/), '2026-11-24')
    await userEvent.type(screen.getByLabelText(/End date/), '2026-11-25')
    await waitFor(() => expect(screen.getByTestId('working-days-summary')).toHaveTextContent(/Working days:\s*2/))
    expect(screen.queryByRole('button', { name: /details/ })).toBeNull()
    expect(screen.queryByText(/Weekly off/)).toBeNull()
  })

  it('does not exclude an optional (non-public) holiday mid-range', async () => {
    setup()
    mock.onGet('/api/v1/organization/weekly-off-days').reply(200, { success: true, data: { weekly_off_days: [0, 6] } })
    mock.onGet('/api/v1/holidays').reply(200, list([{ id: 'h1', name: 'Optional Day', date: '2026-11-27', category_id: 'c', category_is_public: false }]))
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await userEvent.type(screen.getByLabelText(/Start date/), '2026-11-26')
    await userEvent.type(screen.getByLabelText(/End date/), '2026-11-30')
    await waitFor(() => expect(screen.getByTestId('working-days-summary')).toHaveTextContent(/Working days:\s*3/))
    expect(screen.queryByText('Diwali')).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Show details' }))
    expect(screen.getByText('Weekly off (Sat, Sun)')).toBeInTheDocument()
    expect(screen.queryByText('Optional Day')).toBeNull()
  })

  it('allows a range consisting only of an optional holiday', async () => {
    setup()
    mock.onGet('/api/v1/organization/weekly-off-days').reply(200, { success: true, data: { weekly_off_days: [0, 6] } })
    mock.onGet('/api/v1/holidays').reply(200, list([{ id: 'h1', name: 'Optional Day', date: '2026-11-27', category_id: 'c', category_is_public: false }]))
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await userEvent.type(screen.getByLabelText(/Start date/), '2026-11-27')
    await userEvent.type(screen.getByLabelText(/End date/), '2026-11-27')
    await waitFor(() => expect(screen.getByTestId('working-days-summary')).toHaveTextContent(/Working days:\s*1/))
    expect(screen.queryByText(/weekly off days or public holidays/)).toBeNull()
  })

  it('fetches both years for a range crossing a year boundary', async () => {
    setup()
    mock.onGet('/api/v1/organization/weekly-off-days').reply(200, { success: true, data: { weekly_off_days: [0, 6] } })
    mock.onGet('/api/v1/holidays').reply((c) =>
      [200, list(c.params?.year === 2027 ? [{ id: 'h2', name: 'New Year', date: '2027-01-01', category_id: 'c' }] : [])],
    )
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await userEvent.type(screen.getByLabelText(/Start date/), '2099-12-30')
    await userEvent.type(screen.getByLabelText(/End date/), '2100-01-02')
    await waitFor(() => expect(mock.history.get.filter((r) => r.url === '/api/v1/holidays').map((r) => r.params.year).sort()).toEqual([2099, 2100]))
  })

  it('shows the server leave_start_too_old message under the start date', async () => {
    setup()
    mock.onPost('/api/v1/me/leave-requests').reply(422, { success: false, error: { code: 'leave_start_too_old', message: 'start date cannot be in the past' } })
    renderPage('/leave/apply')
    await screen.findByRole('option', { name: 'Casual' })
    await screen.findByRole('option', { name: /Maya/ })
    await userEvent.selectOptions(screen.getByLabelText(/Leave type/), TYPE_ID)
    await userEvent.selectOptions(screen.getByLabelText(/Approver/), 'm1')
    await userEvent.type(screen.getByLabelText(/Start date/), '2099-01-05')
    await userEvent.type(screen.getByLabelText(/End date/), '2099-01-05')
    await userEvent.click(screen.getByRole('button', { name: 'Submit request' }))
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /Confirm/ }))
    expect(await screen.findByText('start date cannot be in the past')).toBeInTheDocument()
    expect(screen.getByLabelText(/Start date/)).toHaveAttribute('aria-invalid', 'true')
  })

  it('shows Team tab with pending queue and approves with a comment', async () => {
    const pending = req({ id: 'r9', user_name: 'Ravi' })
    mock.onGet('/api/v1/leave-requests/team').reply((c) => {
      const s = c.params?.status
      return [200, s === 'pending' ? list([pending]) : list([pending, req({ id: 'r8', user_name: 'Old', status: 'approved' })])]
    })
    mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-approvers').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-requests').reply(200, list([]))
    mock.onPost('/api/v1/leave-requests/r9/approve').reply(200, { success: true, data: req({ status: 'approved' }) })
    renderPage('/leave?tab=team')
    await screen.findByRole('tab', { name: 'Team requests' })
    await userEvent.click(await screen.findByRole('button', { name: /Approve request from Ravi/ }))
    const dlg = await screen.findByRole('dialog')
    await userEvent.type(within(dlg).getByLabelText(/Comment/), 'Enjoy')
    await userEvent.click(within(dlg).getByRole('button', { name: 'Approve' }))
    await waitFor(() => expect(mock.history.post.length).toBe(1))
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ comment: 'Enjoy' })
    expect(screen.getByRole('button', { name: 'History', expanded: false })).toBeInTheDocument()
  })

  it('shows pending with a count and a collapsed History by default', async () => {
    setup({ team: list([req({ id: 'r9', user_name: 'Ravi' })]) })
    renderPage('/leave?tab=team')
    expect(await screen.findByText('Ravi')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /Pending approval/ })).toHaveTextContent('1')
    const toggle = screen.getByRole('button', { name: 'History' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('search')).toBeNull()
    expect(mock.history.get.some((g) => g.params?.exclude_status === 'pending')).toBe(false)
  })

  it('expands History to show filters and non-pending rows, and collapses it again', async () => {
    mock.onGet('/api/v1/leave-requests/team').reply((c) =>
      c.params?.exclude_status === 'pending'
        ? [200, list([req({ id: 'r8', user_name: 'Old', status: 'approved' })])]
        : [200, list([req({ id: 'r9', user_name: 'Ravi' })])],
    )
    mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-approvers').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-requests').reply(200, list([]))
    renderPage('/leave?tab=team')
    await screen.findByText('Ravi')
    await userEvent.click(screen.getByRole('button', { name: 'History' }))
    expect(await screen.findByText('Old')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'History' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('search')).toBeInTheDocument()
    expect(screen.getByText('Ravi')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'History' }))
    expect(screen.queryByRole('search')).toBeNull()
    expect(screen.queryByText('Old')).toBeNull()
  })

  it('auto-expands History when a filter is in the URL', async () => {
    setup({ team: list([req({ id: 'r8', user_name: 'Old', status: 'approved' })]) })
    renderPage('/leave?tab=team&status=approved')
    expect(await screen.findByRole('button', { name: 'History', expanded: true })).toBeInTheDocument()
    expect(await screen.findByText('Old')).toBeInTheDocument()
  })

  it('shows "No pending requests." when the pending queue is empty', async () => {
    mock.onGet('/api/v1/leave-requests/team').reply((c) => [200, c.params?.status === 'pending' ? list([]) : list([req({ id: 'r8', user_name: 'Old', status: 'approved' })])])
    mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-approvers').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-requests').reply(200, list([]))
    renderPage('/leave?tab=team')
    expect(await screen.findByText('No pending requests.')).toBeInTheDocument()
    expect(screen.queryByText('No requests match your filters.')).toBeNull()
  })

  it('shows the filters empty state in team history when a filter is set', async () => {
    mock.onGet('/api/v1/leave-requests/team').reply((c) => [200, c.params?.status === 'approved' ? list([]) : list([req({ id: 'r8', user_name: 'Old', status: 'approved' })])])
    mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-approvers').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-requests').reply(200, list([]))
    renderPage('/leave?tab=team&view=history&status=approved')
    expect(await screen.findByText('No requests match your filters.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Clear filters' })).toBeInTheDocument()
  })

  it('team history "All" asks the server to exclude pending and shows no pending rows', async () => {
    mock.onGet('/api/v1/leave-requests/team').reply((c) =>
      c.params?.exclude_status === 'pending'
        ? [200, list([req({ id: 'r8', user_name: 'Old', status: 'approved' })])]
        : [200, list([req({ id: 'r9', user_name: 'Ravi' })])],
    )
    mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-approvers').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-requests').reply(200, list([]))
    renderPage('/leave?tab=team&view=history')
    const results = await screen.findByRole('region', { name: 'History' })
    expect(await within(results).findByText('Old')).toBeInTheDocument()
    expect(within(results).queryByText('Ravi')).toBeNull()
    expect(mock.history.get.some((g) => g.url === '/api/v1/leave-requests/team' && g.params?.exclude_status === 'pending' && g.params?.status === undefined)).toBe(true)
  })
})

describe('Leave filters', () => {
  it('sends status and a picked date range to my requests, resets to page 1, and clears', async () => {
    setup({ mine: list([req()]) })
    renderPage('/leave?from=2099-01-10&to=2099-01-20')
    await screen.findByText('Casual', { selector: 'td' })
    expect(screen.getByLabelText('Dates')).toHaveValue('10 Jan 2099 - 20 Jan 2099')
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'approved')
    await waitFor(() => {
      const last = mock.history.get.filter((g) => g.url === '/api/v1/me/leave-requests').at(-1)
      expect(last?.params).toMatchObject({ status: 'approved', from: '2099-01-10', to: '2099-01-20', page: 1 })
    })
    await userEvent.click(screen.getByRole('button', { name: 'Clear' }))
    await waitFor(() => {
      const last = mock.history.get.filter((g) => g.url === '/api/v1/me/leave-requests').at(-1)
      expect(last?.params.status).toBeUndefined()
      expect(last?.params.from).toBeUndefined()
    })
    expect(screen.getByLabelText('Dates')).toHaveValue('')
  })

  it('picks a range from the calendar popover and clears it', async () => {
    setup({ mine: list([req()]) })
    renderPage('/leave?from=2099-01-10&to=2099-01-10')
    await screen.findByText('Casual', { selector: 'td' })
    const input = screen.getByLabelText('Dates')
    expect(input).toHaveAttribute('placeholder', 'Select date range')
    await userEvent.click(input)
    await userEvent.click(await screen.findByRole('gridcell', { name: /January 12th, 2099/ }))
    await userEvent.click(await screen.findByRole('gridcell', { name: /January 15th, 2099/ }))
    await waitFor(() => {
      const last = mock.history.get.filter((g) => g.url === '/api/v1/me/leave-requests').at(-1)
      expect(last?.params).toMatchObject({ from: '2099-01-12', to: '2099-01-15' })
    })
    expect(screen.getByLabelText('Dates')).toHaveValue('12 Jan 2099 - 15 Jan 2099')
    await userEvent.click(screen.getByRole('button', { name: 'Clear date range' }))
    await waitFor(() => {
      const last = mock.history.get.filter((g) => g.url === '/api/v1/me/leave-requests').at(-1)
      expect(last?.params.from).toBeUndefined()
    })
    expect(screen.getByLabelText('Dates')).toHaveValue('')
  })

  it('reads filters from the URL', async () => {
    setup({ mine: list([req()]) })
    renderPage('/leave?status=rejected&from=2099-02-01')
    await waitFor(() => {
      const last = mock.history.get.filter((g) => g.url === '/api/v1/me/leave-requests').at(-1)
      expect(last?.params).toMatchObject({ status: 'rejected', from: '2099-02-01' })
    })
    expect(screen.getByLabelText('Status')).toHaveValue('rejected')
  })

  it('shows a filter-specific empty state', async () => {
    setup({ mine: list([]) })
    renderPage('/leave?status=cancelled')
    expect(await screen.findByText('No requests match your filters')).toBeInTheDocument()
  })

  it('applies the filters to team history', async () => {
    mock.onGet('/api/v1/leave-requests/team').reply(200, list([req({ id: 'r8', user_name: 'Old', status: 'approved' })]))
    mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-approvers').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/me/leave-requests').reply(200, list([]))
    renderPage('/leave?tab=team&view=history&status=approved&from=2099-01-01&to=2099-01-31')
    await screen.findByText('Old')
    expect(mock.history.get.some((g) => g.url === '/api/v1/leave-requests/team' && g.params?.status === 'approved' && g.params?.from === '2099-01-01' && g.params?.to === '2099-01-31')).toBe(true)
    expect(within(screen.getByRole('search')).queryByRole('option', { name: 'Pending' })).toBeNull()
  })

  it('filters by leave type: sends leave_type_id, reads it from the URL, and Clear resets it', async () => {
    setup({ mine: list([req()]) })
    const id = '3f2b8c1e-5a4d-4e6f-9b7a-1c2d3e4f5a6b'
    renderPage()
    await screen.findByRole('option', { name: 'Casual' })
    await userEvent.selectOptions(screen.getByLabelText('Leave type'), TYPE_ID)
    await waitFor(() => {
      const last = mock.history.get.filter((g) => g.url === '/api/v1/me/leave-requests').at(-1)
      expect(last?.params).toMatchObject({ leave_type_id: TYPE_ID, page: 1 })
    })
    await userEvent.click(screen.getByRole('button', { name: 'Clear' }))
    await waitFor(() => {
      const last = mock.history.get.filter((g) => g.url === '/api/v1/me/leave-requests').at(-1)
      expect(last?.params.leave_type_id).toBeUndefined()
    })
    expect(screen.getByLabelText('Leave type')).toHaveValue('')
    expect(filtersFromParams(new URLSearchParams(`leave_type_id=${id}`)).leaveTypeId).toBe(id)
    expect(filtersFromParams(new URLSearchParams('leave_type_id=not-a-uuid')).leaveTypeId).toBe('')
  })

  it('shows the filter empty state for an unknown leave type id', async () => {
    setup({ mine: list([]) })
    renderPage('/leave?leave_type_id=3f2b8c1e-5a4d-4e6f-9b7a-1c2d3e4f5a6b')
    expect(await screen.findByText('No requests match your filters')).toBeInTheDocument()
    expect(await screen.findByRole('option', { name: 'Unknown leave type' })).toBeInTheDocument()
  })

  it('applies the leave type filter to team history', async () => {
    const id = '3f2b8c1e-5a4d-4e6f-9b7a-1c2d3e4f5a6b'
    setup({ team: list([req({ id: 'r8', user_name: 'Old', status: 'approved' })]) })
    renderPage(`/leave?tab=team&view=history&leave_type_id=${id}`)
    await screen.findByText('Old')
    expect(mock.history.get.some((g) => g.url === '/api/v1/leave-requests/team' && g.params?.leave_type_id === id)).toBe(true)
  })
})
