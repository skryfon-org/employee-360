import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'

const ADMIN_URL = '/api/v1/dashboard/admin'
const SUPER_URL = '/api/v1/dashboard/super-admin'

let mock: MockAdapter

const base = {
  users: { total: 12, active: 9, inactive: 3, pending_invited: 2 },
  invitations: { pending: 2, accepted: 5, expired: 1, revoked: 0 },
  departments: 4,
  positions: 7,
  recent_invitations: [
    {
      id: 'i1',
      email: 'new@x.com',
      role: 'employee',
      invited_by: { id: 'u1', name: 'Ada Admin', email: 'ada@x.com' },
      invited_on: '2026-01-02T00:00:00Z',
      status: 'pending',
    },
  ],
}
const superData = {
  ...base,
  tenant: { id: 't1', name: 'Acme Corp', is_active: true, created_at: '2025-06-01T00:00:00Z' },
  users_by_role: [{ role: 'employee', total: 10, active: 8, inactive: 2 }],
}

function renderAt(path = '/') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function signIn(roles: string[]) {
  setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' })
  useAuthStore.setState({ accessToken: 'a', user: { id: 'u1', email: 'u@x.com', roles } })
}

beforeEach(() => {
  mock = new MockAdapter(apiClient)
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
})
afterEach(() => mock.restore())

const calls = (url: string) => mock.history.get.filter((r) => r.url === url).length

describe('admin dashboard', () => {
  it('renders stat cards and the invite action; never calls the super-admin endpoint', async () => {
    mock.onGet(ADMIN_URL).reply(200, { success: true, data: base })
    signIn(['admin'])
    renderAt()

    expect(screen.getByRole('status')).toHaveTextContent('Loading dashboard…')
    expect(await screen.findByText('Total users')).toBeInTheDocument()
    expect(screen.getByText('Total users').nextSibling).toHaveTextContent('12')
    expect(within(screen.getByRole('main')).getByText('Departments').nextSibling).toHaveTextContent('4')
    expect(screen.queryByText('Invitations by status')).not.toBeInTheDocument()
    expect(screen.queryByText('Recent invitations')).not.toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'Invite user' })[0]).toHaveAttribute('href', '/invitations/new')
    expect(screen.queryByText('Active')).not.toBeInTheDocument()
    expect(screen.queryByText('Organisation')).not.toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Organisation' })).not.toBeInTheDocument()
    expect(calls(SUPER_URL)).toBe(0)
  })

  it('links stat cards to their routes', async () => {
    mock.onGet(ADMIN_URL).reply(200, { success: true, data: base })
    signIn(['admin'])
    renderAt()
    await screen.findByText('Total users')
    const main = within(screen.getByRole('main'))
    expect(main.getByRole('link', { name: /Total users/ })).toHaveAttribute('href', '/employees')
    expect(main.getByRole('link', { name: /Pending invited/ })).toHaveAttribute('href', '/invitations')
    expect(main.getByRole('link', { name: /Departments/ })).toHaveAttribute('href', '/departments')
    expect(main.getByRole('link', { name: /Positions/ })).toHaveAttribute('href', '/positions')
    expect(main.queryByRole('link', { name: /Active users/ })).not.toBeInTheDocument()
    expect(main.getByRole('link', { name: /Total users/ })).toHaveAccessibleName('Total users 12')
  })

  it('shows an error with retry', async () => {
    mock.onGet(ADMIN_URL).replyOnce(500).onGet(ADMIN_URL).reply(200, { success: true, data: base })
    signIn(['admin'])
    renderAt()
    expect(await screen.findByRole('alert')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('Total users')).toBeInTheDocument()
    expect(calls(ADMIN_URL)).toBe(2)
  })

  it('lists Dashboard in the sidebar and serves it at /dashboard', async () => {
    mock.onGet(ADMIN_URL).reply(200, { success: true, data: base })
    signIn(['admin'])
    renderAt('/dashboard')
    expect(await screen.findByText('Total users')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Dashboard' })).toHaveAttribute('href', '/')
  })
})

describe('super admin dashboard', () => {
  it('renders organisation card and stat cards; never calls the admin endpoint', async () => {
    mock.onGet(SUPER_URL).reply(200, { success: true, data: superData })
    signIn(['super_admin'])
    renderAt()

    expect(await screen.findByText('Organisation')).toBeInTheDocument()
    expect(screen.getByText('Acme Corp')).toBeInTheDocument()
    expect(screen.getByText('1 Jun 2025')).toBeInTheDocument()
    expect(within(screen.getByRole('region', { name: 'Organisation' })).getByText('Active')).toBeInTheDocument()
    expect(screen.queryByText('Users by role')).not.toBeInTheDocument()
    expect(screen.getByText('Total users').nextSibling).toHaveTextContent('12')
    expect(calls(ADMIN_URL)).toBe(0)
  })

  it('shows inactive tenant status', async () => {
    mock.onGet(SUPER_URL).reply(200, {
      success: true,
      data: { ...superData, tenant: { ...superData.tenant, is_active: false } },
    })
    signIn(['super_admin'])
    renderAt()
    expect(await screen.findByText('Inactive')).toBeInTheDocument()
  })

  it('shows loading then error state', async () => {
    mock.onGet(SUPER_URL).reply(500)
    signIn(['super_admin'])
    renderAt()
    expect(screen.getByRole('status')).toHaveTextContent('Loading dashboard…')
    expect(await screen.findByRole('alert')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})
