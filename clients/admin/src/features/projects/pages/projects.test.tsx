import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import { toast } from 'sonner'
import { pickRowAction } from '../../../test/rowActions'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'
import { allocationState } from '../allocationState'

let mock: MockAdapter
let allocations: Record<string, unknown>[]
let userAllocs: Record<string, unknown>

const project = {
  id: 'p1', name: 'Apollo', code: 'APL', description: 'Moon', client_name: 'NASA', status: 'active',
  start_date: '2026-01-01', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}
const emp = (id: string, first: string) => ({ id, employee_code: `E-${id.toUpperCase()}`, department_name: 'Engineering', email: `${first.toLowerCase()}@x.com`, first_name: first, last_name: 'Doe', is_active: true, roles: ['employee'] })

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

function signIn(roles = ['admin']) {
  setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' })
  useAuthStore.setState({ accessToken: 'a', user: { id: 'u1', email: 'admin@x.com', roles } })
}

beforeEach(() => {
  allocations = [
    { id: 'al1', project_id: 'p1', user_id: 'e1', user_name: 'Ada Doe', allocation_percent: 60, role_on_project: 'Dev', start_date: '2026-02-01' },
  ]
  userAllocs = { allocations: [], total_percent: 0, over_allocated: false }
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/projects').reply(200, {
    success: true, data: [project], meta: { page: 1, page_size: 20, total_items: 1, total_pages: 1 },
  })
  mock.onGet('/api/v1/projects/code-suggestion').reply((config) => [
    200,
    { success: true, data: { code: String(config.params?.name).toUpperCase().replace(/\s+/g, '-') } },
  ])
  mock.onGet('/api/v1/projects/p1').reply(200, { success: true, data: project })
  mock.onGet('/api/v1/projects/p1/allocations').reply((config) => {
    const state = config.params?.state as string | undefined
    const data = !state || state === 'all' ? allocations : allocations.filter((a) => allocationState(a) === state)
    return [200, { success: true, data }]
  })
  mock.onGet('/api/v1/employees').reply((config) => {
    const q = String(config.params?.search ?? '').toLowerCase()
    const pctOf: Record<string, number> = { e1: 60, e2: 100 }
    const all = [emp('e1', 'Ada'), emp('e2', 'Bob')].map((e) => (config.params?.include === 'allocation' ? { ...e, allocated_percent: pctOf[e.id] } : e))
    const hit = all.filter((e) => !q || [`${e.first_name} ${e.last_name}`, e.email, e.employee_code].some((v) => v.toLowerCase().includes(q)))
    return [200, { success: true, data: hit, meta: { total_items: hit.length, total_pages: 1 } }]
  })
  mock.onGet('/api/v1/employees/e1').reply(200, { success: true, data: { ...emp('e1', 'Ada'), profile: {} } })
  mock.onGet('/api/v1/employees/e1/allocations').reply(() => [200, { success: true, data: userAllocs }])
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
  toast.dismiss()
})
afterEach(() => mock.restore())

describe('projects list', () => {
  it('shows rows with status badge, filters, and an admin-only nav entry', async () => {
    signIn()
    renderAt('/projects')
    expect(await screen.findByRole('link', { name: 'Apollo' })).toHaveAttribute('href', '/projects/p1')
    const table = screen.getByRole('table')
    expect(within(table).getByText('Active')).toBeInTheDocument()
    expect(within(table).getByText('APL')).toBeInTheDocument()
    expect(screen.getByLabelText('Search')).toBeInTheDocument()
    expect(screen.getByLabelText('Status')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Projects' })).toHaveAttribute('href', '/projects')
  })

  it('sends URL filters to the API and applies a status change', async () => {
    signIn()
    renderAt('/projects?status=on_hold&page=2')
    await screen.findByRole('link', { name: 'Apollo' })
    expect(mock.history.get.find((r) => r.url === '/api/v1/projects')?.params).toMatchObject({ status: 'on_hold', page: 2 })
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'completed')
    await waitFor(() =>
      expect(mock.history.get.some((r) => r.url === '/api/v1/projects' && r.params?.status === 'completed')).toBe(true),
    )
  })

  it('validates the form (end >= start; code is optional on create)', async () => {
    signIn()
    renderAt('/projects')
    await screen.findByRole('link', { name: 'Apollo' })
    await userEvent.click(screen.getByRole('link', { name: 'New project' }))
    const form = screen.getByRole('form', { name: 'Create project' })
    await userEvent.type(within(form).getByLabelText(/^Name/), 'Zeus')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-05-10')
    await userEvent.type(within(form).getByLabelText(/^End date/), '2026-05-01')
    await userEvent.click(within(form).getByRole('button', { name: 'Create' }))
    expect(await within(form).findByText('End date must be on or after the start date')).toBeInTheDocument()
    expect(within(form).queryByText('Code is required')).not.toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })

  it('creates a project, surfaces a unique-code conflict, and refetches on success', async () => {
    signIn()
    let n = 0
    mock.onPost('/api/v1/projects').reply(() =>
      n++ === 0
        ? [409, { success: false, error: { code: 'CONFLICT', message: 'project code already exists' } }]
        : [201, { success: true, data: { ...project, id: 'p2', name: 'Zeus', code: 'ZEU' } }],
    )
    renderAt('/projects')
    await screen.findByRole('link', { name: 'Apollo' })
    await userEvent.click(screen.getByRole('link', { name: 'New project' }))
    const form = screen.getByRole('form', { name: 'Create project' })
    await userEvent.type(within(form).getByLabelText(/^Name/), 'Zeus')
    await userEvent.type(within(form).getByLabelText(/^Code/), 'APL')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-05-10')
    await userEvent.click(within(form).getByRole('button', { name: 'Create' }))
    expect(await within(form).findByText('A project with this code already exists.')).toBeInTheDocument()

    mock.onGet('/api/v1/projects/p2').reply(200, { success: true, data: { ...project, id: 'p2', name: 'Zeus', code: 'ZEU' } })
    mock.onGet('/api/v1/projects/p2/allocations').reply(200, { success: true, data: [] })
    await userEvent.clear(within(form).getByLabelText(/^Code/))
    await userEvent.type(within(form).getByLabelText(/^Code/), 'ZEU')
    await userEvent.click(within(form).getByRole('button', { name: 'Create' }))
    expect(await screen.findByRole('heading', { name: 'Zeus' })).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Create project' })).not.toBeInTheDocument()
    expect(JSON.parse(mock.history.post[1].data)).toMatchObject({ name: 'Zeus', code: 'ZEU', start_date: '2026-05-10' })
  })
})

