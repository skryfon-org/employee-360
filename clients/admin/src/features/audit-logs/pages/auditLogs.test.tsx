import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import { pickRowAction } from '../../../test/rowActions'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'

let mock: MockAdapter

const entry = (id: string, over: Record<string, unknown> = {}) => ({
  id,
  action: 'department.create',
  entity_type: 'department',
  entity_id: '11111111-2222-3333-4444-555555555555',
  actor: { id: 'u1', email: 'ada@x.com', name: 'Ada Lovelace' },
  metadata: { name: 'Engineering' },
  created_at: '2026-03-04T10:00:00Z',
  ...over,
})
let rows: Record<string, unknown>[] = []
let meta = { page: 1, page_size: 20, total_items: 2, total_pages: 1 }

function renderAt(path = '/audit-logs') {
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

const lastParams = () => mock.history.get.filter((r) => r.url === '/api/v1/audit-logs').at(-1)?.params

beforeEach(() => {
  rows = [
    entry('a1'),
    entry('a2', { action: 'tenant.domain.add', entity_type: 'tenant_domain', actor: null, metadata: {}, entity_id: null }),
  ]
  meta = { page: 1, page_size: 20, total_items: 2, total_pages: 1 }
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/audit-logs').reply(() => [200, { success: true, data: rows, meta }])
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
})
afterEach(() => mock.restore())

describe('audit logs', () => {
  it('shows a skeleton, then rows, actor, labels and sidebar entry', async () => {
    signIn()
    renderAt()
    expect(screen.getByRole('status')).toHaveTextContent('Loading audit log…')
    expect(await screen.findByText('Created department')).toBeInTheDocument()
    expect(screen.getByText('Ada Lovelace')).toBeInTheDocument()
    expect(screen.getByText('ada@x.com')).toBeInTheDocument()
    expect(screen.getByText('department.create')).toBeInTheDocument()
    expect(screen.getByText('System')).toBeInTheDocument()
    expect(screen.getByText('Added organization domain')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Audit log' })).toHaveAttribute('href', '/audit-logs')
    expect(lastParams()).toMatchObject({ page: 1, page_size: 20 })
  })

  it('hides the sidebar entry from non-admin roles', async () => {
    signIn(['employee'])
    renderAt('/')
    await waitFor(() => expect(screen.queryByRole('link', { name: 'Audit log' })).not.toBeInTheDocument())
  })

  it('expands metadata as plain text without injecting markup', async () => {
    rows = [entry('a1', { metadata: { name: '<img src=x onerror=alert(1)>' } })]
    signIn()
    const { container } = renderAt()
    await screen.findByText('Created department')
    await pickRowAction(/Actions for/, 'View details')
    const pre = container.querySelector('pre')
    expect(pre?.textContent).toContain('<img src=x onerror=alert(1)>')
    expect(container.querySelector('img')).toBeNull()
    await pickRowAction(/Actions for/, 'Hide details')
    expect(container.querySelector('pre')).toBeNull()
  })

  it('shows a placeholder when metadata is empty', async () => {
    signIn()
    renderAt()
    await screen.findByText('System')
    await pickRowAction(/Actions for Added organization domain/i, 'View details')
    expect(screen.getByText('No additional details.')).toBeInTheDocument()
  })

  it('sends filter params and can clear them', async () => {
    signIn()
    renderAt()
    await screen.findByText('Created department')
    await userEvent.selectOptions(screen.getByLabelText('Entity type'), 'position')
    await waitFor(() => expect(lastParams()).toMatchObject({ entity_type: 'position', page: 1 }))
    await userEvent.selectOptions(screen.getByLabelText('Action'), 'department.')
    await waitFor(() => expect(lastParams()).toMatchObject({ entity_type: 'position', action: 'department.' }))
    await userEvent.click(screen.getByRole('button', { name: 'Clear filters' }))
    await waitFor(() => expect(lastParams()?.entity_type).toBeUndefined())
    expect(lastParams()?.action).toBeUndefined()
  })

  it('sends RFC3339 from/to for a date range', async () => {
    signIn()
    renderAt()
    await screen.findByText('Created department')
    await userEvent.type(screen.getByLabelText('From'), '2026-03-01')
    await userEvent.type(screen.getByLabelText('To'), '2026-03-05')
    await userEvent.click(screen.getByRole('button', { name: 'Apply dates' }))
    await waitFor(() => expect(lastParams()?.from).toBeDefined())
    const p = lastParams()
    expect(p.from).toBe(new Date('2026-03-01T00:00:00').toISOString().replace('.000Z', 'Z'))
    expect(p.to).toBe(new Date('2026-03-05T23:59:59').toISOString().replace('.000Z', 'Z'))
  })

  it('rejects a from date after the to date without requesting', async () => {
    signIn()
    renderAt()
    await screen.findByText('Created department')
    const before = mock.history.get.length
    await userEvent.type(screen.getByLabelText('From'), '2026-03-10')
    await userEvent.type(screen.getByLabelText('To'), '2026-03-05')
    await userEvent.click(screen.getByRole('button', { name: 'Apply dates' }))
    expect(await screen.findByText('End date must be on or after the start date')).toBeInTheDocument()
    expect(mock.history.get.length).toBe(before)
  })

  it('paginates', async () => {
    meta = { page: 1, page_size: 20, total_items: 45, total_pages: 3 }
    signIn()
    renderAt()
    await screen.findByText('Created department')
    expect(screen.getByText('Showing 1–45 of 45'.replace('1–45', '1–20'))).toBeInTheDocument()
    await userEvent.click(within(screen.getByRole('navigation', { name: 'Pagination' })).getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(lastParams()).toMatchObject({ page: 2 }))
  })

  it('shows an empty state', async () => {
    rows = []
    meta = { page: 1, page_size: 20, total_items: 0, total_pages: 0 }
    signIn()
    renderAt()
    expect(await screen.findByText('No audit entries yet')).toBeInTheDocument()
  })

  it('shows an error with retry, including 400s', async () => {
    mock.reset()
    let calls = 0
    mock.onGet('/api/v1/audit-logs').reply(() => {
      calls++
      return calls === 1
        ? [400, { success: false, error: { code: 'VALIDATION_ERROR', message: 'invalid from' } }]
        : [200, { success: true, data: rows, meta }]
    })
    signIn()
    renderAt()
    expect(await screen.findByRole('alert')).toHaveTextContent('invalid from')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('Created department')).toBeInTheDocument()
  })
})
