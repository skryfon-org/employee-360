import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import { toast } from 'sonner'
import { openRowMenu, pickRowAction } from '../../../test/rowActions'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'
import { useEmployeeUiStore } from '../../../stores/employeeUiStore'

let mock: MockAdapter

const summary = (id: string, first: string, extra: Record<string, unknown> = {}) => ({
  id,
  email: `${first.toLowerCase()}@x.com`,
  first_name: first,
  last_name: 'Doe',
  is_active: true,
  roles: ['employee'],
  department_name: 'Engineering',
  position_name: 'Developer',
  employment_status: 'active',
  employment_type: 'full_time',
  profile_completed: true,
  created_at: '2026-01-01T00:00:00Z',
  ...extra,
})

let contacts: Record<string, unknown>[] = []
let detailExtra: Record<string, unknown> = {}

const detail = () => ({
  ...summary('e1', 'Ada'),
  employee_code: 'E-100',
  department_id: 'd1',
  position_id: 'p1',
  profile: { phone: '5551234', city: 'Pune', employment_status: 'active' },
  emergency_contacts: contacts,
  ...detailExtra,
})

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

/** Picks an option in a (lazily populated) address combobox: wait until enabled, type to filter, click the option. */
async function pickOption(label: string, option: string) {
  const input = screen.getByLabelText(label)
  await waitFor(() => expect(input).toBeEnabled(), { timeout: 15000 })
  await userEvent.click(input)
  await userEvent.type(input, option)
  await userEvent.click(await screen.findByRole('option', { name: option }, { timeout: 15000 }))
}

function signIn(roles = ['admin'], id = 'u1') {
  setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' })
  useAuthStore.setState({ accessToken: 'a', user: { id, email: 'admin@x.com', roles } })
}

beforeEach(() => {
  contacts = []
  detailExtra = {}
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/employees').reply(() => [
    200,
    { success: true, data: [summary('e1', 'Ada'), summary('e2', 'Bob', { is_active: false, profile_completed: false })],
      meta: { page: 1, page_size: 20, total_items: 2, total_pages: 1 } },
  ])
  mock.onGet('/api/v1/departments').reply(200, { success: true, data: [{ id: 'd1', name: 'Engineering', is_active: true }] })
  mock.onGet('/api/v1/positions').reply(200, { success: true, data: [{ id: 'p1', name: 'Developer', is_active: true }] })
  mock.onGet('/api/v1/employees/e1').reply(() => [200, { success: true, data: detail() }])
  mock.onGet('/api/v1/employees/e1/assignments').reply(200, {
    success: true,
    data: [
      { id: 'a1', department_id: 'd1', position_id: 'p1', effective_from: '2024-01-10', is_current: true, change_reason: 'Hired' },
    ],
  })
  clearSession()
  useAuthStore.getState().clear()
  useEmployeeUiStore.setState({ listSearch: '' })
  localStorage.clear()
  toast.dismiss()
})
afterEach(() => mock.restore())

