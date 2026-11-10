import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'
import { toast } from 'sonner'
import { openRowMenu, pickRowAction } from '../../../test/rowActions'
import { visibleNavItems } from '../../../components/layout/navItems'

let mock: MockAdapter

const T1 = '11111111-1111-4111-8111-111111111111'
const dom = (id: string, domain: string) => ({ id, tenant_id: T1, domain })
let detail: Record<string, unknown>

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

function signIn(roles: string[]) {
  setSession({ accessToken: 'a', refreshToken: 'r', tenantId: T1 })
  useAuthStore.setState({ accessToken: 'a', user: { id: 'u1', email: 'root@x.com', roles } })
}

const fail = (status: number, code: string, message: string): [number, unknown] => [status, { success: false, error: { code, message } }]

beforeEach(() => {
  detail = {
    id: T1,
    name: 'Acme',
    is_active: true,
    created_at: '2026-01-01T00:00:00Z',
    domains: [dom('d1', 'acme.com'), dom('d2', 'acme.io')],
  }
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/tenant').reply(() => [200, { success: true, data: detail }])
  mock.onGet('/api/v1/organization/settings').reply(200, { success: true, data: { employee_code_prefix: 'EMP', employee_code_padding: 4, employee_code_start_number: 1, employee_code_next_number: 1, employee_codes_issued: false, employee_code_preview: 'EMP0001' } })
  mock.onGet('/api/v1/dashboard/super-admin').reply(200, { success: true, data: {} })
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
  toast.dismiss()
})
afterEach(() => mock.restore())

