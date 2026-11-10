import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import { openRowMenu, pickRowAction } from '../../../test/rowActions'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'
import { toast } from 'sonner'

let mock: MockAdapter
const POS_ID = '33333333-3333-4333-8333-333333333333'
const DEPT_ID = '22222222-2222-4222-8222-222222222222'
const ROLE_ID = '11111111-1111-4111-8111-111111111111'

const inv = (id: string, email: string, status: string) => ({ id, email, status, created_at: '2026-01-01T00:00:00Z' })
let rows: Record<string, unknown>[] = [inv('i1', 'p@x.com', 'pending'), inv('i2', 'a@x.com', 'accepted'), inv('i3', 'r@x.com', 'revoked'), inv('i4', 'e@x.com', 'expired')]

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function signIn() {
  setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' })
  useAuthStore.setState({
    accessToken: 'a',
    user: { id: 'u1', email: 'admin@x.com', roles: ['admin'] },
  })
}

beforeEach(() => {
  rows = [inv('i1', 'p@x.com', 'pending'), inv('i2', 'a@x.com', 'accepted'), inv('i3', 'r@x.com', 'revoked'), inv('i4', 'e@x.com', 'expired')]
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/users/invitations').reply(() => [200, { success: true, data: rows, meta: { total_pages: 1 } }])
  mock.onGet('/api/v1/roles').reply(200, { success: true, data: [{ id: ROLE_ID, name: 'admin' }] })
  mock.onGet('/api/v1/departments').reply(200, { success: true, data: [{ id: DEPT_ID, name: 'Engineering', is_active: true }], meta: { page: 1, page_size: 100, total_items: 1, total_pages: 1 } })
  mock.onGet('/api/v1/positions').reply(200, { success: true, data: [{ id: POS_ID, name: 'Engineer', is_active: true }], meta: { page: 1, page_size: 100, total_items: 1, total_pages: 1 } })
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
  toast.dismiss()
})
afterEach(() => mock.restore())

