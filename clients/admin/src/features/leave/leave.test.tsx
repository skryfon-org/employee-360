import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import { leaveRoutes } from './routes'
import { EmployeeLeaveRequests } from './components/EmployeeLeaveRequests'
import { filtersFromParams, filtersToParams } from './schemas/filters'
import { visibleNavItems } from '../../components/layout/navItems'
import { useAuthStore } from '../../stores/authStore'
import { pickRowAction } from '../../test/rowActions'

const TYPE_ID = '9a1c2b3d-4e5f-4a6b-8c7d-0e1f2a3b4c5d'
let mock: MockAdapter

const req = (o: Record<string, unknown> = {}) => ({
  id: 'r1', user_id: 'u1', user_name: 'Asha', user_email: 'a@x.com', leave_type_id: TYPE_ID, leave_type_name: 'Casual',
  leave_type_code: 'CL', start_date: '2099-01-05', end_date: '2099-01-06', day_part: 'full', days: 2, reason: 'trip',
  approver_id: 'm1', approver_name: 'Maya', decided_by_name: null, status: 'pending', decision_comment: '', ...o,
})
const list = (data: unknown[]) => ({ success: true, data, meta: { page: 1, page_size: 20, total_items: data.length, total_pages: 1 } })
const types = [
  { id: TYPE_ID, name: 'Casual', code: 'CL', description: 'Casual leave', is_active: true },
  { id: 't2', name: 'Sabbatical', code: 'SB', description: '', is_active: false },
]

function renderAt(url: string) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>
      <MemoryRouter initialEntries={[url]}>
        <Routes>{leaveRoutes}</Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  mock = new MockAdapter(apiClient)
  useAuthStore.setState({ user: { id: 'admin1', email: 'admin@x.com', roles: ['admin'] } })
  mock.onGet('/api/v1/leave-types').reply(200, { success: true, data: types })
  mock.onGet('/api/v1/employees').reply(200, list([{ id: 'u1', first_name: 'Asha', last_name: 'K', email: 'a@x.com' }]))
})
afterEach(() => mock.restore())

describe('filters', () => {
  it('defaults to pending and round-trips "all"', () => {
    expect(filtersFromParams(new URLSearchParams()).status).toBe('pending')
    const all = filtersFromParams(new URLSearchParams('status=all'))
    expect(all.status).toBe('')
    expect(filtersToParams(all).get('status')).toBe('all')
    expect(filtersToParams(filtersFromParams(new URLSearchParams())).toString()).toBe('')
  })
  it('swaps an inverted date range and drops bad ids', () => {
    const f = filtersFromParams(new URLSearchParams('from=2026-02-01&to=2026-01-01&employee_id=nope'))
    expect(f).toMatchObject({ from: '2026-01-01', to: '2026-02-01', employeeId: '' })
  })
})

describe('nav', () => {
  it('shows Leave to admins only', () => {
    expect(visibleNavItems(['admin']).some((i) => i.to === '/leave')).toBe(true)
    expect(visibleNavItems(['employee']).some((i) => i.to === '/leave')).toBe(false)
  })
})