describe('organization access', () => {
  it('shows the Organization nav item to admin and super_admin, not employees', async () => {
    signIn(['super_admin'])
    const { unmount } = renderAt('/settings/organization')
    expect(await within(await screen.findByRole('navigation', { name: 'Main' })).findByRole('link', { name: 'Organization' })).toHaveAttribute('href', '/settings/organization')
    unmount()
    signIn(['admin'])
    const second = renderAt('/invitations')
    expect(await within(await screen.findByRole('navigation', { name: 'Main' })).findByRole('link', { name: 'Organization' })).toBeInTheDocument()
    second.unmount()
    const labels = (roles: string[]) => visibleNavItems(roles).map((i) => i.label)
    expect(labels(['admin'])).toContain('Organization')
    expect(labels(['admin'])).not.toContain('Employee codes')
    expect(labels(['employee'])).not.toContain('Organization')
  })

  it('shows admins only the Employee codes, Working days and Welcome email tabs and defaults to it, without fetching the tenant', async () => {
    signIn(['admin'])
    renderAt('/settings/organization')
    expect(await screen.findByLabelText(/Code prefix/)).toBeInTheDocument()
    expect(screen.getAllByRole('tab')).toHaveLength(3)
    expect(screen.getByRole('tab', { name: 'Employee codes' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByRole('tab', { name: 'General' })).toBeNull()
    expect(screen.queryByRole('tab', { name: 'Domains' })).toBeNull()
    expect(mock.history.get.some((r) => r.url === '/api/v1/tenant')).toBe(false)
  })

  it.each(['general', 'domains'])('redirects admins from /%s to employee-codes', async (tab) => {
    signIn(['admin'])
    renderAt(`/settings/organization/${tab}`)
    expect(await screen.findByLabelText(/Code prefix/)).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Rename organization' })).toBeNull()
    expect(mock.history.get.some((r) => r.url === '/api/v1/tenant')).toBe(false)
  })

  it('shows super_admin all four tabs, with the tenant fetched only for tenant tabs', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization/employee-codes')
    expect(await screen.findByLabelText(/Code prefix/)).toBeInTheDocument()
    expect(screen.getAllByRole('tab').map((t) => t.textContent)).toEqual(['General', 'Domains', 'Employee codes', 'Working days', 'Welcome email'])
    expect(mock.history.get.some((r) => r.url === '/api/v1/tenant')).toBe(false)
    await userEvent.click(screen.getByRole('tab', { name: 'Domains' }))
    expect(await screen.findByText('acme.com')).toBeInTheDocument()
  })

  it('redirects employees away without calling the API', async () => {
    signIn(['employee'])
    renderAt('/settings/organization')
    await waitFor(() => expect(screen.queryByRole('tab')).toBeNull())
    expect(mock.history.get.some((r) => r.url === '/api/v1/tenant')).toBe(false)
  })
})

describe('organization page', () => {
  it('renders the general section at /general', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization/general')
    expect(await screen.findByLabelText('Name')).toHaveValue('Acme')
    expect(screen.queryByText('acme.com')).toBeNull()
    expect(screen.getByRole('tab', { name: 'General' })).toHaveAttribute('aria-selected', 'true')
  })

  it('shows the general skeleton while the tenant loads, then the form', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization/general')
    expect(screen.getByRole('status')).toHaveTextContent('Loading organization…')
    expect(screen.queryByLabelText('Name')).toBeNull()
    expect(await screen.findByLabelText('Name')).toHaveValue('Acme')
    expect(screen.queryByText('Loading organization…')).toBeNull()
  })

  it('shows the domains skeleton while the tenant loads, then the domains', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization/domains')
    expect(screen.getByRole('status')).toHaveTextContent('Loading organization…')
    expect(await screen.findByText('acme.com')).toBeInTheDocument()
    expect(screen.queryByText('Loading organization…')).toBeNull()
  })

  it('renders the domains section at /domains', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization/domains')
    expect(await screen.findByText('acme.com')).toBeInTheDocument()
    expect(screen.getByText('acme.io')).toBeInTheDocument()
    expect(screen.queryByLabelText('Name')).toBeNull()
    expect(screen.getByRole('tab', { name: 'Domains' })).toHaveAttribute('aria-selected', 'true')
  })

  it('redirects the root path to general', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization')
    expect(await screen.findByLabelText('Name')).toHaveValue('Acme')
    expect(screen.getByRole('tab', { name: 'General' })).toHaveAttribute('aria-selected', 'true')
  })

  it('switches tabs, updating the URL, breadcrumbs and nav, fetching the tenant once', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization')
    await screen.findByLabelText('Name')
    await userEvent.click(screen.getByRole('tab', { name: 'Domains' }))
    expect(await screen.findByText('acme.com')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Domains' })).toHaveAttribute('aria-selected', 'true')
    const crumbs = screen.getByRole('navigation', { name: /breadcrumb/i })
    expect(within(crumbs).getByText('Domains')).toHaveAttribute('aria-current', 'page')
    expect(within(crumbs).queryByText('Settings')).not.toBeInTheDocument()
    expect(within(crumbs).getByRole('link', { name: 'Organization' })).toHaveAttribute('href', '/settings/organization')
    expect(within(screen.getByRole('navigation', { name: 'Main' })).getByRole('link', { name: 'Organization' })).toHaveAttribute('aria-current', 'page')
    await userEvent.click(screen.getByRole('tab', { name: 'General' }))
    expect(await screen.findByLabelText('Name')).toBeInTheDocument()
    expect(mock.history.get.filter((r) => r.url === '/api/v1/tenant')).toHaveLength(1)
  })

  it('renames the organization', async () => {
    signIn(['super_admin'])
    mock.onPatch('/api/v1/tenant').reply(() => {
      detail = { ...detail, name: 'Acme Corp' }
      return [200, { success: true, data: detail }]
    })
    renderAt('/settings/organization')
    const input = await screen.findByLabelText('Name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Acme Corp')
    await userEvent.click(screen.getByRole('button', { name: 'Save name' }))
    await waitFor(() => expect(JSON.parse(mock.history.patch[0].data)).toEqual({ name: 'Acme Corp' }))
    await waitFor(() => expect(mock.history.get.filter((r) => r.url === '/api/v1/tenant').length).toBeGreaterThan(1))
    expect(screen.getByLabelText('Name')).toHaveValue('Acme Corp')
  })

  it('shows INVALID_NAME inline on the name field', async () => {
    signIn(['super_admin'])
    mock.onPatch('/api/v1/tenant').reply(() => fail(400, 'INVALID_NAME', 'Name is not allowed'))
    renderAt('/settings/organization')
    const input = await screen.findByLabelText('Name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Bad')
    await userEvent.click(screen.getByRole('button', { name: 'Save name' }))
    expect(await screen.findByText('Name is not allowed')).toBeInTheDocument()
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })

  it('adds a domain', async () => {
    signIn(['super_admin'])
    mock.onPost('/api/v1/tenant/domains').reply(() => {
      detail = { ...detail, domains: [...(detail.domains as unknown[]), dom('d3', 'acme.dev')] }
      return [201, { success: true, data: dom('d3', 'acme.dev') }]
    })
    renderAt('/settings/organization/domains')
    await userEvent.type(await screen.findByLabelText('New domain'), 'acme.dev')
    await userEvent.click(screen.getByRole('button', { name: 'Add domain' }))
    await waitFor(() => expect(JSON.parse(mock.history.post[0].data)).toEqual({ domain: 'acme.dev' }))
    expect(await screen.findByText('acme.dev')).toBeInTheDocument()
  })

  it('shows DOMAIN_ALREADY_EXISTS inline when adding a domain', async () => {
    signIn(['super_admin'])
    mock.onPost('/api/v1/tenant/domains').reply(() => fail(409, 'DOMAIN_ALREADY_EXISTS', 'Domain is already registered'))
    renderAt('/settings/organization/domains')
    await userEvent.type(await screen.findByLabelText('New domain'), 'taken.com')
    await userEvent.click(screen.getByRole('button', { name: 'Add domain' }))
    expect(await screen.findByText('Domain is already registered')).toBeInTheDocument()
  })

  it('edits a domain inline', async () => {
    signIn(['super_admin'])
    mock.onPatch('/api/v1/tenant/domains/d2').reply(() => {
      detail = { ...detail, domains: [dom('d1', 'acme.com'), dom('d2', 'acme.org')] }
      return [200, { success: true, data: dom('d2', 'acme.org') }]
    })
    renderAt('/settings/organization/domains')
    await pickRowAction('Actions for domain acme.io', 'Edit')
    const form = screen.getByRole('form', { name: 'Edit domain acme.io' })
    const input = within(form).getByLabelText('Domain')
    await userEvent.clear(input)
    await userEvent.type(input, 'acme.org')
    await userEvent.click(within(form).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(JSON.parse(mock.history.patch[0].data)).toEqual({ domain: 'acme.org' }))
    expect(await screen.findByText('acme.org')).toBeInTheDocument()
    expect(screen.queryByText('acme.io')).toBeNull()
  })

  it('shows INVALID_DOMAIN inline when editing a domain', async () => {
    signIn(['super_admin'])
    mock.onPatch('/api/v1/tenant/domains/d2').reply(() => fail(400, 'INVALID_DOMAIN', 'Domain is not valid'))
    renderAt('/settings/organization/domains')
    await pickRowAction('Actions for domain acme.io', 'Edit')
    const form = screen.getByRole('form', { name: 'Edit domain acme.io' })
    const input = within(form).getByLabelText('Domain')
    await userEvent.clear(input)
    await userEvent.type(input, 'bad.example')
    await userEvent.click(within(form).getByRole('button', { name: 'Save' }))
    expect(await within(form).findByText('Domain is not valid')).toBeInTheDocument()
  })

  it('removes a domain after confirmation', async () => {
    signIn(['super_admin'])
    mock.onDelete('/api/v1/tenant/domains/d2').reply(() => {
      detail = { ...detail, domains: [dom('d1', 'acme.com')] }
      return [200, { success: true }]
    })
    renderAt('/settings/organization/domains')
    await pickRowAction('Actions for domain acme.io', 'Remove')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
    await waitFor(() => expect(screen.queryByText('acme.io')).toBeNull())
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('disables remove when only one domain remains', async () => {
    signIn(['super_admin'])
    detail = { ...detail, domains: [dom('d1', 'acme.com')] }
    renderAt('/settings/organization/domains')
    await openRowMenu('Actions for domain acme.com')
    expect(screen.getByRole('menuitem', { name: 'Remove' })).toBeDisabled()
  })

  it('shows a persistent LAST_DOMAIN alert when the server returns 409', async () => {
    signIn(['super_admin'])
    mock.onDelete('/api/v1/tenant/domains/d2').reply(() => fail(409, 'LAST_DOMAIN', 'An organization must keep at least one domain.'))
    renderAt('/settings/organization/domains')
    await pickRowAction('Actions for domain acme.io', 'Remove')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Remove' }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('An organization must keep at least one domain.')
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(screen.getByText('acme.io')).toBeInTheDocument()
  })

  const IN_USE = 'users in the organization still sign in with an @acme.io email; move or deactivate them before removing or changing this domain'

  it('shows a persistent DOMAIN_IN_USE alert and closes the dialog on remove', async () => {
    signIn(['super_admin'])
    mock.onDelete('/api/v1/tenant/domains/d2').reply(() => fail(409, 'DOMAIN_IN_USE', IN_USE))
    renderAt('/settings/organization/domains')
    await pickRowAction('Actions for domain acme.io', 'Remove')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Remove' }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(IN_USE)
    expect(alert).toHaveTextContent('Users page')
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(screen.getByText('acme.io')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('shows DOMAIN_IN_USE in the card and keeps the edit form open', async () => {
    signIn(['super_admin'])
    mock.onPatch('/api/v1/tenant/domains/d2').reply(() => fail(409, 'DOMAIN_IN_USE', IN_USE))
    renderAt('/settings/organization/domains')
    await pickRowAction('Actions for domain acme.io', 'Edit')
    const form = screen.getByRole('form', { name: 'Edit domain acme.io' })
    const input = within(form).getByLabelText('Domain')
    await userEvent.clear(input)
    await userEvent.type(input, 'acme.org')
    await userEvent.click(within(form).getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(IN_USE)
    expect(screen.getByRole('form', { name: 'Edit domain acme.io' })).toBeInTheDocument()
    expect(within(form).getByLabelText('Domain')).toHaveValue('acme.org')
  })

  it('shows the full-domain helper text under the add and edit inputs', async () => {
    signIn(['super_admin'])
    renderAt('/settings/organization/domains')
    expect(await screen.findByText('Enter a full domain such as example.com')).toBeInTheDocument()
    await pickRowAction('Actions for domain acme.io', 'Edit')
    expect(screen.getAllByText('Enter a full domain such as example.com')).toHaveLength(2)
  })
})
