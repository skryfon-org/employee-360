import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import MyProjectsPage from './MyProjectsPage'

let mock: MockAdapter

function Loc() {
  return <span data-testid="loc">{useLocation().search}</span>
}

function mockNarrow(narrow: boolean) {
  vi.stubGlobal('matchMedia', (q: string) => ({
    matches: narrow && q.includes('max-width'),
    media: q,
    addEventListener: () => {},
    removeEventListener: () => {},
  }))
}

function renderPage(url = '/') {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[url]}>
        <Loc />
        <MyProjectsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  mock = new MockAdapter(apiClient)
})
afterEach(() => {
  mock.restore()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

function pick(label: string) {
  const select = screen.getByRole('combobox', { name: 'Allocation status' }) as HTMLSelectElement
  const opt = within(select).getAllByRole('option').find((o) => o.textContent?.startsWith(label)) as HTMLOptionElement
  fireEvent.change(select, { target: { value: opt.value } })
}

function body(allocations: unknown[]) {
  return { success: true, data: { allocations, total_percent: 0, over_allocated: false } }
}

describe('My projects', () => {
  it('lists active allocations with project, percent, role and dates', async () => {
    mock.onGet('/api/v1/me/projects', { params: { scope: 'active' } }).reply(200, {
      success: true,
      data: {
        total_percent: 100,
        over_allocated: false,
        allocations: [
          { id: 'a1', project_id: 'p1', project_name: 'Apollo', project_code: 'APL', project_status: 'active', allocation_percent: 60, role_on_project: 'Developer', start_date: '2026-01-01', end_date: '2026-12-31' },
          { id: 'a2', project_id: 'p2', project_name: 'Zeus', project_code: 'ZEU', project_status: 'on_hold', allocation_percent: 40, start_date: '2026-02-01', end_date: null },
        ],
      },
    })
    renderPage()
    expect(screen.getByRole('status')).toHaveTextContent('Loading your projects')
    const table = await screen.findByRole('table')
    const apollo = within(table).getByText('Apollo').closest('tr') as HTMLElement
    for (const t of ['60%', 'Developer', '2026-01-01', '2026-12-31']) expect(within(apollo).getByText(t)).toBeInTheDocument()
    const zeus = within(table).getByText('Zeus').closest('tr') as HTMLElement
    expect(within(zeus).getByText('Ongoing')).toBeInTheDocument()
  })

  it('shows an empty state', async () => {
    mock.onGet('/api/v1/me/projects').reply(200, body([]))
    renderPage()
    expect(await screen.findByText('No active projects')).toBeInTheDocument()
  })

  it('shows an error with retry', async () => {
    mock.onGet('/api/v1/me/projects').reply(500, { success: false, error: { message: 'boom' } })
    renderPage()
    expect(await screen.findByRole('alert')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })

  it('sends the matching scope per dropdown option and shows per-scope empty states', async () => {
    mock.onGet('/api/v1/me/projects', { params: { scope: 'active' } }).reply(200, body([]))
    mock.onGet('/api/v1/me/projects', { params: { scope: 'upcoming' } }).reply(200, body([]))
    mock.onGet('/api/v1/me/projects', { params: { scope: 'past' } }).reply(200, body([
      { id: 'p', project_name: 'Old', project_code: 'OLD', project_status: 'completed', allocation_percent: 50, start_date: '2025-01-01', end_date: '2025-06-01' },
    ]))
    renderPage()
    expect(await screen.findByText('No active projects')).toBeInTheDocument()
    pick('Upcoming')
    expect(await screen.findByText('No upcoming projects')).toBeInTheDocument()
    pick('Past')
    expect(await screen.findByText('Old')).toBeInTheDocument()
    const scopes = mock.history.get.map((r) => r.params?.scope)
    expect(scopes).toEqual(expect.arrayContaining(['active', 'upcoming', 'past']))
  })

  it('toggles to the calendar view, fetching scope=all, and draws bars', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 5, 15))
    mock.onGet('/api/v1/me/projects', { params: { scope: 'active' } }).reply(200, body([]))
    mock.onGet('/api/v1/me/projects', { params: { scope: 'all' } }).reply(200, body([
      { id: 'a1', project_name: 'Apollo', project_code: 'APL', project_status: 'active', allocation_percent: 60, start_date: '2026-06-03', end_date: '2026-06-10' },
      { id: 'a2', project_name: 'Zeus', project_code: 'ZEU', project_status: 'active', allocation_percent: 40, start_date: '2026-05-20', end_date: null },
      { id: 'a3', project_name: 'Gone', project_code: 'GON', project_status: 'completed', allocation_percent: 20, start_date: '2026-06-01', end_date: '2026-06-02' },
      { id: 'a4', project_name: 'Soon', project_code: 'SON', project_status: 'active', allocation_percent: 10, start_date: '2026-06-20', end_date: '2026-06-25' },
    ]))
    renderPage()
    await screen.findByText('No active projects')
    fireEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    expect(screen.getByTestId('loc')).toHaveTextContent('view=calendar')
    expect(screen.getByRole('button', { name: 'Calendar' })).toHaveAttribute('aria-pressed', 'true')
    expect(await screen.findByText('June 2026')).toBeInTheDocument()
    await waitFor(() => expect(screen.getAllByTestId('allocation-bar').length).toBeGreaterThan(0))
    const bars = screen.getAllByTestId('allocation-bar')
    const apollo = bars.filter((b) => b.textContent?.includes('Apollo'))
    expect(apollo[0]).toHaveTextContent('60%')
    expect(apollo[0]).toHaveAttribute('data-open-ended', 'false')
    const zeus = bars.filter((b) => b.textContent?.includes('Zeus'))
    expect(zeus.length).toBeGreaterThan(1) // spans every week of the month
    expect(zeus[0]).toHaveAttribute('data-open-ended', 'true')
    expect(apollo[0]).toHaveAttribute('data-state', 'past')
    expect(apollo[0]).toHaveAttribute('aria-label', expect.stringContaining('Past'))
    for (const z of zeus) expect(z).toHaveAttribute('data-state', 'active')
    expect(bars.find((b) => b.textContent?.includes('Gone'))).toHaveAttribute('data-state', 'past')
    const soon = bars.find((b) => b.textContent?.includes('Soon'))
    expect(soon).toHaveAttribute('data-state', 'upcoming')
    expect(soon).toHaveAttribute('aria-label', expect.stringContaining('Upcoming'))
    expect(within(screen.getByTestId('calendar-legend')).getAllByRole('listitem').map((l) => l.textContent)).toEqual(['Past', 'Active', 'Upcoming'])
    expect(mock.history.get.some((r) => r.params?.scope === 'all')).toBe(true)
    fireEvent.click(screen.getByRole('button', { name: 'Next month' }))
    expect(screen.getByText('July 2026')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Today' }))
    expect(screen.getByText('June 2026')).toBeInTheDocument()
  })

  const CAL = [
    { id: 'a1', project_name: 'Apollo', project_code: 'APL', project_status: 'active', allocation_percent: 60, role_on_project: 'Lead', start_date: '2026-06-03', end_date: '2026-06-10' },
    { id: 'a2', project_name: 'Zeus', project_code: 'ZEU', project_status: 'active', allocation_percent: 40, start_date: '2026-05-20', end_date: null },
  ]
  const calBody = (over: boolean) => ({ success: true, data: { allocations: CAL, total_percent: 100, over_allocated: over } })

  it('shows the total chip, and an amber warning with text when over-allocated', async () => {
    mock.onGet('/api/v1/me/projects').replyOnce(200, { success: true, data: { allocations: [], total_percent: 80, over_allocated: false } })
    const { unmount } = renderPage()
    const chip = await screen.findByTestId('total-allocation')
    expect(chip).toHaveTextContent('Total today: 80%')
    expect(chip).toHaveAttribute('data-over-allocated', 'false')
    expect(chip).not.toHaveTextContent('Over-allocated')
    unmount()
    mock.onGet('/api/v1/me/projects').reply(200, { success: true, data: { allocations: [], total_percent: 130, over_allocated: true } })
    renderPage()
    await waitFor(() => expect(screen.getByTestId('total-allocation')).toHaveTextContent('Total today: 130%'))
    expect(screen.getByTestId('total-allocation')).toHaveTextContent('Over-allocated')
    expect(screen.getByTestId('total-allocation').className).toContain('amber')
  })

  it('opens a details dialog from a bar and closes on Esc and outside click', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 5, 15))
    mock.onGet('/api/v1/me/projects').reply(200, calBody(false))
    renderPage('/?view=calendar')
    const bar = (await screen.findAllByTestId('allocation-bar')).find((b) => b.textContent?.includes('Apollo')) as HTMLElement
    expect(screen.queryByRole('dialog')).toBeNull()
    fireEvent.click(bar)
    const dlg = screen.getByRole('dialog')
    for (const t of ['Apollo', 'APL', '60%', 'Lead', '2026-06-03', '2026-06-10', 'Past']) expect(within(dlg).getByText(t)).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(bar).toHaveFocus()
    const zeus = screen.getAllByTestId('allocation-bar').find((b) => b.textContent?.includes('Zeus')) as HTMLElement
    fireEvent.click(zeus)
    expect(within(screen.getByRole('dialog')).getByText('Ongoing')).toBeInTheDocument()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('marks today and shades weekends', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 5, 15))
    mock.onGet('/api/v1/me/projects').reply(200, calBody(false))
    const { container } = renderPage('/?view=calendar')
    await screen.findAllByTestId('allocation-bar')
    const today = screen.getByTestId('today-cell')
    expect(today).toHaveTextContent('15')
    expect(today).toHaveAttribute('data-today', 'true')
    expect(container.querySelectorAll('[data-weekend="true"]').length).toBeGreaterThanOrEqual(8)
  })

  it('renders a week-grouped agenda instead of the grid at narrow widths', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 5, 15))
    mockNarrow(true)
    mock.onGet('/api/v1/me/projects').reply(200, calBody(false))
    renderPage('/?view=calendar')
    const items = await screen.findAllByTestId('agenda-item')
    expect(screen.queryByRole('grid')).toBeNull()
    expect(screen.getAllByRole('heading', { level: 3 }).length).toBeGreaterThan(1)
    const apollo = items.find((i) => i.textContent?.includes('Apollo')) as HTMLElement
    expect(apollo).toHaveAttribute('data-state', 'past')
    expect(apollo).toHaveTextContent('Past')
    expect(items.find((i) => i.textContent?.includes('Zeus'))).toHaveTextContent('Ongoing')
  })

  it('keeps the grid at desktop widths', async () => {
    mockNarrow(false)
    mock.onGet('/api/v1/me/projects').reply(200, calBody(false))
    renderPage('/?view=calendar')
    expect(await screen.findByRole('grid')).toBeInTheDocument()
    expect(screen.queryByTestId('agenda-item')).toBeNull()
  })

  it('persists tab and view in the URL; invalid values fall back to active/list', async () => {
    mock.onGet('/api/v1/me/projects').reply(200, body([]))
    renderPage('/?tab=past')
    expect(await screen.findByText('No past projects')).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Allocation status' })).toHaveValue('past')
    pick('Upcoming')
    expect(await screen.findByText('No upcoming projects')).toBeInTheDocument()
    expect(screen.getByTestId('loc')).toHaveTextContent('tab=upcoming')
    fireEvent.click(screen.getByRole('button', { name: 'Calendar' }))
    expect(screen.getByTestId('loc')).toHaveTextContent('tab=upcoming')
    expect(screen.getByTestId('loc')).toHaveTextContent('view=calendar')
  })

  it('falls back to the active tab and list view for invalid params', async () => {
    mock.onGet('/api/v1/me/projects').reply(200, body([]))
    renderPage('/?tab=bogus&view=nope')
    expect(await screen.findByText('No active projects')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'List' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('offers Active, Upcoming and Past options without counts', async () => {
    const counts = { active: 1, upcoming: 6, past: 2 }
    mock.onGet('/api/v1/me/projects').reply(200, { success: true, data: { allocations: [], total_percent: 0, over_allocated: false, counts } })
    renderPage()
    const select = screen.getByRole('combobox', { name: 'Allocation status' })
    expect(await screen.findByText('No active projects')).toBeInTheDocument()
    expect(within(select).getAllByRole('option').map((o) => o.textContent)).toEqual(['Active', 'Upcoming', 'Past'])
    expect(select).toHaveValue('active')
    pick('Upcoming')
    expect(await screen.findByText('No upcoming projects')).toBeInTheDocument()
    expect(mock.history.get.some((r) => r.params?.scope === 'upcoming')).toBe(true)
  })

  const MANY = Array.from({ length: 25 }, (_, i) => ({
    id: `m${i + 1}`,
    project_name: `Proj ${String(i + 1).padStart(2, '0')}`,
    project_code: `C${i + 1}`,
    project_status: 'active',
    allocation_percent: 10,
    role_on_project: i === 3 ? 'Architect' : 'Dev',
    start_date: '2026-01-01',
    end_date: null,
  }))
  const manyReply = () => mock.onGet('/api/v1/me/projects').reply(200, body(MANY))
  const rowCount = () => within(screen.getByRole('table')).getAllByRole('row').length - 1

  it('filters rows by name, code and role, case-insensitively and trimmed, persisting q', async () => {
    manyReply()
    renderPage()
    await screen.findByRole('table')
    const input = screen.getByRole('searchbox', { name: 'Search projects' })
    fireEvent.change(input, { target: { value: '  PROJ 07 ' } })
    expect(rowCount()).toBe(1)
    expect(screen.getByTestId('loc')).toHaveTextContent('q=')
    fireEvent.change(input, { target: { value: 'c12' } })
    expect(within(screen.getByRole('table')).getByText('Proj 12')).toBeInTheDocument()
    fireEvent.change(input, { target: { value: 'architect' } })
    expect(within(screen.getByRole('table')).getByText('Proj 04')).toBeInTheDocument()
    expect(rowCount()).toBe(1)
  })

  it('shows a no-match state with a clear-search action', async () => {
    manyReply()
    renderPage('/?q=zzz')
    expect(await screen.findByText('No projects match your search')).toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: 'Pagination' })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Clear search' }))
    expect(await screen.findByRole('table')).toBeInTheDocument()
    expect(screen.getByRole('searchbox', { name: 'Search projects' })).toHaveValue('')
    expect(screen.getByTestId('loc')).not.toHaveTextContent('q=')
  })

  it('paginates 10 per page with prev/next and a showing summary', async () => {
    manyReply()
    renderPage()
    await screen.findByRole('table')
    expect(rowCount()).toBe(10)
    expect(screen.getByText('Showing 1–10 of 25')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(screen.getByText('Showing 11–20 of 25')).toBeInTheDocument()
    expect(screen.getByText('Proj 11')).toBeInTheDocument()
    expect(screen.getByTestId('loc')).toHaveTextContent('page=2')
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(rowCount()).toBe(5)
    expect(screen.getByText('Showing 21–25 of 25')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }))
    expect(screen.getByText('Showing 11–20 of 25')).toBeInTheDocument()
  })

  it('hides pagination when rows fit on one page', async () => {
    mock.onGet('/api/v1/me/projects').reply(200, body(MANY.slice(0, 10)))
    renderPage()
    await screen.findByRole('table')
    expect(screen.queryByRole('navigation', { name: 'Pagination' })).toBeNull()
  })

  it('restores search and page from the URL and clamps an out-of-range page', async () => {
    manyReply()
    const { unmount } = renderPage('/?page=3')
    await screen.findByRole('table')
    expect(screen.getByText('Showing 21–25 of 25')).toBeInTheDocument()
    unmount()
    renderPage('/?page=99')
    await screen.findByRole('table')
    expect(screen.getByText('Showing 21–25 of 25')).toBeInTheDocument()
  })

  it('restores q from the URL', async () => {
    manyReply()
    renderPage('/?q=C1')
    await screen.findByRole('table')
    expect(screen.getByRole('searchbox', { name: 'Search projects' })).toHaveValue('C1')
    expect(screen.getByText('Showing 1–10 of 11')).toBeInTheDocument()
  })

  it('resets to page 1 when the search or scope changes', async () => {
    manyReply()
    renderPage('/?page=2')
    await screen.findByRole('table')
    expect(screen.getByText('Showing 11–20 of 25')).toBeInTheDocument()
    fireEvent.change(screen.getByRole('searchbox', { name: 'Search projects' }), { target: { value: 'proj' } })
    expect(screen.getByText('Showing 1–10 of 25')).toBeInTheDocument()
    expect(screen.getByTestId('loc')).not.toHaveTextContent('page=')
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(screen.getByTestId('loc')).toHaveTextContent('page=2')
    pick('Past')
    expect(screen.getByTestId('loc')).not.toHaveTextContent('page=')
    expect(await screen.findByText('Showing 1–10 of 25')).toBeInTheDocument()
  })

  it('calendar view ignores search and page and still uses scope=all', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 5, 15))
    mock.onGet('/api/v1/me/projects').reply(200, calBody(false))
    renderPage('/?view=calendar&q=zzz&page=5')
    expect((await screen.findAllByTestId('allocation-bar')).length).toBeGreaterThan(0)
    expect(screen.queryByRole('searchbox')).toBeNull()
    expect(screen.queryByRole('combobox')).toBeNull()
    expect(screen.queryByRole('navigation', { name: 'Pagination' })).toBeNull()
    expect(mock.history.get.every((r) => r.params?.scope === 'all')).toBe(true)
  })
})
