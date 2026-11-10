import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import { AppToaster, useToast } from '@employee360/ui'
import DailyStatusPage from './pages/DailyStatusPage'
import { isoOf, lastSevenDays } from './dates'

let mock: MockAdapter
const [today, yesterday] = lastSevenDays()

function renderPage() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <DailyStatusPage />
        <AppToaster />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const entry = { id: 'e1', log_date: today, project_id: 'p1', project_name: 'Apollo', project_code: 'APL', hours: 6, notes: 'Built it' }

beforeEach(() => {
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/me/projects').reply(200, {
    success: true,
    data: { allocations: [{ id: 'a1', project_id: 'p1', project_name: 'Apollo' }], total_percent: 100, over_allocated: false },
  })
  mock.onGet('/api/v1/me/status-logs').reply((cfg) => {
    const { from, to } = cfg.params as { from: string; to: string }
    return [200, { success: true, data: [{ date: today, total_hours: 6, entries: [entry] }].filter((g) => g.date >= from && g.date <= to) }]
  })
})
afterEach(() => {
  mock.restore()
  useToast().dismiss() // sonner's store is module-global; don't leak toasts into the next test
})

describe('Daily status page', () => {
  it('shows 7 days with totals, flags missed days, and lists the selected day entries', async () => {
    renderPage()
    const strip = await screen.findByRole('group', { name: 'Last 7 days' })
    expect(within(strip).getAllByRole('button')).toHaveLength(7)
    expect(within(strip).getAllByText('None')).toHaveLength(6)
    expect(within(strip).getByText('6h')).toBeInTheDocument()
    expect(screen.getByText('Apollo (APL)')).toBeInTheDocument()
    expect(screen.getByText('Built it')).toBeInTheDocument()
    expect(screen.getByText(/Total:/)).toHaveTextContent('6h of 24h')
  })

  it('keeps the selected day entries when only the week request fails, and the strip offers Retry', async () => {
    mock.onGet('/api/v1/me/status-logs').reply((cfg) => {
      const { from, to } = cfg.params as { from: string; to: string }
      if (from !== to) return [500, { success: false, error: { message: 'week down' } }]
      return [200, { success: true, data: [{ date: today, total_hours: 6, entries: [entry] }].filter((g) => g.date >= from && g.date <= to) }]
    })
    renderPage()
    expect(await screen.findByText('Apollo (APL)')).toBeInTheDocument()
    expect(screen.getByText(/Total:/)).toHaveTextContent('6h of 24h')
    expect(await screen.findByText('week down')).toBeInTheDocument()
    expect(screen.queryByRole('group', { name: 'Last 7 days' })).not.toBeInTheDocument()
    mock.onGet('/api/v1/me/status-logs').reply((cfg) => {
      const { from, to } = cfg.params as { from: string; to: string }
      return [200, { success: true, data: [{ date: today, total_hours: 6, entries: [entry] }].filter((g) => g.date >= from && g.date <= to) }]
    })
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByRole('group', { name: 'Last 7 days' })).toBeInTheDocument()
  })

  it('requests the selected day from the backend (from = to) when a date is selected', async () => {
    renderPage()
    await screen.findByText('Apollo (APL)')
    const dayCalls = () => mock.history.get.filter((r) => r.url === '/api/v1/me/status-logs' && r.params.from === r.params.to)
    expect(dayCalls().map((r) => r.params.from)).toContain(today)
    const strip = screen.getByRole('group', { name: 'Last 7 days' })
    await userEvent.click(within(strip).getAllByRole('button')[5])
    await waitFor(() => expect(dayCalls().map((r) => r.params.from)).toContain(yesterday))
    expect(await screen.findByText(/Total:/)).toHaveTextContent('0h of 24h')
    expect(screen.queryByText('Apollo (APL)')).not.toBeInTheDocument()
  })

  it('creates an entry for the selected day', async () => {
    mock.onPost('/api/v1/me/status-logs').reply(201, { success: true, data: { ...entry, id: 'e2' } })
    renderPage()
    await screen.findByRole('group', { name: 'Last 7 days' })
    await userEvent.click(within(screen.getByRole('group', { name: 'Last 7 days' })).getByRole('button', { name: new RegExp(`${Number(yesterday.slice(8))},`) }))
    await userEvent.click(screen.getByRole('button', { name: 'Add entry' }))
    expect(screen.getByRole('dialog', { name: 'New entry' })).toHaveAttribute('aria-modal', 'true')
    await userEvent.selectOptions(screen.getByLabelText(/^Project or category/), 'category:meeting')
    await userEvent.type(screen.getByLabelText(/^Hours\s*\*$/), '2.5')
    await userEvent.click(screen.getByRole('button', { name: 'Save entry' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ log_date: yesterday, activity_category: 'meeting', hours: 2.5 })
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Entry added.')).toBeInTheDocument()
  })

  it('blocks entries that push the total over 24h client-side', async () => {
    renderPage()
    await screen.findByText('Apollo (APL)')
    await userEvent.click(screen.getByRole('button', { name: 'Add entry' }))
    await userEvent.selectOptions(screen.getByLabelText(/^Project or category/), 'category:other')
    await userEvent.type(screen.getByLabelText(/^Hours\s*\*$/), '20')
    await userEvent.click(screen.getByRole('button', { name: 'Save entry' }))
    expect(await screen.findByText(/cannot exceed 24 hours/)).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })

  it('surfaces a log_window_closed server error inline', async () => {
    mock.onPost('/api/v1/me/status-logs').reply(422, { success: false, error: { code: 'log_window_closed', message: 'closed' } })
    renderPage()
    await screen.findByText('Apollo (APL)')
    await userEvent.click(screen.getByRole('button', { name: 'Add entry' }))
    await userEvent.selectOptions(screen.getByLabelText(/^Project or category/), 'project:p1')
    await userEvent.type(screen.getByLabelText(/^Hours\s*\*$/), '1')
    await userEvent.click(screen.getByRole('button', { name: 'Save entry' }))
    expect(await within(screen.getByRole('dialog')).findByText(/outside the 7-day logging window/)).toBeInTheDocument()
  })

  it('closes the modal on Escape, Cancel and backdrop click, restoring focus to the trigger', async () => {
    renderPage()
    await screen.findByText('Apollo (APL)')
    const add = screen.getByRole('button', { name: 'Add entry' })
    await userEvent.click(add)
    expect(screen.getByRole('dialog', { name: 'New entry' })).toBeInTheDocument()
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(add).toHaveFocus()
    await userEvent.click(add)
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await userEvent.click(add)
    await userEvent.pointer({ keys: '[MouseLeft]', target: screen.getByRole('dialog').parentElement as HTMLElement })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('edits and deletes an entry', async () => {
    mock.onPatch('/api/v1/me/status-logs/e1').reply(200, { success: true, data: entry })
    mock.onDelete('/api/v1/me/status-logs/e1').reply(200, { success: true })
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Edit Apollo (APL)' }))
    expect(screen.getByRole('dialog', { name: 'Edit entry' })).toBeInTheDocument()
    const hours = screen.getByLabelText(/^Hours\s*\*$/)
    expect(hours).toHaveValue('6')
    await userEvent.clear(hours)
    await userEvent.type(hours, '7')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toMatchObject({ hours: 7, project_id: 'p1' })
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Entry updated.')).toBeInTheDocument()
  })

  it('marks required fields with an asterisk and aria-required', async () => {
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Add entry' }))
    const dialog = screen.getByRole('dialog', { name: 'New entry' })
    for (const name of [/^Project or category/, /^Hours\s*\*$/]) {
      const input = within(dialog).getByLabelText(name)
      expect(input).toHaveAttribute('aria-required', 'true')
      expect(input.closest('div')?.querySelector('label span[aria-hidden="true"]')).toHaveTextContent('*')
    }
    expect(within(dialog).getByLabelText('Notes')).not.toHaveAttribute('aria-required')
  })

  describe('delete confirmation', () => {
    beforeEach(() => {
      mock.onDelete('/api/v1/me/status-logs/e1').reply(200, { success: true })
    })

    it('does nothing until confirmed, and Cancel keeps the entry', async () => {
      renderPage()
      const trigger = await screen.findByRole('button', { name: 'Delete Apollo (APL)' })
      await userEvent.click(trigger)
      const dialog = screen.getByRole('dialog', { name: 'Delete entry?' })
      expect(within(dialog).getByText(/Apollo \(APL\)/)).toBeInTheDocument()
      expect(mock.history.delete).toHaveLength(0)
      await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(mock.history.delete).toHaveLength(0)
      expect(trigger).toHaveFocus()
      expect(screen.getByRole('button', { name: 'Edit Apollo (APL)' })).toBeInTheDocument()
    })

    it('closes on Escape without deleting', async () => {
      renderPage()
      await userEvent.click(await screen.findByRole('button', { name: 'Delete Apollo (APL)' }))
      await userEvent.keyboard('{Escape}')
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(mock.history.delete).toHaveLength(0)
    })

    it('deletes on confirm', async () => {
      renderPage()
      await userEvent.click(await screen.findByRole('button', { name: 'Delete Apollo (APL)' }))
      await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }))
      await waitFor(() => expect(mock.history.delete).toHaveLength(1))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(await screen.findByText('Entry deleted.')).toBeInTheDocument()
    })

    it('shows delete errors in the page-level alert', async () => {
      mock.onDelete('/api/v1/me/status-logs/e1').reply(500, { success: false, error: { message: 'boom' } })
      renderPage()
      await userEvent.click(await screen.findByRole('button', { name: 'Delete Apollo (APL)' }))
      await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }))
      expect(await screen.findByRole('alert')).toBeInTheDocument()
    })
  })

  describe('notes view more', () => {
    const longNotes = 'Long note about the work done today. '.repeat(6).trim()
    const mockNotes = (notes: string) =>
      mock.onGet('/api/v1/me/status-logs').reply(200, {
        success: true,
        data: [{ date: today, total_hours: 6, entries: [{ ...entry, notes }] }],
      })

    it('shows no toggle for short notes', async () => {
      renderPage()
      await screen.findByText('Built it')
      expect(screen.queryByRole('button', { name: 'View more' })).not.toBeInTheDocument()
    })

    it('expands and collapses long notes', async () => {
      mockNotes(longNotes)
      renderPage()
      const toggle = await screen.findByRole('button', { name: 'View more' })
      expect(toggle).toHaveAttribute('aria-expanded', 'false')
      expect(screen.getByText(longNotes)).toHaveClass('line-clamp-2')
      await userEvent.click(toggle)
      const less = screen.getByRole('button', { name: 'View less' })
      expect(less).toHaveAttribute('aria-expanded', 'true')
      expect(screen.getByText(longNotes)).not.toHaveClass('line-clamp-2')
      await userEvent.click(less)
      expect(screen.getByRole('button', { name: 'View more' })).toHaveAttribute('aria-expanded', 'false')
      expect(screen.getByText(longNotes)).toHaveClass('line-clamp-2')
    })

    it('renders a dash for empty notes', async () => {
      mockNotes('')
      renderPage()
      await screen.findByText('Apollo (APL)')
      expect(screen.queryByRole('button', { name: 'View more' })).not.toBeInTheDocument()
      expect(screen.getByText('-')).toBeInTheDocument()
    })
  })

  describe('notes lists', () => {
    const openForm = async () => {
      renderPage()
      await userEvent.click(await screen.findByRole('button', { name: 'Add entry' }))
      const dialog = screen.getByRole('dialog', { name: 'New entry' })
      return { dialog, notes: within(dialog).getByLabelText('Notes') as HTMLTextAreaElement }
    }

    it('applies and toggles a bullet list via the toolbar', async () => {
      const { dialog, notes } = await openForm()
      await userEvent.type(notes, 'one{Enter}two')
      notes.setSelectionRange(0, notes.value.length)
      await userEvent.click(within(dialog).getByRole('button', { name: 'Bullet list' }))
      expect(notes.value).toBe('- one\n- two')
      notes.setSelectionRange(0, notes.value.length)
      await userEvent.click(within(dialog).getByRole('button', { name: 'Numbered list' }))
      expect(notes.value).toBe('1. one\n2. two')
      expect(notes).toHaveFocus()
    })

    it('continues the list on Enter and ends it on an empty item', async () => {
      const { dialog, notes } = await openForm()
      await userEvent.click(within(dialog).getByRole('button', { name: 'Bullet list' }))
      await userEvent.type(notes, 'a{Enter}b{Enter}')
      expect(notes.value).toBe('- a\n- b\n- ')
      await userEvent.keyboard('{Enter}')
      expect(notes.value).toBe('- a\n- b\n')
    })
  })

  describe('notes list rendering', () => {
    it('renders bullet and numbered lines as real lists', async () => {
      mock.onGet('/api/v1/me/status-logs').reply(200, {
        success: true,
        data: [{ date: today, total_hours: 6, entries: [{ ...entry, notes: 'Intro\n- alpha\n- beta\n1. first\n2. second' }] }],
      })
      renderPage()
      const items = await screen.findAllByRole('listitem')
      expect(items.map((li) => li.textContent)).toEqual(['alpha', 'beta', 'first', 'second'])
      expect(items[0].parentElement?.tagName).toBe('UL')
      expect(items[2].parentElement?.tagName).toBe('OL')
      expect(screen.getByText('Intro')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'View more' })).toBeInTheDocument()
    })
  })

  describe('project list scoped to the selected date', () => {
    const dayBtn = (d: string) => within(screen.getByRole('group', { name: 'Last 7 days' })).getByRole('button', { name: new RegExp(`${Number(d.slice(8))},`) })
    const mockProjects = (allocations: object[]) =>
      mock.onGet('/api/v1/me/projects').reply(200, { success: true, data: { allocations, total_percent: 0, over_allocated: false } })

    it('lists only projects whose allocation covers the date, and hints when none', async () => {
      mockProjects([
        { id: 'a1', project_id: 'p1', project_name: 'Apollo', start_date: '2000-01-01' },
        { id: 'a2', project_id: 'p2', project_name: 'Ended', start_date: '2000-01-01', end_date: yesterday },
        { id: 'a3', project_id: 'p3', project_name: 'Later', start_date: '2999-01-01' },
      ])
      renderPage()
      await screen.findByText('Apollo (APL)')
      await userEvent.click(screen.getByRole('button', { name: 'Add entry' }))
      let select = screen.getByLabelText(/^Project or category/)
      expect(within(select).getByRole('option', { name: 'Apollo' })).toBeInTheDocument()
      expect(within(select).queryByRole('option', { name: 'Later' })).not.toBeInTheDocument()
      expect(within(select).queryByRole('option', { name: 'Ended' })).not.toBeInTheDocument()
      await userEvent.keyboard('{Escape}')
      // yesterday: Ended is active, Later is not
      await userEvent.click(dayBtn(yesterday))
      await userEvent.click(screen.getByRole('button', { name: 'Add entry' }))
      select = screen.getByLabelText(/^Project or category/)
      expect(within(select).getByRole('option', { name: 'Ended' })).toBeInTheDocument()
      expect(within(select).queryByRole('option', { name: 'Later' })).not.toBeInTheDocument()
    })

    it('shows a hint and keeps categories when no project is active', async () => {
      mockProjects([{ id: 'a3', project_id: 'p3', project_name: 'Later', start_date: '2999-01-01' }])
      renderPage()
      await screen.findByText('Apollo (APL)')
      await userEvent.click(screen.getByRole('button', { name: 'Add entry' }))
      expect(screen.getByText(/No projects are allocated to you on this date/)).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'Meeting' })).toBeInTheDocument()
    })

    it('keeps an edited entry project selectable when it is no longer allocated that day', async () => {
      mockProjects([])
      renderPage()
      await userEvent.click(await screen.findByRole('button', { name: 'Edit Apollo (APL)' }))
      const select = screen.getByLabelText(/^Project or category/)
      expect(select).toHaveValue('project:p1')
      expect(within(select).getByRole('option', { name: 'Apollo' })).toBeInTheDocument()
    })
  })

  describe('unsaved changes warning', () => {
    const openDirty = async () => {
      renderPage()
      await screen.findByText('Apollo (APL)')
      const add = screen.getByRole('button', { name: 'Add entry' })
      await userEvent.click(add)
      await userEvent.type(screen.getByLabelText(/^Hours\s*\*$/), '3')
      return add
    }

    it('closes immediately when the form is pristine', async () => {
      renderPage()
      await screen.findByText('Apollo (APL)')
      await userEvent.click(screen.getByRole('button', { name: 'Add entry' }))
      await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it.each([
      ['Escape', async () => userEvent.keyboard('{Escape}')],
      ['Cancel', async () => userEvent.click(screen.getByRole('button', { name: 'Cancel' }))],
      ['backdrop', async () => userEvent.pointer({ keys: '[MouseLeft]', target: screen.getByRole('dialog', { name: 'New entry' }).parentElement as HTMLElement })],
    ])('asks to discard on %s and Keep editing preserves the form', async (_n, close) => {
      await openDirty()
      await close()
      const confirm = screen.getByRole('dialog', { name: 'Discard changes?' })
      await userEvent.click(within(confirm).getByRole('button', { name: 'Keep editing' }))
      expect(screen.queryByRole('dialog', { name: 'Discard changes?' })).not.toBeInTheDocument()
      expect(screen.getByRole('dialog', { name: 'New entry' })).toBeInTheDocument()
      expect(screen.getByLabelText(/^Hours\s*\*$/)).toHaveValue('3')
    })

    it('Escape on the confirm only dismisses the confirm; Discard closes everything and restores focus', async () => {
      const add = await openDirty()
      await userEvent.keyboard('{Escape}')
      await userEvent.keyboard('{Escape}')
      expect(screen.getByRole('dialog', { name: 'New entry' })).toBeInTheDocument()
      expect(screen.queryByRole('dialog', { name: 'Discard changes?' })).not.toBeInTheDocument()
      await userEvent.keyboard('{Escape}')
      await userEvent.click(within(screen.getByRole('dialog', { name: 'Discard changes?' })).getByRole('button', { name: 'Discard' }))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(add).toHaveFocus()
      expect(document.body.style.overflow).toBe('')
    })

    it('saving closes without prompting', async () => {
      mock.onPost('/api/v1/me/status-logs').reply(201, { success: true, data: { ...entry, id: 'e2' } })
      await openDirty()
      await userEvent.selectOptions(screen.getByLabelText(/^Project or category/), 'category:meeting')
      await userEvent.click(screen.getByRole('button', { name: 'Save entry' }))
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    })
  })

  describe('keyboard shortcuts', () => {
    const selectedDay = () => screen.getAllByRole('button', { pressed: true })[0]

    it('moves the selected day with arrows, clamped to the window', async () => {
      renderPage()
      await screen.findByText('Apollo (APL)')
      expect(selectedDay()).toHaveAccessibleName(new RegExp(`${Number(today.slice(8))},`))
      await userEvent.keyboard('{ArrowRight}')
      expect(selectedDay()).toHaveAccessibleName(new RegExp(`${Number(today.slice(8))},`))
      await userEvent.keyboard('{ArrowLeft}')
      expect(selectedDay()).toHaveAccessibleName(new RegExp(`${Number(yesterday.slice(8))},`))
      for (let i = 0; i < 10; i++) await userEvent.keyboard('{ArrowLeft}')
      const oldest = lastSevenDays()[6]
      expect(selectedDay()).toHaveAccessibleName(new RegExp(`${Number(oldest.slice(8))},`))
    })

    it('N opens the new entry modal; ignored while typing or when a modal is open', async () => {
      renderPage()
      await screen.findByText('Apollo (APL)')
      await userEvent.keyboard('n')
      expect(screen.getByRole('dialog', { name: 'New entry' })).toBeInTheDocument()
      await userEvent.type(screen.getByLabelText('Notes'), 'n{ArrowLeft}')
      expect(screen.getByLabelText('Notes')).toHaveValue('n')
      expect(screen.getAllByRole('dialog')).toHaveLength(1)
    })

    it('does not trigger from a text field', async () => {
      renderPage()
      await screen.findByText('Apollo (APL)')
      const input = document.createElement('input')
      document.body.appendChild(input)
      input.focus()
      await userEvent.keyboard('n')
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      input.remove()
    })

    it('Ctrl+Enter inside the modal submits the form', async () => {
      mock.onPost('/api/v1/me/status-logs').reply(201, { success: true, data: { ...entry, id: 'e2' } })
      renderPage()
      await screen.findByText('Apollo (APL)')
      await userEvent.keyboard('n')
      await userEvent.selectOptions(screen.getByLabelText(/^Project or category/), 'category:meeting')
      await userEvent.type(screen.getByLabelText(/^Hours\s*\*$/), '1')
      await userEvent.type(screen.getByLabelText('Notes'), 'x{Control>}{Enter}{/Control}')
      await waitFor(() => expect(mock.history.post).toHaveLength(1))
      expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ activity_category: 'meeting', hours: 1, notes: 'x' })
    })

    it('advertises the shortcuts', async () => {
      renderPage()
      expect(await screen.findByText(/Shortcuts:/)).toHaveTextContent('N new entry')
      expect(await screen.findByRole('button', { name: 'Add entry' })).toHaveAttribute('aria-keyshortcuts', 'N')
    })
  })

  describe('weekly total', () => {
    it('shows the sum of the 7 days', async () => {
      const [d0, d1] = lastSevenDays()
      mock.onGet('/api/v1/me/status-logs').reply(200, {
        success: true,
        data: [
          { date: d0, total_hours: 6, entries: [entry] },
          { date: d1, total_hours: 8.5, entries: [] },
        ],
      })
      renderPage()
      const strip = await screen.findByRole('group', { name: 'Last 7 days' })
      expect(screen.getByText(/^Week:/)).toHaveTextContent('Week: 14.5h')
      expect(within(strip).getAllByText('None')).toHaveLength(5)
    })
  })
})