describe('project code auto-generation', () => {
  it('shows the suggestion, and omits code from the request when left on auto', async () => {
    signIn()
    mock.onPost('/api/v1/projects').reply(201, { success: true, data: { ...project, id: 'p3', name: 'Big Move', code: 'BIG-MOVE' } })
    mock.onGet('/api/v1/projects/p3').reply(200, { success: true, data: { ...project, id: 'p3', name: 'Big Move', code: 'BIG-MOVE' } })
    mock.onGet('/api/v1/projects/p3/allocations').reply(200, { success: true, data: [] })
    renderAt('/projects/new')
    const form = await screen.findByRole('form', { name: 'Create project' })
    expect(within(form).getByLabelText(/^Code/)).not.toBeRequired()
    await userEvent.type(within(form).getByLabelText(/^Name/), 'Big Move')
    await waitFor(() => expect(within(form).getByLabelText(/^Code/)).toHaveAttribute('placeholder', 'BIG-MOVE'))
    expect(within(screen.getByRole('complementary', { name: 'Summary' })).getByText('BIG-MOVE')).toBeInTheDocument()
    expect(within(form).getByText(/Auto-generated from the name/)).toBeInTheDocument()
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-05-10')
    await userEvent.click(within(form).getByRole('button', { name: 'Create' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).not.toHaveProperty('code')
  })

  it('sends a manually typed code and stops fetching suggestions', async () => {
    signIn()
    mock.onPost('/api/v1/projects').reply(201, { success: true, data: { ...project, id: 'p4' } })
    mock.onGet('/api/v1/projects/p4').reply(200, { success: true, data: { ...project, id: 'p4' } })
    mock.onGet('/api/v1/projects/p4/allocations').reply(200, { success: true, data: [] })
    renderAt('/projects/new')
    const form = await screen.findByRole('form', { name: 'Create project' })
    await userEvent.type(within(form).getByLabelText(/^Code/), 'MY-CODE')
    await userEvent.type(within(form).getByLabelText(/^Name/), 'Zed')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-05-10')
    await userEvent.click(within(form).getByRole('button', { name: 'Create' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ code: 'MY-CODE', name: 'Zed' })
    expect(mock.history.get.some((r) => r.url === '/api/v1/projects/code-suggestion')).toBe(false)
  })

  it('does not fetch suggestions in edit mode', async () => {
    signIn()
    renderAt('/projects/p1/edit')
    const form = await screen.findByRole('form', { name: 'Edit project' })
    expect(within(form).getByLabelText(/^Code/)).toHaveValue('APL')
    expect(mock.history.get.some((r) => r.url === '/api/v1/projects/code-suggestion')).toBe(false)
  })
})

describe('project edit page', () => {
  it('opens from the detail Edit link, saves, and returns to the detail page', async () => {
    signIn()
    mock.onPatch('/api/v1/projects/p1').reply(200, { success: true, data: { ...project, name: 'Apollo 2' } })
    renderAt('/projects/p1')
    expect(await screen.findByRole('link', { name: 'Edit project' })).toHaveAttribute('href', '/projects/p1/edit')
    await userEvent.click(screen.getByRole('link', { name: 'Edit project' }))
    const form = await screen.findByRole('form', { name: 'Edit project' })
    expect(within(form).getByLabelText(/^Code/)).toHaveValue('APL')
    expect(screen.getByRole('link', { name: 'Cancel' })).toHaveAttribute('href', '/projects/p1')
    await userEvent.clear(within(form).getByLabelText(/^Name/))
    await userEvent.type(within(form).getByLabelText(/^Name/), 'Apollo 2')
    await userEvent.click(within(form).getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toMatchObject({ name: 'Apollo 2', code: 'APL' })
    expect(await screen.findByRole('button', { name: 'Add allocation' })).toBeInTheDocument()
  })
})

describe('project detail and allocations', () => {
  it('shows the name returned by the API even when the person is absent from the employee list', async () => {
    signIn()
    allocations = [{ id: 'al9', project_id: 'p1', user_id: 'e9', user_name: 'Zed Doe', allocation_percent: 50, start_date: '2026-02-01' }]
    renderAt('/projects/p1')
    expect(await screen.findByRole('link', { name: 'Zed Doe' })).toBeInTheDocument()
    expect(screen.queryByText('Unknown employee')).not.toBeInTheDocument()
    expect(mock.history.get.some((r) => /^\/api\/v1\/employees\/e9/.test(r.url ?? ''))).toBe(false)
  })

  it('falls back to "Unknown employee" when user_name is empty', async () => {
    signIn()
    allocations = [{ id: 'al8', project_id: 'p1', user_id: 'e8', user_name: '', allocation_percent: 50, start_date: '2026-02-01' }]
    renderAt('/projects/p1')
    expect(await screen.findByText('Unknown employee')).toBeInTheDocument()
  })

  it('lists allocations with employee names and adds one, warning (not blocking) on over-allocation', async () => {
    signIn()
    mock.onPost('/api/v1/projects/p1/allocations').reply(() => {
      allocations = [...allocations, { id: 'al2', project_id: 'p1', user_id: 'e2', user_name: 'Bob Doe', allocation_percent: 80, start_date: '2026-03-01' }]
      return [201, { success: true, data: { id: 'al2', over_allocated: true, user_total_percent: 140 } }]
    })
    renderAt('/projects/p1')
    const row = (await screen.findByRole('link', { name: 'Ada Doe' })).closest('tr') as HTMLElement
    expect(within(row).getByText('60%')).toBeInTheDocument()
    expect(within(row).getByText('Dev')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Add allocation' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    const combo = within(form).getByRole('combobox', { name: /Employee/ })
    await userEvent.type(combo, 'bob')
    await userEvent.click(await within(form).findByRole('option', { name: /Bob Doe/ }))
    expect(combo).toHaveValue('Bob Doe (E-E2)')
    expect(mock.history.get.some((r) => r.url === '/api/v1/employees' && r.params?.status === 'active' && r.params?.page_size === 10)).toBe(true)
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '80')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-03-01')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))

    await waitFor(() => expect(screen.queryByRole('form', { name: 'Add allocation' })).not.toBeInTheDocument())
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ user_id: 'e2', allocation_percent: 80, start_date: '2026-03-01', end_date: null })
    expect(await screen.findByRole('link', { name: 'Bob Doe' })).toBeInTheDocument()
  })

  it('allocates from the employee page by picking an active project, validating its window', async () => {
    signIn()
    userAllocs = { allocations: [], total_percent: 0, over_allocated: false }
    mock.onGet('/api/v1/projects').reply((config) => {
      const q = String(config.params?.search ?? '').toLowerCase()
      const hit = [{ ...project, end_date: '2026-12-31' }].filter((p) => !q || p.name.toLowerCase().includes(q))
      return [200, { success: true, data: hit, meta: { page: 1, page_size: 10, total_items: hit.length, total_pages: 1 } }]
    })
    mock.onPost('/api/v1/projects/p1/allocations').reply(201, { success: true, data: { id: 'al5', over_allocated: false, user_total_percent: 40 } })
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Allocate to project' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    expect(within(form).queryByRole('combobox', { name: /Employee/ })).not.toBeInTheDocument()
    await userEvent.type(within(form).getByRole('combobox', { name: /Project/ }), 'apo')
    await userEvent.click(await within(form).findByRole('option', { name: /Apollo/ }))
    expect(mock.history.get.some((r) => r.url === '/api/v1/projects' && r.params?.status === 'active')).toBe(true)
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '40')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2025-12-01')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    expect(await within(form).findByText('Start date must not be before the project start date')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
    const start = within(form).getByLabelText(/^Start date/)
    await userEvent.clear(start)
    await userEvent.type(start, '2026-03-01')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    await waitFor(() => expect(screen.queryByRole('form', { name: 'Add allocation' })).not.toBeInTheDocument())
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ user_id: 'e1', allocation_percent: 40, start_date: '2026-03-01' })
  })

  it('counts only allocations covering today as active in the summary', async () => {
    signIn()
    const day = (offset: number) => new Date(Date.now() + offset * 86400000).toISOString().slice(0, 10)
    allocations = [
      { id: 'a1', project_id: 'p1', user_id: 'e1', user_name: 'Ada Doe', allocation_percent: 50, start_date: day(-10), end_date: day(-1) },
      { id: 'a2', project_id: 'p1', user_id: 'e2', user_name: 'Bob Doe', allocation_percent: 50, start_date: day(1), end_date: day(5) },
      { id: 'a3', project_id: 'p1', user_id: 'e1', user_name: 'Ada Doe', allocation_percent: 50, start_date: day(-3) },
    ]
    renderAt('/projects/p1')
    const summary = await screen.findByRole('complementary', { name: 'Project summary' })
    await waitFor(() => expect(within(summary).getByText('Active allocations').nextElementSibling).toHaveTextContent('1'))
    expect(within(summary).getByText('Allocated employees').nextElementSibling).toHaveTextContent('1')
  })

  it('does not mark the allocation end date as required', async () => {
    signIn()
    renderAt('/projects/p1')
    await userEvent.click(await screen.findByRole('button', { name: 'Add allocation' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    expect(within(form).getByLabelText(/^End date/)).not.toBeRequired()
  })

  it('rejects an out-of-range percent client-side', async () => {
    signIn()
    renderAt('/projects/p1')
    await userEvent.click(await screen.findByRole('button', { name: 'Add allocation' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '150')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    expect(await within(form).findByText('Enter a whole number from 1 to 100')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })

  it('allows clearing the end date when editing (sends null)', async () => {
    signIn()
    allocations = [{ ...allocations[0], end_date: '2026-06-30' }]
    mock.onPatch('/api/v1/allocations/al1').reply(200, { success: true, data: { id: 'al1', over_allocated: false, user_total_percent: 60 } })
    renderAt('/projects/p1')
    await screen.findByRole('link', { name: 'Ada Doe' })
    await pickRowAction('Actions for allocation Ada Doe', 'Edit')
    const form = screen.getByRole('form', { name: 'Edit allocation' })
    await userEvent.clear(within(form).getByLabelText(/^End date/))
    await userEvent.click(within(form).getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data).end_date).toBeNull()
  })

  it('edits and ends an allocation', async () => {
    signIn()
    mock.onPatch('/api/v1/allocations/al1').reply(200, { success: true, data: { id: 'al1', over_allocated: false, user_total_percent: 50 } })
    renderAt('/projects/p1')
    await screen.findByRole('link', { name: 'Ada Doe' })
    await pickRowAction('Actions for allocation Ada Doe', 'Edit')
    const form = screen.getByRole('form', { name: 'Edit allocation' })
    const pct = within(form).getByLabelText(/^Allocation %/)
    await userEvent.clear(pct)
    await userEvent.type(pct, '50')
    await userEvent.click(within(form).getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toMatchObject({ allocation_percent: 50 })

    await pickRowAction('Actions for allocation Ada Doe', 'End allocation')
    await waitFor(() => expect(mock.history.patch).toHaveLength(2))
    expect(JSON.parse(mock.history.patch[1].data).end_date).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})

describe('allocation states, filter and timeline', () => {
  const day = (offset: number) => new Date(Date.now() + offset * 86400000).toISOString().slice(0, 10)
  const seed = () => {
    allocations = [
      { id: 'a1', project_id: 'p1', user_id: 'e1', user_name: 'Ada Doe', allocation_percent: 50, start_date: day(-10), end_date: day(-1) },
      { id: 'a2', project_id: 'p1', user_id: 'e2', user_name: 'Bob Doe', allocation_percent: 40, start_date: day(1), end_date: day(5) },
      { id: 'a3', project_id: 'p1', user_id: 'e1', user_name: 'Ada Doe', allocation_percent: 30, start_date: day(-3) },
    ]
  }

  it('shows a state badge per row and filters via ?state= with a per-filter empty state', async () => {
    signIn()
    seed()
    renderAt('/projects/p1')
    await screen.findAllByRole('link', { name: 'Ada Doe' })
    const table = screen.getByRole('table')
    expect(within(table).getByText('Past')).toBeInTheDocument()
    expect(within(table).getByText('Upcoming')).toBeInTheDocument()
    expect(within(table).getByText('Active')).toBeInTheDocument()
    expect(within(table).getAllByRole('row')).toHaveLength(4)

    const listCalls = () => mock.history.get.filter((r) => r.url === '/api/v1/projects/p1/allocations')
    expect(listCalls().some((r) => r.params?.state === undefined)).toBe(true)
    await userEvent.selectOptions(screen.getByLabelText('Show'), 'upcoming')
    expect(screen.getByLabelText('Show')).toHaveValue('upcoming')
    await waitFor(() => expect(listCalls().some((r) => r.params?.state === 'upcoming')).toBe(true))
    await waitFor(() => expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2))
    expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2)
    expect(within(screen.getByRole('table')).getByRole('link', { name: 'Bob Doe' })).toBeInTheDocument()
  })

  it('reads the filter from the URL and shows an empty state', async () => {
    signIn()
    allocations = [{ id: 'a1', project_id: 'p1', user_id: 'e1', user_name: 'Ada Doe', allocation_percent: 50, start_date: day(-3) }]
    renderAt('/projects/p1?state=past')
    expect(await screen.findByText('No past allocations.')).toBeInTheDocument()
    expect(screen.getByLabelText('Show')).toHaveValue('past')
    await userEvent.click(screen.getByRole('button', { name: 'Show all allocations' }))
    expect(await screen.findByRole('link', { name: 'Ada Doe' })).toBeInTheDocument()
  })

  it('validates allocation dates against the project window', async () => {
    signIn()
    mock.onGet('/api/v1/projects/p1').reply(200, { success: true, data: { ...project, end_date: '2026-12-31' } })
    renderAt('/projects/p1')
    await userEvent.click(await screen.findByRole('button', { name: 'Add allocation' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '50')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2025-12-01')
    await userEvent.type(within(form).getByLabelText(/^End date/), '2027-01-15')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    expect(await within(form).findByText('Start date must not be before the project start date')).toBeInTheDocument()
    expect(within(form).getByText('End date must not be after the project end date')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })

  it('renders timeline bars with accessible labels and project markers', async () => {
    signIn()
    seed()
    renderAt('/projects/p1?view=timeline')
    const bars = await screen.findAllByTestId('timeline-bar')
    expect(bars).toHaveLength(3)
    expect(screen.getByRole('img', { name: /Bob Doe, 40% allocation, upcoming/ })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Ada Doe, 30% allocation, active, .* to open-ended/ })).toBeInTheDocument()
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Previous|Next|Today/ })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Table' }))
    expect(await screen.findByRole('table')).toBeInTheDocument()
  })
})

describe('auto-fitting timeline', () => {
  const day = (offset: number) => new Date(Date.now() + offset * 86400000).toISOString().slice(0, 10)
  const setup = async (list: Record<string, unknown>[], projectOverride: Record<string, unknown> = {}) => {
    signIn()
    mock.onGet('/api/v1/projects/p1').reply(200, { success: true, data: { ...project, start_date: day(-1), ...projectOverride } })
    allocations = list
    renderAt('/projects/p1?view=timeline')
    await screen.findAllByTestId('timeline-bar')
  }
  const granularity = () => document.querySelector('[data-granularity]')?.getAttribute('data-granularity')
  const al = (id: string, user: string, start: string, end?: string) => ({
    id, project_id: 'p1', user_id: user, user_name: user === 'e1' ? 'Ada Doe' : 'Bob Doe', allocation_percent: 50, start_date: start, ...(end ? { end_date: end } : {}),
  })

  it('shows daily columns and a 7-column bar for a one-week allocation', async () => {
    await setup([al('w1', 'e1', day(1), day(7))])
    expect(granularity()).toBe('day')
    const bar = screen.getByRole('img', { name: /Ada Doe, 50% allocation/ })
    const cols = screen.getAllByTestId('axis-label').length
    expect(parseFloat(bar.style.width)).toBeCloseTo((7 / cols) * 100, 1)
    expect(screen.getAllByTestId('axis-label')[0].textContent).toMatch(/^[A-Z][a-z]{2} \d{1,2}$/)
    expect(bar).toHaveTextContent('50%')
    expect(screen.queryByText(/Outside the visible range/)).not.toBeInTheDocument()
  })

  it('ignores an old project start: daily columns, 7-column bar, no markers outside the range', async () => {
    await setup([al('w2', 'e1', day(1), day(7))], { start_date: day(-120), end_date: day(-60) })
    expect(granularity()).toBe('day')
    const bar = screen.getByRole('img', { name: /Ada Doe, 50% allocation/ })
    expect(parseFloat(bar.style.width)).toBeCloseTo((7 / screen.getAllByTestId('axis-label').length) * 100, 1)
    expect(document.querySelectorAll('[title="Project start"], [title="Project end"]')).toHaveLength(0)
  })

  it('shows project markers that fall inside the range', async () => {
    await setup([al('w3', 'e1', day(1), day(7))], { start_date: day(3) })
    expect(document.querySelectorAll('[title="Project start"]').length).toBeGreaterThan(0)
  })

  it('uses weekly columns for a multi-month span', async () => {
    await setup([al('m1', 'e1', day(1), day(90))])
    expect(granularity()).toBe('week')
    expect(screen.getAllByTestId('axis-label')[0].textContent).toMatch(/^[A-Z][a-z]{2} \d{1,2}$/)
  })

  it('uses monthly columns for a long span', async () => {
    await setup([al('l1', 'e1', day(1), day(400))])
    expect(granularity()).toBe('month')
    expect(screen.getAllByTestId('axis-label')[0].textContent).toMatch(/\d{4}$/)
  })

  it('shows an arrow for open-ended allocations and fits every row', async () => {
    await setup([al('o1', 'e1', day(1)), al('o2', 'e2', day(3), day(9))])
    expect(screen.getAllByTestId('timeline-bar')).toHaveLength(2)
    const open = screen.getByRole('img', { name: /Ada Doe.*open-ended/ })
    expect(open).toHaveTextContent('→')
    expect(screen.getByRole('img', { name: /Bob Doe/ })).not.toHaveTextContent('→')
  })

  it('shows a legend with only the entries that apply', async () => {
    await setup([al('g1', 'e1', day(-3), day(3))], { start_date: day(-1), end_date: day(2) })
    const legend = within(screen.getByRole('list', { name: 'Timeline legend' }))
    expect(legend.getByText('Today')).toBeInTheDocument()
    expect(legend.getByText('Project start / end')).toBeInTheDocument()
    expect(legend.queryByText(/not in view/)).not.toBeInTheDocument()
    expect(legend.getByText('Active')).toBeInTheDocument()
    expect(legend.queryByText('Past')).not.toBeInTheDocument()
    expect(legend.queryByText('Upcoming')).not.toBeInTheDocument()
    expect(legend.queryByText('No end date (continues)')).not.toBeInTheDocument()
  })

  it('shows the open-ended entry and keeps marker entries flagged when not drawn', async () => {
    await setup([al('g2', 'e1', day(20)), al('g3', 'e2', day(-60), day(-50))], { start_date: day(-200), end_date: day(-190) })
    const legend = within(screen.getByRole('list', { name: 'Timeline legend' }))
    expect(legend.getByText('No end date (continues)')).toBeInTheDocument()
    expect(legend.getByText('Upcoming')).toBeInTheDocument()
    expect(legend.getByText('Past')).toBeInTheDocument()
    expect(legend.getByText('Project start / end (not in view)')).toBeInTheDocument()
    expect(legend.getByText('Today')).toBeInTheDocument()
  })

  it('marks Today as not in view when outside the range', async () => {
    await setup([al('g4', 'e1', day(10), day(15))], { start_date: day(10), end_date: day(15) })
    const legend = within(screen.getByRole('list', { name: 'Timeline legend' }))
    expect(legend.getByText('Today (not in view)')).toBeInTheDocument()
    expect(legend.getByText('Project start / end')).toBeInTheDocument()
  })

  it('shows a plain empty state when no allocations match the filter', async () => {
    signIn()
    allocations = [al('p1a', 'e1', day(-10), day(-5))]
    renderAt('/projects/p1?view=timeline&state=active')
    expect(await screen.findByText('No active allocations.')).toBeInTheDocument()
    expect(screen.queryByTestId('timeline-bar')).not.toBeInTheDocument()
  })
})

describe('employee combobox', () => {
  async function openDialog() {
    signIn()
    renderAt('/projects/p1')
    await userEvent.click(await screen.findByRole('button', { name: 'Add allocation' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    return { form, combo: within(form).getByRole('combobox', { name: /Employee/ }) }
  }

  it('searches by employee code and by email, showing code and email under the name', async () => {
    const { form, combo } = await openDialog()
    await userEvent.type(combo, 'e-e1')
    const opt = await within(form).findByRole('option', { name: /Ada Doe/ })
    expect(opt).toHaveTextContent('E-E1')
    expect(opt).toHaveTextContent('ada@x.com')
    await waitFor(() => expect(within(form).queryByRole('option', { name: /Bob Doe/ })).not.toBeInTheDocument())
    await userEvent.clear(combo)
    await userEvent.type(combo, 'bob@x')
    expect(await within(form).findByRole('option', { name: /Bob Doe/ })).toBeInTheDocument()
  })

  it('requests include=allocation and shows an allocation hint per option, amber with text when full', async () => {
    const { form, combo } = await openDialog()
    await userEvent.type(combo, 'doe')
    const ada = await within(form).findByRole('option', { name: /Ada Doe/ })
    expect(ada).toHaveTextContent('60% allocated')
    expect(within(ada).getByText('60% allocated')).not.toHaveClass('text-amber-800')
    const bob = within(form).getByRole('option', { name: /Bob Doe/ })
    expect(within(bob).getByText(/Fully allocated/)).toHaveClass('text-amber-800')
    expect(mock.history.get.some((r) => r.url === '/api/v1/employees' && r.params?.include === 'allocation')).toBe(true)
  })

  it('warns about the resulting total after selecting, without blocking submit', async () => {
    mock.onPost('/api/v1/projects/p1/allocations').reply(201, { success: true, data: { id: 'al9', over_allocated: true, user_total_percent: 90 } })
    const { form, combo } = await openDialog()
    await userEvent.type(combo, 'ada')
    await userEvent.click(await within(form).findByRole('option', { name: /Ada Doe/ }))
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '30')
    expect(await within(form).findByText('Resulting total: 90% allocated.')).toBeInTheDocument()
    await userEvent.clear(within(form).getByLabelText(/^Allocation %/))
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '50')
    expect(await within(form).findByText(/would bring Ada Doe to 110% allocated in total/)).toBeInTheDocument()
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-03-01')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
  })

  it('does not request allocation data outside the picker', async () => {
    signIn()
    renderAt('/projects/p1')
    await screen.findByRole('link', { name: 'Ada Doe' })
    expect(mock.history.get.filter((r) => r.url === '/api/v1/employees').every((r) => r.params?.include === undefined)).toBe(true)
  })

  it('supports keyboard selection, a clear button, and an empty state', async () => {
    const { form, combo } = await openDialog()
    await userEvent.type(combo, 'doe')
    await within(form).findByRole('option', { name: /Bob Doe/ })
    await userEvent.keyboard('{ArrowDown}{Enter}')
    expect(combo).toHaveValue('Bob Doe (E-E2)')
    await userEvent.click(within(form).getByRole('button', { name: 'Clear employee' }))
    expect(combo).toHaveValue('')
    await userEvent.type(combo, 'zzz')
    expect(await within(form).findByText('No matching employees')).toBeInTheDocument()
    await userEvent.keyboard('{Escape}')
    expect(within(form).queryByRole('listbox')).not.toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Add allocation' })).toBeInTheDocument()
  })

  it('requires an employee before submitting', async () => {
    const { form } = await openDialog()
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '50')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-03-01')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    expect(await within(form).findByText('Select an employee')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })
})

describe('employee allocations section', () => {
  it('shows the total and an over-allocation warning', async () => {
    signIn()
    userAllocs = {
      total_percent: 130, over_allocated: true,
      allocations: [
        { id: 'a1', project_id: 'p1', project_name: 'Apollo', project_code: 'APL', project_status: 'active', allocation_percent: 80, start_date: '2026-01-01', counts_toward_total: true },
        { id: 'a2', project_id: 'p2', project_name: 'Zeus', project_code: 'ZEU', project_status: 'active', allocation_percent: 50, start_date: '2026-01-01', counts_toward_total: true },
      ],
    }
    renderAt('/employees/e1/allocations')
    expect(await screen.findByText('Total 130% - over-allocated')).toBeInTheDocument()
    expect(screen.getByText(/Over-allocated: total allocation is 130%/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Zeus' })).toHaveAttribute('href', '/projects/p2')
  })

  it('shows no warning at or below 100%', async () => {
    signIn()
    userAllocs = { total_percent: 100, over_allocated: false, allocations: [] }
    renderAt('/employees/e1/allocations')
    expect(await screen.findByText('Total 100%')).toBeInTheDocument()
    expect(screen.queryByText(/Over-allocated/)).not.toBeInTheDocument()
  })

  const day = (offset: number) => new Date(Date.now() + offset * 86400000).toISOString().slice(0, 10)
  const allocRow = (id: string, start: string, end?: string) => ({
    id, project_id: 'p1', project_name: 'Apollo', project_code: 'APL', project_status: 'active',
    allocation_percent: 40, start_date: start, end_date: end, counts_toward_total: true,
  })

  it('requests the selected scope, labels rows by state and hides the total for upcoming/past', async () => {
    signIn()
    userAllocs = { total_percent: 40, over_allocated: false, allocations: [allocRow('a1', day(-5))] }
    renderAt('/employees/e1/allocations')
    expect(await screen.findByText('Total 40%')).toBeInTheDocument()
    const scopeOf = () => mock.history.get.filter((r) => r.url === '/api/v1/employees/e1/allocations').map((r) => r.params?.scope)
    expect(scopeOf()).toContain('active')
    userAllocs = { total_percent: 0, over_allocated: false, allocations: [allocRow('a2', day(10))] }
    await userEvent.selectOptions(await screen.findByLabelText('Show'), 'upcoming')
    expect(await screen.findByText('Upcoming', { selector: 'span' })).toBeInTheDocument()
    expect(scopeOf()).toContain('upcoming')
    expect(screen.queryByText(/^Total /)).not.toBeInTheDocument()
    userAllocs = { total_percent: 0, over_allocated: false, allocations: [] }
    await userEvent.selectOptions(await screen.findByLabelText('Show'), 'past')
    expect(await screen.findByText('No past project allocations.')).toBeInTheDocument()
    expect(scopeOf()).toContain('past')
  })

  it('shows the server total and warning under All without summing rows', async () => {
    signIn()
    userAllocs = { total_percent: 130, over_allocated: true, allocations: [allocRow('a1', day(-5))] }
    renderAt('/employees/e1/allocations')
    await screen.findByText('Total 130% - over-allocated')
    userAllocs = {
      total_percent: 40, over_allocated: false,
      allocations: [allocRow('a1', day(-5)), allocRow('a2', day(10)), allocRow('a3', day(-30), day(-10))],
    }
    await userEvent.selectOptions(await screen.findByLabelText('Show'), 'all')
    expect(await screen.findByText('Total 40%')).toBeInTheDocument()
    expect(screen.queryByText(/Over-allocated/)).not.toBeInTheDocument()
    const rows = screen.getAllByRole('row').slice(1)
    expect(rows).toHaveLength(3)
    expect(rows.map((r) => /Upcoming/.test(r.textContent ?? '') ? 'upcoming' : /Past/.test(r.textContent ?? '') ? 'past' : 'active')).toEqual(['active', 'upcoming', 'past'])
  })

  it('switches to Upcoming after allocating with a future start date', async () => {
    signIn()
    userAllocs = { allocations: [], total_percent: 0, over_allocated: false }
    mock.onGet('/api/v1/projects').reply(() => [200, { success: true, data: [project], meta: { page: 1, page_size: 10, total_items: 1, total_pages: 1 } }])
    mock.onPost('/api/v1/projects/p1/allocations').reply(201, { success: true, data: { id: 'al9', over_allocated: false, user_total_percent: 0 } })
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Allocate to project' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    await userEvent.type(within(form).getByRole('combobox', { name: /Project/ }), 'apo')
    await userEvent.click(await within(form).findByRole('option', { name: /Apollo/ }))
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '40')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2099-01-01')
    userAllocs = { total_percent: 0, over_allocated: false, allocations: [allocRow('al9', '2099-01-01')] }
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    await waitFor(() => expect(screen.getByLabelText('Show')).toHaveValue('upcoming'))
    expect(await screen.findByText('Upcoming', { selector: 'span' })).toBeInTheDocument()
  })
})

describe('allocation review fixes', () => {
  const day = (offset: number) => new Date(Date.now() + offset * 86400000).toISOString().slice(0, 10)
  const row = (id: string, start: string, end?: string) => ({
    id, project_id: 'p1', project_name: 'Apollo', project_code: 'APL', project_status: 'active',
    allocation_percent: 40, start_date: start, end_date: end, counts_toward_total: true,
  })
  const projectsReply = (list: Record<string, unknown>[]) =>
    mock.onGet('/api/v1/projects').reply((config) => {
      const q = String(config.params?.search ?? '').toLowerCase()
      const hit = list.filter((p) => !q || String(p.name).toLowerCase().includes(q))
      return [200, { success: true, data: hit, meta: { page: 1, page_size: 10, total_items: hit.length, total_pages: 1 } }]
    })

  it('disables "End allocation" for upcoming rows with a hint', async () => {
    signIn()
    userAllocs = { total_percent: 0, over_allocated: false, allocations: [row('a1', day(10))] }
    renderAt('/employees/e1/allocations')
    await userEvent.selectOptions(await screen.findByLabelText('Show'), 'upcoming')
    await userEvent.click(await screen.findByRole('button', { name: 'Actions for allocation Apollo' }))
    const item = await screen.findByRole('menuitem', { name: 'End allocation' })
    expect(item).toBeDisabled()
    expect(item).toHaveAttribute('title', expect.stringMatching(/Upcoming/))
  })

  it('keeps "End allocation" enabled for a current allocation', async () => {
    signIn()
    userAllocs = { total_percent: 40, over_allocated: false, allocations: [row('a1', day(-5))] }
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Actions for allocation Apollo' }))
    expect(await screen.findByRole('menuitem', { name: 'End allocation' })).toBeEnabled()
  })

  it('switches to Past after saving an allocation that already ended', async () => {
    signIn()
    userAllocs = { allocations: [], total_percent: 0, over_allocated: false }
    projectsReply([project])
    mock.onPost('/api/v1/projects/p1/allocations').reply(201, { success: true, data: { id: 'al8', over_allocated: false, user_total_percent: 0 } })
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Allocate to project' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    await userEvent.type(within(form).getByRole('combobox', { name: /Project/ }), 'apo')
    await userEvent.click(await within(form).findByRole('option', { name: /Apollo/ }))
    await userEvent.type(within(form).getByLabelText(/^Allocation %/), '40')
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-02-01')
    await userEvent.type(within(form).getByLabelText(/^End date/), '2026-02-28')
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    await waitFor(() => expect(screen.getByLabelText('Show')).toHaveValue('past'))
  })

  it('validates an edited allocation against the fetched project window', async () => {
    signIn()
    userAllocs = { total_percent: 40, over_allocated: false, allocations: [row('a1', day(-5))] }
    mock.onGet('/api/v1/projects/p1').reply(200, { success: true, data: { ...project, end_date: '2026-12-31' } })
    renderAt('/employees/e1/allocations')
    await screen.findByRole('link', { name: 'Apollo' })
    await pickRowAction('Actions for allocation Apollo', 'Edit')
    const form = screen.getByRole('form', { name: 'Edit allocation' })
    expect(await within(form).findByText(/Project runs 2026-01-01 to 2026-12-31/)).toBeInTheDocument()
    const end = within(form).getByLabelText(/^End date/)
    await userEvent.type(end, '2027-03-01')
    await userEvent.click(within(form).getByRole('button', { name: 'Save changes' }))
    expect(await within(form).findByText('End date must not be after the project end date')).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
  })

  it('shows "Select a project" together with other field errors on submit', async () => {
    signIn()
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Allocate to project' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    await userEvent.click(within(form).getByRole('button', { name: 'Add allocation' }))
    expect(await within(form).findByText('Allocation is required')).toBeInTheDocument()
    expect(within(form).getByText('Select a project')).toBeInTheDocument()
  })

  it('re-validates entered dates when the selected project changes', async () => {
    signIn()
    projectsReply([
      { ...project, id: 'p1', name: 'Apollo' },
      { ...project, id: 'p2', name: 'Borealis', code: 'BOR', start_date: '2026-06-01' },
    ])
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Allocate to project' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    const combo = within(form).getByRole('combobox', { name: /Project/ })
    await userEvent.type(combo, 'apo')
    await userEvent.click(await within(form).findByRole('option', { name: /Apollo/ }))
    await userEvent.type(within(form).getByLabelText(/^Start date/), '2026-03-01')
    expect(within(form).queryByText(/must not be before/)).not.toBeInTheDocument()
    await userEvent.click(within(form).getByRole('button', { name: 'Clear project' }))
    await userEvent.type(combo, 'bor')
    await userEvent.click(await within(form).findByRole('option', { name: /Borealis/ }))
    expect(await within(form).findByText('Start date must not be before the project start date')).toBeInTheDocument()
    // Going back to a window that fits clears the stale error.
    await userEvent.click(within(form).getByRole('button', { name: 'Clear project' }))
    await userEvent.type(combo, 'apo')
    await userEvent.click(await within(form).findByRole('option', { name: /Apollo/ }))
    await waitFor(() => expect(within(form).queryByText(/must not be before/)).not.toBeInTheDocument())
  })

  it('project combobox: ArrowDown on an empty list does not break Enter selection', async () => {
    signIn()
    projectsReply([project])
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Allocate to project' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    const combo = within(form).getByRole('combobox', { name: /Project/ })
    await userEvent.type(combo, 'apo')
    await userEvent.keyboard('{ArrowDown}{ArrowDown}')
    await within(form).findByRole('option', { name: /Apollo/ })
    await userEvent.keyboard('{Enter}')
    await waitFor(() => expect(combo).toHaveValue('Apollo (APL)'))
  })

  it('edit modal shows a loading state and cannot submit before the project loads', async () => {
    signIn()
    userAllocs = { total_percent: 40, over_allocated: false, allocations: [row('a1', day(-5))] }
    let release: () => void = () => {}
    const gate = new Promise<void>((r) => { release = r })
    mock.onGet('/api/v1/projects/p1').reply(async () => { await gate; return [200, { success: true, data: project }] })
    renderAt('/employees/e1/allocations')
    await screen.findByRole('link', { name: 'Apollo' })
    await pickRowAction('Actions for allocation Apollo', 'Edit')
    expect(await screen.findByText('Loading project…')).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Edit allocation' })).not.toBeInTheDocument()
    release()
    expect(await screen.findByRole('form', { name: 'Edit allocation' })).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
  })

  it('edit modal shows an inline error with retry and close when the project fails to load', async () => {
    signIn()
    userAllocs = { total_percent: 40, over_allocated: false, allocations: [row('a1', day(-5))] }
    let fail = true
    mock.onGet('/api/v1/projects/p1').reply(() => (fail ? [500, { success: false, error: { message: 'boom' } }] : [200, { success: true, data: project }]))
    renderAt('/employees/e1/allocations')
    await screen.findByRole('link', { name: 'Apollo' })
    await pickRowAction('Actions for allocation Apollo', 'Edit')
    const retry = await screen.findByRole('button', { name: 'Retry' })
    expect(screen.queryByRole('form', { name: 'Edit allocation' })).not.toBeInTheDocument()
    fail = false
    await userEvent.click(retry)
    expect(await screen.findByRole('form', { name: 'Edit allocation' })).toBeInTheDocument()
  })

  it('switches to Past after editing an allocation so it has ended', async () => {
    signIn()
    userAllocs = { total_percent: 40, over_allocated: false, allocations: [{ ...row('a1', day(-5)), user_id: 'e1' }] }
    mock.onPatch('/api/v1/allocations/a1').reply(200, { success: true, data: { id: 'a1', over_allocated: false, user_total_percent: 0 } })
    renderAt('/employees/e1/allocations')
    await screen.findByRole('link', { name: 'Apollo' })
    await pickRowAction('Actions for allocation Apollo', 'Edit')
    const form = await screen.findByRole('form', { name: 'Edit allocation' })
    await userEvent.type(within(form).getByLabelText(/^End date/), day(-1))
    await userEvent.click(within(form).getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(screen.getByLabelText('Show')).toHaveValue('past'))
  })

  it('pristine edit form shows no errors on load', async () => {
    signIn()
    // Prefilled start predates the project window, so it would be invalid if validated eagerly.
    userAllocs = { total_percent: 40, over_allocated: false, allocations: [row('a1', '2025-06-01')] }
    renderAt('/employees/e1/allocations')
    await screen.findByRole('link', { name: 'Apollo' })
    await pickRowAction('Actions for allocation Apollo', 'Edit')
    const form = await screen.findByRole('form', { name: 'Edit allocation' })
    await within(form).findByText(/Project runs 2026-01-01/)
    expect(within(form).queryByText(/must not be before/)).not.toBeInTheDocument()
    expect(within(form).queryByRole('alert')).not.toBeInTheDocument()
  })

  it('project combobox: a background refetch does not reset the highlight', async () => {
    signIn()
    projectsReply([project, { ...project, id: 'p2', name: 'Apollo Two', code: 'AP2' }])
    renderAt('/employees/e1/allocations')
    await userEvent.click(await screen.findByRole('button', { name: 'Allocate to project' }))
    const form = screen.getByRole('form', { name: 'Add allocation' })
    const combo = within(form).getByRole('combobox', { name: /Project/ })
    await userEvent.type(combo, 'apo')
    await within(form).findByRole('option', { name: /Apollo Two/ })
    await userEvent.keyboard('{ArrowDown}')
    expect(within(form).getByRole('option', { name: /Apollo Two/ })).toHaveAttribute('aria-selected', 'true')
    // Same ids, new data reference.
    const searches = () => mock.history.get.filter((r) => r.url === '/api/v1/projects' && r.params?.search === 'apo').length
    const before = searches()
    document.dispatchEvent(new Event('visibilitychange'))
    await waitFor(() => expect(searches()).toBeGreaterThan(before))
    await new Promise((r) => setTimeout(r, 50))
    expect(within(form).getByRole('option', { name: /Apollo Two/ })).toHaveAttribute('aria-selected', 'true')
  })
})