describe('invitations', () => {
  it('redirects unauthenticated users to login', () => {
    renderAt('/invitations')
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
  })

  it('shows a skeleton while pending, then replaces it with rows', async () => {
    signIn()
    renderAt('/invitations')
    const region = screen.getByRole('status')
    expect(region).toHaveAttribute('aria-busy', 'true')
    expect(region).toHaveTextContent('Loading invitations…')
    expect(await screen.findByText('p@x.com')).toBeInTheDocument()
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('shows resend/revoke only for pending invitations', async () => {
    signIn()
    renderAt('/invitations')
    expect(await screen.findByText('p@x.com')).toBeInTheDocument()
    await openRowMenu(/Actions for invitation for p@x.com/)
    expect(screen.getAllByRole('menuitem', { name: 'Resend' })).toHaveLength(1)
    expect(screen.getAllByRole('menuitem', { name: 'Revoke' })).toHaveLength(1)
    expect(within(screen.getByRole('table')).getByText('Accepted')).toBeInTheDocument()
  })

  it('shows resend/revoke for expired and no actions for accepted/revoked', async () => {
    signIn()
    renderAt('/invitations')
    expect(await screen.findByText('e@x.com')).toBeInTheDocument()
    await openRowMenu(/Actions for invitation for e@x.com/)
    expect(screen.getAllByRole('menuitem', { name: 'Resend' })).toHaveLength(1)
    expect(screen.getAllByRole('menuitem', { name: 'Revoke' })).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: /Actions for invitation for/ })).toHaveLength(2)
    expect(screen.queryByRole('button', { name: /Actions for invitation for a@x.com/ })).toBeNull()
    expect(screen.queryByRole('button', { name: /Actions for invitation for r@x.com/ })).toBeNull()
    expect(within(screen.getByRole('table')).getByText('Expired')).toBeInTheDocument()
  })

  it('resends an expired invitation with a success toast', async () => {
    signIn()
    mock.onPost('/api/v1/users/invitations/i4/resend').reply(200, { success: true, data: rows[3] })
    renderAt('/invitations')
    await pickRowAction(/Actions for invitation for e@x.com/, 'Resend')
    expect(await screen.findByText('Invitation resent to e@x.com.')).toBeInTheDocument()
  })

  it('renders role, invited by and invited on columns', async () => {
    signIn()
    rows = [
      { ...inv('i1', 'p@x.com', 'pending'), role: 'admin', invited_by: { id: 'u1', name: 'Alice Admin', email: 'alice@x.com' }, invited_on: '2026-02-03T00:00:00Z' },
    ]
    renderAt('/invitations')
    expect(await screen.findByText('Alice Admin')).toBeInTheDocument()
    const table = screen.getByRole('table')
    for (const h of ['Email', 'Role', 'Invited by', 'Invited on', 'Status']) {
      expect(within(table).getByRole('columnheader', { name: h })).toBeInTheDocument()
    }
    expect(within(table).getByText('admin')).toBeInTheDocument()
  })

  it('sends status filter and debounced search, resetting to page 1', async () => {
    signIn()
    renderAt('/invitations?page=3')
    await screen.findByText('p@x.com')
    expect(mock.history.get.find((r) => r.url === '/api/v1/users/invitations')?.params).toMatchObject({ page: 3, page_size: 20 })
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'pending')
    await waitForCondition(() => mock.history.get.some((r) => r.params?.status === 'pending' && r.params?.page === 1))
    await userEvent.type(screen.getByLabelText('Search'), 'p@x')
    await waitForCondition(() => mock.history.get.some((r) => r.params?.search === 'p@x'))
    const searches = mock.history.get.filter((r) => r.params?.search)
    expect(searches.every((r) => r.params.search === 'p@x')).toBe(true)
  })

  it('shows range, paginates and changes page size', async () => {
    signIn()
    mock.onGet('/api/v1/users/invitations').reply((cfg) => [
      200,
      { success: true, data: rows, meta: { page: cfg.params.page, page_size: cfg.params.page_size, total_items: 45, total_pages: 3 } },
    ])
    renderAt('/invitations')
    expect(await screen.findByText('Showing 1–20 of 45')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Showing 21–40 of 45')).toBeInTheDocument()
    await userEvent.selectOptions(screen.getByLabelText('Rows per page'), '50')
    await waitForCondition(() => mock.history.get.some((r) => r.params?.page_size === 50 && r.params?.page === 1))
  })

  it('shows an empty state when filters match nothing and an error state on failure', async () => {
    signIn()
    mock.onGet('/api/v1/users/invitations').reply(200, { success: true, data: [], meta: { total_items: 0, total_pages: 0 } })
    renderAt('/invitations?status=expired')
    expect(await screen.findByText('No invitations match your filters')).toBeInTheDocument()
  })

  it('shows an error when the list fails to load', async () => {
    signIn()
    mock.onGet('/api/v1/users/invitations').reply(500, { success: false, error: { message: 'List exploded' } })
    renderAt('/invitations')
    expect(await screen.findByText('List exploded')).toBeInTheDocument()
  })

  it('revokes after confirmation and refreshes the list', async () => {
    signIn()
    mock.onDelete('/api/v1/users/invitations/i1').reply(() => {
      rows = rows.map((r) => (r.id === 'i1' ? { ...r, status: 'revoked' } : r))
      return [200, { success: true }]
    })
    renderAt('/invitations')
    await pickRowAction(/Actions for invitation for p@x.com/, 'Revoke')
    const dialog = screen.getByRole('dialog')
    expect(mock.history.delete).toHaveLength(0)
    await userEvent.click(within(dialog).getByRole('button', { name: 'Revoke' }))
    await waitForRevokeDone()
    expect(mock.history.delete).toHaveLength(1)
    expect(await screen.findByText('Invitation for p@x.com revoked.')).toBeInTheDocument()
  })

  it('resends and invalidates the list', async () => {
    signIn()
    mock.onPost('/api/v1/users/invitations/i1/resend').reply(200, { success: true, data: rows[0] })
    renderAt('/invitations')
    await pickRowAction(/Actions for invitation for p@x.com/, 'Resend')
    await waitForCondition(() => mock.history.get.filter((r) => r.url === '/api/v1/users/invitations').length >= 2)
    expect(mock.history.post).toHaveLength(1)
    expect(await screen.findByText('Invitation resent to p@x.com.')).toBeInTheDocument()
  })

  it('list page has a Create invitation button that navigates and no inline form', async () => {
    signIn()
    renderAt('/invitations')
    expect(screen.queryByRole('form', { name: 'Invite user' })).toBeNull()
    await userEvent.click(await screen.findByRole('link', { name: 'Create invitation' }))
    expect(await screen.findByRole('heading', { name: 'Create invitation' })).toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Invite user' })).toBeInTheDocument()
    const sidebar = screen.getByRole('complementary', { name: 'Sidebar' })
    expect(within(sidebar).getByRole('link', { name: 'Invitations' })).toHaveAttribute('aria-current', 'page')
    const crumbs = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(within(crumbs).getByRole('link', { name: 'Invitations' })).toHaveAttribute('href', '/invitations')
  })

  it('redirects unauthenticated users from /invitations/new to login', () => {
    renderAt('/invitations/new')
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
  })

  it('cancel returns to the list', async () => {
    signIn()
    renderAt('/invitations/new')
    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))
    expect(await screen.findByRole('link', { name: 'Create invitation' })).toBeInTheDocument()
  })

  it('validates, submits the invite form, and redirects to the list', async () => {
    signIn()
    mock.onPost('/api/v1/users/invitations').reply(201, { success: true, data: rows[0] })
    renderAt('/invitations/new')
    await userEvent.click(await screen.findByRole('button', { name: 'Send invitation' }))
    expect(await screen.findByText('Email is required')).toBeInTheDocument()
    expect(screen.getAllByRole('alert').map((a) => a.textContent)).toContain('Select a role')
    await userEvent.type(screen.getByLabelText('Email'), 'new@x.com')
    await screen.findByRole('option', { name: 'admin' })
    await userEvent.selectOptions(screen.getByLabelText('Role'), ROLE_ID)
    await userEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
    expect(await screen.findByRole('link', { name: 'Create invitation' })).toBeInTheDocument()
    expect(await screen.findByText('p@x.com')).toBeInTheDocument()
    expect(await screen.findByText('Invitation sent to new@x.com.')).toBeInTheDocument()
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ email: 'new@x.com', role_id: ROLE_ID })
  })

  it('shows EMAIL_DOMAIN_NOT_ALLOWED inline on the email field without a toast', async () => {
    signIn()
    const msg = 'Email domain is not registered for this organization'
    mock.onPost('/api/v1/users/invitations').reply(400, { success: false, error: { code: 'EMAIL_DOMAIN_NOT_ALLOWED', message: msg } })
    renderAt('/invitations/new')
    await userEvent.type(await screen.findByLabelText('Email'), 'new@other.com')
    await screen.findByRole('option', { name: 'admin' })
    await userEvent.selectOptions(screen.getByLabelText('Role'), ROLE_ID)
    await userEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
    expect(await screen.findByText(msg)).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('form', { name: 'Invite user' })).toBeInTheDocument()
  })

  it('loads department options from the active filter and sends the selected department_id', async () => {
    signIn()
    mock.onPost('/api/v1/users/invitations').reply(201, { success: true, data: rows[0] })
    renderAt('/invitations/new')
    await screen.findByRole('option', { name: 'Engineering' })
    const req = mock.history.get.find((r) => r.url === '/api/v1/departments')
    expect(req?.params).toMatchObject({ is_active: true, page_size: 100 })
    await userEvent.type(screen.getByLabelText('Email'), 'new@x.com')
    await screen.findByRole('option', { name: 'admin' })
    await userEvent.selectOptions(screen.getByLabelText('Role'), ROLE_ID)
    await userEvent.selectOptions(screen.getByLabelText('Department (optional)'), DEPT_ID)
    await userEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ email: 'new@x.com', role_id: ROLE_ID, department_id: DEPT_ID })
  })

  it('maps DEPARTMENT_INACTIVE to the department field and refetches options', async () => {
    signIn()
    const msg = 'Department is inactive'
    mock.onPost('/api/v1/users/invitations').reply(400, { success: false, error: { code: 'DEPARTMENT_INACTIVE', message: msg } })
    renderAt('/invitations/new')
    await screen.findByRole('option', { name: 'Engineering' })
    await userEvent.type(screen.getByLabelText('Email'), 'new@x.com')
    await screen.findByRole('option', { name: 'admin' })
    await userEvent.selectOptions(screen.getByLabelText('Role'), ROLE_ID)
    await userEvent.selectOptions(screen.getByLabelText('Department (optional)'), DEPT_ID)
    const before = mock.history.get.filter((r) => r.url === '/api/v1/departments').length
    await userEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
    expect(await screen.findByText(msg)).toBeInTheDocument()
    expect(screen.getByLabelText('Department (optional)')).toHaveAttribute('aria-invalid', 'true')
    await waitFor(() => expect(mock.history.get.filter((r) => r.url === '/api/v1/departments').length).toBeGreaterThan(before))
  })

  it('shows an empty state linking to departments when none are active', async () => {
    signIn()
    mock.onGet('/api/v1/departments').reply(200, { success: true, data: [], meta: { page: 1, page_size: 100, total_items: 0, total_pages: 0 } })
    renderAt('/invitations/new')
    const link = await screen.findByRole('link', { name: 'Create one in Departments' })
    expect(link).toHaveAttribute('href', '/departments')
  })

  it('sends the selected position_id from active positions', async () => {
    signIn()
    mock.onPost('/api/v1/users/invitations').reply(201, { success: true, data: rows[0] })
    renderAt('/invitations/new')
    await screen.findByRole('option', { name: 'Engineer' })
    const req = mock.history.get.find((r) => r.url === '/api/v1/positions')
    expect(req?.params).toMatchObject({ is_active: true, page_size: 100 })
    await userEvent.type(screen.getByLabelText('Email'), 'new@x.com')
    await screen.findByRole('option', { name: 'admin' })
    await userEvent.selectOptions(screen.getByLabelText('Role'), ROLE_ID)
    await userEvent.selectOptions(screen.getByLabelText('Position (optional)'), POS_ID)
    await userEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ email: 'new@x.com', role_id: ROLE_ID, position_id: POS_ID })
  })

  it('maps POSITION_INACTIVE to the position field and refetches options', async () => {
    signIn()
    const msg = 'Position is inactive'
    mock.onPost('/api/v1/users/invitations').reply(400, { success: false, error: { code: 'POSITION_INACTIVE', message: msg } })
    renderAt('/invitations/new')
    await screen.findByRole('option', { name: 'Engineer' })
    await userEvent.type(screen.getByLabelText('Email'), 'new@x.com')
    await screen.findByRole('option', { name: 'admin' })
    await userEvent.selectOptions(screen.getByLabelText('Role'), ROLE_ID)
    await userEvent.selectOptions(screen.getByLabelText('Position (optional)'), POS_ID)
    const before = mock.history.get.filter((r) => r.url === '/api/v1/positions').length
    await userEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
    expect(await screen.findByText(msg)).toBeInTheDocument()
    expect(screen.getByLabelText('Position (optional)')).toHaveAttribute('aria-invalid', 'true')
    await waitFor(() => expect(mock.history.get.filter((r) => r.url === '/api/v1/positions').length).toBeGreaterThan(before))
  })

  it('shows an empty state linking to positions when none are active', async () => {
    signIn()
    mock.onGet('/api/v1/positions').reply(200, { success: true, data: [], meta: { page: 1, page_size: 100, total_items: 0, total_pages: 0 } })
    renderAt('/invitations/new')
    const link = await screen.findByRole('link', { name: 'Create one in Positions' })
    expect(link).toHaveAttribute('href', '/positions')
  })

  it('shows an error when departments fail to load', async () => {
    signIn()
    mock.onGet('/api/v1/departments').reply(500, { success: false, error: { message: 'boom' } })
    renderAt('/invitations/new')
    expect(await screen.findByText(/Could not load departments/)).toBeInTheDocument()
  })

  it('shows an error when roles fail to load', async () => {
    signIn()
    mock.onGet('/api/v1/roles').reply(500, { success: false, error: { message: 'boom' } })
    renderAt('/invitations/new')
    expect(await screen.findByText('Could not load roles.')).toBeInTheDocument()
  })

  it('shows an error when resend fails', async () => {
    signIn()
    mock.onPost('/api/v1/users/invitations/i1/resend').reply(500, { success: false, error: { message: 'Resend exploded' } })
    renderAt('/invitations')
    await pickRowAction(/Actions for invitation for p@x.com/, 'Resend')
    expect(await screen.findByText('Resend exploded')).toBeInTheDocument()
  })

  it('shows revoke failure in the dialog and keeps it open', async () => {
    signIn()
    mock.onDelete('/api/v1/users/invitations/i1').reply(500, { success: false, error: { message: 'Revoke exploded' } })
    renderAt('/invitations')
    await pickRowAction(/Actions for invitation for p@x.com/, 'Revoke')
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Revoke' }))
    expect(await within(dialog).findByText('Revoke exploded')).toBeInTheDocument()
    expect(screen.getAllByText('Revoke exploded')).toHaveLength(2)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('modal moves focus in, closes on Escape, and restores focus to the trigger', async () => {
    signIn()
    renderAt('/invitations')
    const trigger = await pickRowAction(/Actions for invitation for p@x.com/, 'Revoke')
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus()
    await userEvent.tab()
    await userEvent.tab()
    expect(dialog.contains(document.activeElement)).toBe(true)
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(trigger).toHaveFocus()
  })
})

async function waitForCondition(cond: () => boolean) {
  await waitFor(() => expect(cond()).toBe(true))
}
async function waitForRevokeDone() {
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  await waitFor(() => expect(screen.queryAllByRole('button', { name: /^Actions for invitation/ })).toHaveLength(1))
}