describe('LeaveRequestsPage', () => {
  it('queries pending by default and hides actions on the admin own request', async () => {
    mock.onGet('/api/v1/leave-requests').reply(200, list([req(), req({ id: 'r2', user_id: 'admin1', user_name: 'Me Admin' })]))
    renderAt('/leave')
    expect(await screen.findByText('Asha')).toBeInTheDocument()
    expect(mock.history.get.find((g) => g.url === '/api/v1/leave-requests')?.params).toMatchObject({ status: 'pending' })
    expect(screen.queryByRole('button', { name: 'Approve request from Asha' })).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'View request from Asha' }))
    expect(within(screen.getByRole('dialog')).getByText('trip')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Approve request from Asha' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Close' }))
    await userEvent.click(screen.getByRole('button', { name: 'View request from Me Admin' }))
    expect(screen.getByText('Me Admin')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Approve request from Me Admin/ })).toBeNull()
    expect(screen.queryByRole('button', { name: /Reject request from Me Admin/ })).toBeNull()
  })

  it('shows the approver and, separately, who decided (dash when undecided)', async () => {
    mock.onGet('/api/v1/leave-requests').reply(200, list([req({ id: 'r1', user_name: 'Asha' }), req({ id: 'r2', user_name: 'Ben', status: 'approved', decided_by_name: 'Dana Admin' })]))
    renderAt('/leave')
    const rows = await screen.findAllByRole('row')
    const headers = within(rows[0]).getAllByRole('columnheader').map((h) => h.textContent)
    const approverIdx = headers.indexOf('Approver')
    const decidedIdx = headers.indexOf('Decided by')
    expect(approverIdx).toBeGreaterThan(-1)
    expect(decidedIdx).toBe(approverIdx + 1)
    const cells = (name: string) => within(rows.find((r) => within(r).queryByText(name))!).getAllByRole('cell')
    expect(cells('Asha')[approverIdx]).toHaveTextContent('Maya')
    expect(cells('Asha')[decidedIdx]).toHaveTextContent('-')
    expect(cells('Ben')[approverIdx]).toHaveTextContent('Maya')
    expect(cells('Ben')[decidedIdx]).toHaveTextContent('Dana Admin')
  })

  it('rejects with a comment', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/leave-requests').reply(200, list([req()]))
    mock.onPost('/api/v1/leave-requests/r1/reject').reply(200, { success: true, data: req({ status: 'rejected' }) })
    renderAt('/leave')
    await user.click(await screen.findByRole('button', { name: 'View request from Asha' }))
    await user.click(await screen.findByRole('button', { name: 'Reject request from Asha' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText(/Comment/), 'busy period')
    await user.click(within(dialog).getByRole('button', { name: 'Reject' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(mock.history.post[0].url).toBe('/api/v1/leave-requests/r1/reject')
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ comment: 'busy period' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it.each(['/leave', '/leave/types'])('redirects non-admin users away from %s', async (path) => {
    useAuthStore.setState({ user: { id: 'e1', email: 'e@x.com', roles: ['employee'] } })
    mock.onGet('/api/v1/leave-requests').reply(200, list([]))
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            {leaveRoutes}
            <Route path="/" element={<p>dashboard home</p>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(await screen.findByText('dashboard home')).toBeInTheDocument()
    expect(mock.history.get.some((g) => g.url === '/api/v1/leave-requests')).toBe(false)
  })

  it('filters by employee through a server-side search picker', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/leave-requests').reply(200, list([]))
    mock.onGet('/api/v1/employees').reply(200, list([{ id: TYPE_ID, first_name: 'Asha', last_name: 'K', email: 'a@x.com' }]))
    renderAt('/leave')
    await user.type(await screen.findByRole('combobox', { name: 'Employee' }), 'Ash')
    await user.click(await screen.findByRole('option', { name: /Asha K/ }))
    await waitFor(() =>
      expect(mock.history.get.filter((g) => g.url === '/api/v1/leave-requests').some((g) => g.params?.employee_id === TYPE_ID)).toBe(true),
    )
    const search = mock.history.get.find((g) => g.url === '/api/v1/employees')
    expect(search?.params).not.toHaveProperty('status')
    expect(search?.params).not.toHaveProperty('include')
  })

  it('sends filters from the URL', async () => {
    mock.onGet('/api/v1/leave-requests').reply(200, list([]))
    renderAt(`/leave?status=approved&leave_type_id=${TYPE_ID}&employee_id=${TYPE_ID}&from=2099-01-01&to=2099-01-31`)
    expect(await screen.findByText('No matching requests')).toBeInTheDocument()
    expect(mock.history.get.find((g) => g.url === '/api/v1/leave-requests')?.params).toMatchObject({
      status: 'approved', leave_type_id: TYPE_ID, employee_id: TYPE_ID, from: '2099-01-01', to: '2099-01-31',
    })
  })

  it('approves with a comment', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/leave-requests').reply(200, list([req()]))
    mock.onPost('/api/v1/leave-requests/r1/approve').reply(200, { success: true, data: req({ status: 'approved' }) })
    renderAt('/leave')
    await user.click(await screen.findByRole('button', { name: 'View request from Asha' }))
    await user.click(await screen.findByRole('button', { name: 'Approve request from Asha' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText(/Comment/), 'enjoy')
    await user.click(within(dialog).getByRole('button', { name: 'Approve' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ comment: 'enjoy' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it('shows a server error when the decision fails', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/leave-requests').reply(200, list([req()]))
    mock.onPost('/api/v1/leave-requests/r1/reject').reply(409, { success: false, error: { code: 'leave_not_pending', message: 'already decided' } })
    renderAt('/leave')
    await user.click(await screen.findByRole('button', { name: 'View request from Asha' }))
    await user.click(await screen.findByRole('button', { name: 'Reject request from Asha' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Reject' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/already decided/)
  })
})

describe('LeaveTypesPage', () => {
  it('lists active and inactive types and creates one', async () => {
    const user = userEvent.setup()
    mock.onPost('/api/v1/leave-types').reply(201, { success: true, data: { ...types[0], id: 't3' } })
    renderAt('/leave/types')
    expect(await screen.findByText('Sabbatical')).toBeInTheDocument()
    expect(mock.history.get.find((g) => g.url === '/api/v1/leave-types')?.params).toMatchObject({ include_inactive: true })
    await user.click(screen.getByRole('button', { name: 'Add leave type' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Add leave type' }))
    expect(await within(dialog).findByText('Name is required')).toBeInTheDocument()
    await user.type(within(dialog).getByLabelText('Name'), 'Study')
    await user.type(within(dialog).getByLabelText('Code'), 'ST')
    await user.click(within(dialog).getByRole('button', { name: 'Add leave type' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ name: 'Study', code: 'ST', description: '', is_active: true })
  })

  it('maps a duplicate code to a field error', async () => {
    const user = userEvent.setup()
    mock.onPost('/api/v1/leave-types').reply(409, { success: false, error: { code: 'CONFLICT', message: 'code taken' } })
    renderAt('/leave/types')
    await user.click(await screen.findByRole('button', { name: 'Add leave type' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Name'), 'Study')
    await user.type(within(dialog).getByLabelText('Code'), 'CL')
    await user.click(within(dialog).getByRole('button', { name: 'Add leave type' }))
    expect(await within(dialog).findByText('A leave type with this code already exists.')).toBeInTheDocument()
  })

  it('sends is_active false when the Active toggle is switched off on create', async () => {
    const user = userEvent.setup()
    mock.onPost('/api/v1/leave-types').reply(201, { success: true, data: types[1] })
    renderAt('/leave/types')
    await user.click(await screen.findByRole('button', { name: 'Add leave type' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Name'), 'Study')
    await user.type(within(dialog).getByLabelText('Code'), 'ST')
    const toggle = within(dialog).getByRole('switch', { name: /Active/ })
    expect(toggle).toBeChecked()
    await user.click(toggle)
    await user.click(within(dialog).getByRole('button', { name: 'Add leave type' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ is_active: false })
  })

  it('does not map a 409 without the CONFLICT code to the code field', async () => {
    const user = userEvent.setup()
    mock.onPost('/api/v1/leave-types').reply(409, { success: false, error: { code: 'something_else', message: 'other problem' } })
    renderAt('/leave/types')
    await user.click(await screen.findByRole('button', { name: 'Add leave type' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Name'), 'Study')
    await user.type(within(dialog).getByLabelText('Code'), 'CL')
    await user.click(within(dialog).getByRole('button', { name: 'Add leave type' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(within(dialog).queryByText('A leave type with this code already exists.')).toBeNull()
  })

  it('deactivates after confirmation and activates directly', async () => {
    const user = userEvent.setup()
    mock.onPatch(`/api/v1/leave-types/${TYPE_ID}`).reply(200, { success: true, data: types[0] })
    mock.onPatch('/api/v1/leave-types/t2').reply(200, { success: true, data: types[1] })
    renderAt('/leave/types')
    await pickRowAction('Actions for Casual', 'Deactivate')
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Deactivate' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toEqual({ is_active: false })
    await pickRowAction('Actions for Sabbatical', 'Activate')
    await waitFor(() => expect(mock.history.patch).toHaveLength(2))
    expect(JSON.parse(mock.history.patch[1].data)).toEqual({ is_active: true })
  })
})

describe('EmployeeLeaveRequests', () => {
  it('pins the list to the employee and hides the employee column', async () => {
    mock.onGet('/api/v1/leave-requests').reply(200, list([req({ status: 'approved' })]))
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <EmployeeLeaveRequests employeeId="u1" />
      </QueryClientProvider>,
    )
    expect(await screen.findByText('Casual', { selector: 'td' })).toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: 'Employee' })).toBeNull()
    const params = mock.history.get.find((g) => g.url === '/api/v1/leave-requests')?.params
    expect(params).toMatchObject({ employee_id: 'u1' })
    expect(params.status).toBeUndefined()
  })
})
