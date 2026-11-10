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

let mock: MockAdapter

const dept = (id: string, name: string, description = '') => ({
  id,
  name,
  description,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-02-03T00:00:00Z',
  is_active: !name.startsWith('Old'),
})
let rows: Record<string, unknown>[] = []

function renderAt(path = '/positions') {
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

const conflict = (message: string) => [409, { success: false, error: { code: 'CONFLICT', message } }]

beforeEach(() => {
  rows = [dept('d1', 'Engineering', 'Builds things'), dept('d2', 'Finance')]
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/positions').reply(() => [
    200,
    { success: true, data: rows, meta: { page: 1, page_size: 20, total_items: rows.length, total_pages: 1 } },
  ])
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
  toast.dismiss()
})
afterEach(() => mock.restore())

describe('positions', () => {
  it('shows a skeleton, then the list with a sidebar entry', async () => {
    signIn()
    renderAt()
    expect(screen.getByRole('status')).toHaveTextContent('Loading positions…')
    expect(await screen.findByText('Engineering')).toBeInTheDocument()
    const table = screen.getByRole('table')
    for (const h of ['Name', 'Description', 'Status', 'Updated']) {
      expect(within(table).getByRole('columnheader', { name: h })).toBeInTheDocument()
    }
    expect(within(table).getByText('Builds things')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Positions' })).toHaveAttribute('href', '/positions')
    expect(mock.history.get.find((r) => r.url === '/api/v1/positions')?.params).toMatchObject({ page: 1, page_size: 20 })
  })

  it('shows Active/Inactive badges', async () => {
    signIn()
    rows = [dept('d1', 'Engineering'), dept('d2', 'Old Ops')]
    renderAt()
    await screen.findByText('Engineering')
    const table = screen.getByRole('table')
    expect(within(table).getByText('Active')).toBeInTheDocument()
    expect(within(table).getByText('Inactive')).toBeInTheDocument()
  })

  it('filters by status via is_active, keeps it in the URL and resets to page 1', async () => {
    signIn()
    renderAt('/positions?page=2&is_active=true')
    await screen.findByText('Engineering')
    expect(screen.getByLabelText('Status')).toHaveValue('true')
    expect(mock.history.get.at(-1)?.params).toMatchObject({ page: 2, is_active: true })
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'false')
    await waitFor(() => expect(mock.history.get.at(-1)?.params).toMatchObject({ page: 1, is_active: false }))
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'All')
    await waitFor(() => expect(mock.history.get.at(-1)?.params.is_active).toBeUndefined())
  })

  it('creates inactive when the Active checkbox is cleared (default checked)', async () => {
    signIn()
    mock.onPost('/api/v1/positions').reply(201, { success: true, data: dept('d4', 'Ops') })
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Create position' }))
    const dialog = screen.getByRole('dialog')
    const active = within(dialog).getByLabelText(/^Active/)
    expect(active).toBeChecked()
    await userEvent.type(within(dialog).getByLabelText('Name'), 'Ops')
    await userEvent.click(active)
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ name: 'Ops', description: '', is_active: false })
  })

  it('paginates', async () => {
    signIn()
    mock.onGet('/api/v1/positions').reply((cfg) => [
      200,
      { success: true, data: rows, meta: { page: cfg.params.page, page_size: cfg.params.page_size, total_items: 45, total_pages: 3 } },
    ])
    renderAt()
    expect(await screen.findByText('Showing 1–20 of 45')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Showing 21–40 of 45')).toBeInTheDocument()
  })

  it('shows an empty state', async () => {
    signIn()
    rows = []
    renderAt()
    expect(await screen.findByText('No positions yet')).toBeInTheDocument()
  })

  it('shows an error state on failure', async () => {
    signIn()
    mock.onGet('/api/v1/positions').reply(500, { success: false, error: { message: 'List exploded' } })
    renderAt()
    expect(await screen.findByText('List exploded')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })

  it('validates the create form before calling the API', async () => {
    signIn()
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Create position' }))
    const dialog = screen.getByRole('dialog', { name: 'Create position' })
    await userEvent.type(within(dialog).getByLabelText('Name'), '   ')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create' }))
    expect(await within(dialog).findByText('Name is required')).toBeInTheDocument()

    await userEvent.type(within(dialog).getByLabelText('Name'), 'x'.repeat(101))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create' }))
    expect(await within(dialog).findByText('Name must be at most 100 characters')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })

  it('counts characters, not UTF-16 units, for length limits', async () => {
    signIn()
    mock.onPost('/api/v1/positions').reply(201, { success: true, data: dept('d9', '😀'.repeat(100)) })
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Create position' }))
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByLabelText('Name'))
    await userEvent.paste('😀'.repeat(100))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
  })

  it('creates a position (trimmed) and refreshes the list', async () => {
    signIn()
    mock.onPost('/api/v1/positions').reply((cfg) => {
      const body = JSON.parse(cfg.data)
      rows = [...rows, dept('d3', body.name, body.description)]
      return [201, { success: true, data: rows[rows.length - 1] }]
    })
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Create position' }))
    const dialog = screen.getByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText('Name'), '  Legal  ')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create' }))
    expect(await screen.findByText('Legal')).toBeInTheDocument()
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ name: 'Legal', description: '', is_active: true })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('shows a duplicate-name 409 on the name field and keeps the modal open', async () => {
    signIn()
    mock.onPost('/api/v1/positions').reply(...(conflict('a position with this name already exists') as [number, unknown]))
    renderAt()
    await userEvent.click(await screen.findByRole('button', { name: 'Create position' }))
    const dialog = screen.getByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText('Name'), 'engineering')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create' }))
    const msg = await within(dialog).findByText('A position with this name already exists.')
    expect(msg).toHaveAttribute('role', 'alert')
    expect(within(dialog).getByLabelText('Name')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('edits a position with prefilled values', async () => {
    signIn()
    mock.onPut('/api/v1/positions/d1').reply((cfg) => {
      const body = JSON.parse(cfg.data)
      rows = rows.map((r) => (r.id === 'd1' ? { ...r, ...body } : r))
      return [200, { success: true, data: rows[0] }]
    })
    renderAt()
    await pickRowAction('Actions for position Engineering', 'Edit')
    const dialog = screen.getByRole('dialog', { name: 'Edit position' })
    const name = within(dialog).getByLabelText('Name')
    expect(name).toHaveValue('Engineering')
    expect(within(dialog).getByLabelText('Description')).toHaveValue('Builds things')
    await userEvent.clear(name)
    await userEvent.type(name, 'Platform')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText('Platform')).toBeInTheDocument()
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ name: 'Platform', description: 'Builds things', is_active: true })
  })

  it('shows duplicate-name 409 when editing', async () => {
    signIn()
    mock.onPut('/api/v1/positions/d1').reply(...(conflict('dup') as [number, unknown]))
    renderAt()
    await pickRowAction('Actions for position Engineering', 'Edit')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText('A position with this name already exists.')).toBeInTheDocument()
  })

  it('deletes only after confirmation and refreshes the list', async () => {
    signIn()
    mock.onDelete('/api/v1/positions/d2').reply(() => {
      rows = rows.filter((r) => r.id !== 'd2')
      return [200, { success: true }]
    })
    renderAt()
    await pickRowAction('Actions for position Finance', 'Delete')
    const dialog = screen.getByRole('dialog', { name: 'Delete position' })
    expect(mock.history.delete).toHaveLength(0)
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(screen.queryByText('Finance')).toBeNull())
    expect(mock.history.delete).toHaveLength(1)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('cancels delete without calling the API', async () => {
    signIn()
    renderAt()
    await pickRowAction('Actions for position Finance', 'Delete')
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(mock.history.delete).toHaveLength(0)
  })

  it('shows an in-use message in the modal on delete 409', async () => {
    signIn()
    mock.onDelete('/api/v1/positions/d1').reply(...(conflict('position cannot be deleted') as [number, unknown]))
    renderAt()
    await pickRowAction('Actions for position Engineering', 'Delete')
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/still assigned to users or pending invitations/)
    expect(within(screen.getByRole('table')).getByText('Engineering')).toBeInTheDocument()
  })
})