describe('calendar view', () => {
  const old = (() => {
    const d = new Date()
    d.setDate(d.getDate() - 9)
    return isoOf(d)
  })()
  const oldEntry = { ...entry, id: 'o1', log_date: old, hours: 3 }
  const mockMonth = () =>
    mock.onGet('/api/v1/me/status-logs').reply((cfg) => {
      const from = cfg.params?.from as string
      const data = [
        { date: today, total_hours: 6, entries: [entry] },
        { date: old, total_hours: 3, entries: [oldEntry] },
      ].filter((g) => g.date >= from && g.date <= cfg.params?.to)
      return [200, { success: true, data }]
    })
  const openCalendar = async () => {
    renderPage()
    await screen.findByText('Apollo (APL)')
    await userEvent.click(screen.getByRole('tab', { name: 'Calendar' }))
    // Arrow keys on a focused tab switch tabs (tablist pattern); move focus away so they navigate days.
    ;(document.activeElement as HTMLElement).blur()
    return screen.findByRole('group', { name: 'Daily status calendar' })
  }
  const cell = (date: string) => document.querySelector(`[data-date="${date}"]`) as HTMLElement
  const selectedCell = () => document.querySelector('.holiday-day--selected') as HTMLElement
  const clickDay = (date: string) => userEvent.click(cell(date).querySelector('.fc-daygrid-day-frame') as HTMLElement)

  it('defaults to the week view and toggles with tabs', async () => {
    renderPage()
    await screen.findByText('Apollo (APL)')
    expect(screen.getByRole('tab', { name: 'Week' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('group', { name: 'Last 7 days' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('tab', { name: 'Calendar' }))
    expect(screen.getByRole('tab', { name: 'Calendar' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByRole('group', { name: 'Last 7 days' })).not.toBeInTheDocument()
  })

  it('fetches the visible month with one ranged request and labels the days', async () => {
    mockMonth()
    await openCalendar()
    const calls = mock.history.get.filter((r) => r.url === '/api/v1/me/status-logs')
    expect(calls[calls.length - 1].params.from).toBe(`${today.slice(0, 8)}01`)
    expect(await within(cell(today)).findByText('6h')).toBeInTheDocument()
    expect(within(cell(today)).getByText('Edit')).toBeInTheDocument()
    expect(cell(today)).toHaveAttribute('aria-current', 'date')
    expect(within(cell(yesterday)).getByText('None')).toBeInTheDocument()
  })

  it('marks future days disabled and blocks next month at the current month', async () => {
    mockMonth()
    await openCalendar()
    expect(screen.getByRole('button', { name: 'Next month' })).toBeDisabled()
    const future = [...document.querySelectorAll<HTMLElement>('.fc-daygrid-day[data-date]')].filter((c) => c.dataset.date! > today)
    future.forEach((c) => expect(c).toHaveAttribute('aria-disabled', 'true'))
    if (future.length) {
      await userEvent.click(future[0].querySelector('.fc-daygrid-day-frame') as HTMLElement)
      expect(selectedCell().dataset.date).toBe(today)
    }
  })

  it('shows read-only notice and hides actions for an older day, and shortcuts do nothing', async () => {
    mockMonth()
    await openCalendar()
    if (old.slice(0, 7) !== today.slice(0, 7)) return // window straddles months; covered by month navigation test
    await clickDay(old)
    expect(within(cell(old)).getByText('3h')).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('Read-only — older than 7 days')
    expect(screen.getByText(/Total:/)).toHaveTextContent('3h of 24h')
    expect(screen.queryByRole('button', { name: 'Add entry' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Edit / })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Delete / })).not.toBeInTheDocument()
    await userEvent.keyboard('n')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shares the selected day between views and clamps back into the window', async () => {
    mockMonth()
    await openCalendar()
    await clickDay(yesterday)
    await userEvent.click(screen.getByRole('tab', { name: 'Week' }))
    expect(screen.getAllByRole('button', { pressed: true })[0]).toHaveAccessibleName(new RegExp(`^\\w+ ${Number(yesterday.slice(8))},`))
    expect(screen.queryByRole('note')).not.toBeInTheDocument()
  })

  it('arrow keys move across the window edge in the calendar but never into the future', async () => {
    mockMonth()
    await openCalendar()
    await userEvent.keyboard('{ArrowRight}')
    expect(selectedCell().dataset.date).toBe(today)
    for (let i = 0; i < 8; i++) await userEvent.keyboard('{ArrowLeft}')
    // 8 days back: read-only (or previous month)
    expect(await screen.findByRole('note')).toHaveTextContent('Read-only')
    await userEvent.keyboard('{ArrowDown}')
    expect(selectedCell()).toBeDefined()
  })

  it('navigates to the previous month and refetches that range', async () => {
    mockMonth()
    await openCalendar()
    await userEvent.click(screen.getByRole('button', { name: 'Previous month' }))
    await waitFor(() => {
      const last = mock.history.get.filter((r) => r.url === '/api/v1/me/status-logs').pop()
      expect(last?.params.from.slice(0, 7)).not.toBe(today.slice(0, 7))
    })
    expect(screen.getByRole('button', { name: 'Next month' })).toBeEnabled()
    expect(await screen.findByRole('note')).toHaveTextContent('Read-only')
  })

  describe('day statuses', () => {
    const weekDates = lastSevenDays().slice().reverse()
    const mockStatuses = (overrides: Record<string, object>) =>
      mock.onGet('/api/v1/me/status-logs/day-statuses').reply(200, {
        success: true,
        data: weekDates.map((date) => ({ date, day_status: date === today ? 'logged' : 'missing', total_hours: date === today ? 6 : 0, ...overrides[date] })),
      })

    it('labels holiday, weekly off, leave and half-day-missing instead of None, with notes for the selected day', async () => {
      mockStatuses({
        [weekDates[0]]: { day_status: 'holiday', holiday_name: 'Diwali' },
        [weekDates[1]]: { day_status: 'weekly_off' },
        [weekDates[2]]: { day_status: 'on_leave', leave_type_name: 'Annual' },
        [weekDates[3]]: { day_status: 'missing', half_day: true, leave_type_name: 'Annual' },
      })
      renderPage()
      const strip = await screen.findByRole('group', { name: 'Last 7 days' })
      await within(strip).findByText('Holiday')
      expect(within(strip).getByText('Holiday')).toHaveAttribute('title', 'Public holiday: Diwali')
      expect(within(strip).getByText('Weekly off')).toBeInTheDocument()
      expect(within(strip).getByText('On leave')).toHaveAttribute('title', 'On leave: Annual')
      expect(within(strip).getByText('Missing').className).toContain('text-amber-700')
      expect(within(strip).getByText('Half-day leave')).toBeInTheDocument()
      expect(within(strip).getAllByText('None')).toHaveLength(2)
      expect(within(strip).getByRole('button', { name: /Public holiday, Diwali/ })).toBeInTheDocument()
      await userEvent.click(within(strip).getByRole('button', { name: /Public holiday, Diwali/ }))
      expect(await screen.findByRole('note')).toHaveTextContent('Public holiday: Diwali — no log needed.')
      await userEvent.click(within(strip).getByRole('button', { name: /half-day leave/ }))
      expect(await screen.findByRole('note')).toHaveTextContent('Half-day leave — please log your other half.')
    })
  })
})