describe('employees list', () => {
  it('shows rows with status badges, filters and a sidebar entry', async () => {
    signIn()
    renderAt('/employees')
    expect(screen.getByRole('status')).toHaveTextContent('Loading employees…')
    expect(await screen.findByText('Ada Doe')).toBeInTheDocument()
    const table = screen.getByRole('table')
    for (const h of ['Employee', 'Department', 'Position', 'Employment', 'Status', 'Actions']) {
      expect(within(table).getByRole('columnheader', { name: h })).toBeInTheDocument()
    }
    expect(within(table).getByText('Inactive')).toBeInTheDocument()
    expect(within(table).getByText('Profile incomplete')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Employees' })).toHaveAttribute('href', '/employees')
    for (const l of ['Search', 'Status', 'Department', 'Position', 'Employment status']) {
      expect(screen.getByLabelText(l)).toBeInTheDocument()
    }
    await openRowMenu('Actions for employee Ada Doe')
    expect(screen.getByRole('menuitem', { name: 'View' })).toBeInTheDocument()
  })

  it('sends filters from the URL to the API', async () => {
    signIn()
    renderAt('/employees?status=inactive&department_id=d1&employment_status=probation&page=2')
    await screen.findByText('Ada Doe')
    const call = mock.history.get.find((r) => r.url === '/api/v1/employees')
    expect(call?.params).toMatchObject({ status: 'inactive', department_id: 'd1', employment_status: 'probation', page: 2 })
  })

  it('applies a filter change and the debounced search', async () => {
    signIn()
    renderAt('/employees')
    await screen.findByText('Ada Doe')
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'active')
    await waitFor(() =>
      expect(mock.history.get.some((r) => r.url === '/api/v1/employees' && r.params?.status === 'active')).toBe(true),
    )
    await userEvent.type(screen.getByLabelText('Search'), 'ada')
    await waitFor(() =>
      expect(mock.history.get.some((r) => r.url === '/api/v1/employees' && r.params?.search === 'ada')).toBe(true),
    )
  })

  it('confirms before deactivating and calls the API', async () => {
    signIn()
    mock.onPost('/api/v1/employees/e1/deactivate').reply(200, { success: true, data: detail() })
    renderAt('/employees')
    await pickRowAction('Actions for employee Ada Doe', 'Deactivate')
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByText('Deactivate Ada Doe?')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
    await userEvent.click(within(dialog).getByRole('button', { name: 'Deactivate' }))
    await waitFor(() => expect(mock.history.post.some((r) => r.url === '/api/v1/employees/e1/deactivate')).toBe(true))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('activates an inactive employee', async () => {
    signIn()
    mock.onPost('/api/v1/employees/e2/activate').reply(200, { success: true, data: detail() })
    renderAt('/employees')
    await pickRowAction('Actions for employee Bob Doe', 'Activate')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Activate' }))
    await waitFor(() => expect(mock.history.post.some((r) => r.url === '/api/v1/employees/e2/activate')).toBe(true))
  })

  it('shows an error with retry when loading fails', async () => {
    signIn()
    mock.onGet('/api/v1/employees').reply(500, { success: false, error: { message: 'boom' } })
    renderAt('/employees')
    expect(await screen.findByRole('alert')).toHaveTextContent(/boom|500/)
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})

describe('employee detail', () => {
  it('redirects to the overview and shows sub-navigation', async () => {
    signIn()
    renderAt('/employees/e1')
    expect(await screen.findByRole('heading', { name: 'Ada Doe' })).toBeInTheDocument()
    const nav = screen.getByRole('navigation', { name: 'Employee sections' })
    const hrefs = within(nav).getAllByRole('link').map((l) => l.getAttribute('href'))
    expect(hrefs).toEqual(['overview', 'employment', 'personal', 'address', 'emergency-contacts', 'assignments', 'allocations', 'status-logs', 'leave'].map((p) => `/employees/e1/${p}`))
    await waitFor(() => expect(within(nav).getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page'))
    expect(within(nav).queryByRole('link', { name: 'Role' })).not.toBeInTheDocument()
    expect(screen.getByText('E-100')).toBeInTheDocument()
    await userEvent.click(within(nav).getByRole('link', { name: 'Employment' }))
    expect(await screen.findByText('Employee code')).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Employment details' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit employment' }))
    expect(await screen.findByRole('form', { name: 'Employment details' })).toBeInTheDocument()
    expect(mock.history.get.filter((r) => r.url === '/api/v1/employees/e1')).toHaveLength(1)
  })

  it('back link preserves the list filters', async () => {
    signIn()
    useEmployeeUiStore.setState({ listSearch: 'status=active&page=2' })
    renderAt('/employees/e1/employment')
    await screen.findByRole('heading', { name: 'Ada Doe' })
    expect(screen.getByRole('link', { name: 'Back to employees' })).toHaveAttribute('href', '/employees?status=active&page=2')
  })

  it('shows assignment history on its own page', async () => {
    signIn()
    renderAt('/employees/e1/assignments')
    expect(await screen.findByText('Hired')).toBeInTheDocument()
  })

  it('saves only changed personal fields and address fields separately', async () => {
    signIn()
    mock.onPatch('/api/v1/employees/e1').reply(200, { success: true, data: detail() })
    renderAt('/employees/e1/personal')
    expect(await screen.findByText('5551234')).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Personal details' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit personal details' }))
    expect(await screen.findByRole('form', { name: 'Personal details' })).toBeInTheDocument()
    expect(screen.getByLabelText('Phone')).toHaveValue('5551234')
    await userEvent.clear(screen.getByLabelText('Phone'))
    await userEvent.type(screen.getByLabelText('Phone'), '5550000')
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toEqual({ phone: '5550000' })
    await waitFor(() => expect(screen.queryByRole('form', { name: 'Personal details' })).not.toBeInTheDocument())

    await userEvent.click(screen.getByRole('link', { name: 'Address' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Edit address' }))
    expect(await screen.findByRole('form', { name: 'Address' })).toBeInTheDocument()
    await pickOption('Country', 'India')
    await pickOption('State', 'Maharashtra')
    await pickOption('City', 'Mumbai')
    await userEvent.click(screen.getByRole('button', { name: 'Save address' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(2))
    expect(JSON.parse(mock.history.patch[1].data)).toEqual({ country: 'India', state: 'Maharashtra', city: 'Mumbai' })
  }, 30000)

  it('cascades country -> state -> city, resets children and prefills stored values in edit mode', async () => {
    signIn()
    detailExtra = { profile: { country: 'India', state: 'Kerala', city: 'Kozhikode', pincode: '673001' } }
    renderAt('/employees/e1/address')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit address' }))
    await waitFor(() => expect(screen.getByLabelText('Country')).toHaveValue('India'))
    await waitFor(() => expect(screen.getByLabelText('State')).toHaveValue('Kerala'))
    await waitFor(() => expect(screen.getByLabelText('City')).toHaveValue('Kozhikode'), { timeout: 15000 })
    expect(screen.getByLabelText('Pincode')).toHaveValue('673001')

    await pickOption('State', 'Goa')
    expect(screen.getByLabelText('City')).toHaveValue('')
    await pickOption('Country', 'Monaco')
    expect(screen.getByLabelText('State')).toHaveValue('')
  }, 30000)

  it('shows a specific PIN code error for India and revalidates when the country changes', async () => {
    signIn()
    detailExtra = { profile: { country: 'India', pincode: '68200' } }
    renderAt('/employees/e1/address')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit address' }))
    await waitFor(() => expect(screen.getByLabelText('Country')).toHaveValue('India'))
    await userEvent.clear(screen.getByLabelText('Pincode'))
    await userEvent.type(screen.getByLabelText('Pincode'), '012345')
    await userEvent.click(screen.getByRole('button', { name: 'Save address' }))
    expect(await screen.findByText('Enter a valid 6-digit PIN code')).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
    await pickOption('Country', 'Monaco')
    await waitFor(() => expect(screen.queryByText('Enter a valid 6-digit PIN code')).not.toBeInTheDocument())
  })

  it('edits names in personal details and patches only the changed one', async () => {
    signIn()
    mock.onPatch('/api/v1/employees/e1').reply(200, { success: true, data: detail() })
    renderAt('/employees/e1/personal')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit personal details' }))
    expect(screen.getByLabelText('Last name *')).toHaveValue('Doe')
    await userEvent.clear(screen.getByLabelText('First name *'))
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    expect(await screen.findByText('First name is required')).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
    await userEvent.type(screen.getByLabelText('First name *'), '  Grace ')
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toEqual({ first_name: 'Grace' })
  })

  it('cancel discards edits and returns to view mode', async () => {
    signIn()
    renderAt('/employees/e1/personal')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit personal details' }))
    await userEvent.type(screen.getByLabelText('Nationality'), 'Indian')
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('form', { name: 'Personal details' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit personal details' }))
    expect(screen.getByLabelText('Nationality')).toHaveValue('')
    expect(mock.history.patch).toHaveLength(0)
  })

  it('validates personal fields before submitting', async () => {
    signIn()
    renderAt('/employees/e1/personal')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit personal details' }))
    await userEvent.clear(screen.getByLabelText('Phone'))
    await userEvent.type(screen.getByLabelText('Phone'), 'abc')
    await userEvent.type(screen.getByLabelText('Personal email'), 'nope')
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    expect(await screen.findByText('Phone is not a valid phone number')).toBeInTheDocument()
    expect(screen.getByText(/Enter a valid email address/)).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
  })

  it('shows effective date and reason when the department changes', async () => {
    signIn()
    mock.onGet('/api/v1/departments').reply(200, {
      success: true,
      data: [{ id: 'd1', name: 'Engineering', is_active: true }, { id: 'd2', name: 'Finance', is_active: true }],
    })
    mock.onPatch('/api/v1/employees/e1').reply(200, { success: true, data: detail() })
    renderAt('/employees/e1/employment')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit employment' }))
    expect(screen.queryByLabelText('Change reason')).not.toBeInTheDocument()
    const dept = await screen.findByLabelText('Department')
    await waitFor(() => expect(within(dept).getByRole('option', { name: 'Finance' })).toBeInTheDocument())
    await userEvent.selectOptions(dept, 'd2')
    await userEvent.type(screen.getByLabelText('Change reason'), 'Transfer')
    await userEvent.click(screen.getByRole('button', { name: 'Save employment details' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toEqual({ department_id: 'd2', change_reason: 'Transfer' })
  })

  it('adds an emergency contact and validates it', async () => {
    signIn()
    mock.onPost('/api/v1/employees/e1/emergency-contacts').reply(201, { success: true, data: { id: 'c1' } })
    renderAt('/employees/e1/emergency-contacts')
    await screen.findByRole('heading', { name: 'Ada Doe' })
    expect(screen.queryByRole('button', { name: 'Add emergency contact' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit emergency contacts' }))
    await userEvent.click(screen.getByRole('button', { name: 'Add emergency contact' }))
    const form = screen.getByRole('form', { name: 'Add emergency contact' })
    await userEvent.click(within(form).getByRole('button', { name: 'Add contact' }))
    expect(await within(form).findByText('Name is required')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
    await userEvent.type(within(form).getByLabelText('Name *'), 'Bo')
    await userEvent.type(within(form).getByLabelText('Relationship *'), 'Sibling')
    await userEvent.type(within(form).getByLabelText('Phone *'), '5559876')
    await userEvent.click(within(form).getByLabelText('Primary contact'))
    await userEvent.click(within(form).getByRole('button', { name: 'Add contact' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ name: 'Bo', relationship: 'Sibling', phone: '5559876', is_primary: true })
  })

  it('removes a contact after confirmation and disables adding at the limit', async () => {
    signIn()
    contacts = Array.from({ length: 10 }, (_, i) => ({ id: `c${i}`, name: `Contact ${i}`, relationship: 'Friend', phone: '5551111', is_primary: i === 0 }))
    mock.onDelete('/api/v1/employees/e1/emergency-contacts/c3').reply(200, { success: true })
    renderAt('/employees/e1/emergency-contacts')
    await screen.findByText('Contact 3')
    expect(screen.queryByRole('button', { name: 'Actions for contact Contact 3' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit emergency contacts' }))
    expect(screen.getByRole('button', { name: 'Add emergency contact' })).toBeDisabled()
    expect(screen.getByText('Primary')).toBeInTheDocument()
    await pickRowAction('Actions for contact Contact 3', 'Remove')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
  })

  it('only offers the role control to super admins, never for themselves', async () => {
    signIn(['admin'])
    renderAt('/employees/e1/role')
    await screen.findByRole('heading', { name: 'Ada Doe' })
    await waitFor(() => expect(screen.queryByRole('link', { name: 'Role' })).not.toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Change role' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Role')).not.toBeInTheDocument()
  })

  it('lets a super admin grant admin after confirmation', async () => {
    signIn(['super_admin'])
    mock.onPut('/api/v1/employees/e1/role').reply(200, { success: true, data: detail() })
    renderAt('/employees/e1')
    await screen.findByRole('heading', { name: 'Ada Doe' })
    await userEvent.click(screen.getByRole('link', { name: 'Role' }))
    expect(await screen.findByText('Current role')).toBeInTheDocument()
    expect(screen.queryByLabelText('Role')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit role' }))
    const change = screen.getByRole('button', { name: 'Change role' })
    expect(change).toBeDisabled()
    await userEvent.selectOptions(screen.getByLabelText('Role'), 'admin')
    await userEvent.click(change)
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Grant admin' }))
    await waitFor(() => expect(mock.history.put).toHaveLength(1))
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ role: 'admin' })
  })

  it('hides the role page for oneself', async () => {
    signIn(['super_admin'], 'e1')
    renderAt('/employees/e1/role')
    await screen.findByRole('heading', { name: 'Ada Doe' })
    expect(screen.queryByRole('link', { name: 'Role' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Change role' })).not.toBeInTheDocument()
  })

  it('makes a super admin account read-only for plain admins', async () => {
    signIn(['admin'])
    detailExtra = { roles: ['super_admin'] }
    renderAt('/employees/e1/address')
    expect(await screen.findByText(/Only a super admin can edit/)).toBeInTheDocument()
    expect(screen.getByText('Pune')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Edit/ })).not.toBeInTheDocument()
  })
})

describe('employee code override', () => {
  it('is read-only by default and validates an override against the org prefix', async () => {
    mock.onGet('/api/v1/organization/settings').reply(200, {
      success: true, data: { employee_code_prefix: 'EMP', employee_code_padding: 4, employee_code_start_number: 1, employee_code_next_number: 1 },
    })
    signIn()
    renderAt('/employees/e1/employment')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit employment' }))
    const input = await screen.findByLabelText('Employee code')
    expect(input).toHaveAttribute('readonly')
    await userEvent.click(screen.getByRole('button', { name: 'Override code' }))
    expect(input).not.toHaveAttribute('readonly')
    await userEvent.clear(input)
    await userEvent.type(input, 'XYZ12')
    await userEvent.click(screen.getByRole('button', { name: /Save employment details/ }))
    expect(await screen.findByText(/EMP followed by digits/)).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
  })

  it('sends a valid override with the PATCH', async () => {
    mock.onGet('/api/v1/organization/settings').reply(200, {
      success: true, data: { employee_code_prefix: 'EMP', employee_code_padding: 4, employee_code_start_number: 1, employee_code_next_number: 1 },
    })
    mock.onPatch('/api/v1/employees/e1').reply(200, { success: true, data: detail() })
    signIn()
    renderAt('/employees/e1/employment')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit employment' }))
    const input = await screen.findByLabelText('Employee code')
    await userEvent.click(screen.getByRole('button', { name: 'Override code' }))
    await userEvent.clear(input)
    await userEvent.type(input, 'EMP0042')
    await userEvent.click(screen.getByRole('button', { name: /Save employment details/ }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toMatchObject({ employee_code: 'EMP0042' })
  })
})
