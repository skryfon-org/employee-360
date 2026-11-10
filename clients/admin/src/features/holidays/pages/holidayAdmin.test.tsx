import { pickRowAction } from '../../../test/rowActions'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import HolidayCategoriesPage from './HolidayCategoriesPage'
import HolidaySettingsPage from './HolidaySettingsPage'

function renderPage(ui: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

let mock: MockAdapter
beforeEach(() => {
  mock = new MockAdapter(apiClient)
})
afterEach(() => mock.restore())

describe('HolidayCategoriesPage', () => {
  const cats = [{ id: 'c1', name: 'Public Holiday', color: '#15803D', is_active: true }]

  it('shows a clear error when deleting an in-use category', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: cats })
    mock.onDelete('/api/v1/holiday-categories/c1').reply(409, {
      success: false,
      error: { code: 'CATEGORY_IN_USE', message: 'holiday category is in use by holidays' },
    })
    renderPage(<HolidayCategoriesPage />)
    await pickRowAction('Actions for Public Holiday', 'Delete')
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/still used by one or more holidays/)
  })

  it('creates a category and validates the colour', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: cats })
    mock.onPost('/api/v1/holiday-categories').reply(201, { success: true, data: { id: 'c2' } })
    renderPage(<HolidayCategoriesPage />)
    await user.click(await screen.findByRole('button', { name: 'Add category' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Name'), 'Offsite')
    await user.clear(within(dialog).getByLabelText('Hex colour'))
    await user.type(within(dialog).getByLabelText('Hex colour'), 'red')
    await user.click(within(dialog).getByRole('button', { name: 'Add category' }))
    expect(await within(dialog).findByText('Enter a colour as #RRGGBB')).toBeInTheDocument()
    await user.clear(within(dialog).getByLabelText('Hex colour'))
    await user.type(within(dialog).getByLabelText('Hex colour'), '#112233')
    await user.click(within(dialog).getByRole('button', { name: 'Add category' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ name: 'Offsite', color: '#112233', is_active: true })
  })
})

describe('CategoryFormModal colour picker', () => {
  it('selects a preset swatch and a custom colour, updating the hex and preview', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: [] })
    mock.onPost('/api/v1/holiday-categories').reply(201, { success: true, data: { id: 'c3' } })
    renderPage(<HolidayCategoriesPage />)
    await user.click(await screen.findByRole('button', { name: 'Add category' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('radio', { name: 'Blue #1D4ED8' }))
    expect(within(dialog).getByLabelText('Hex colour')).toHaveValue('#1D4ED8')
    expect(within(dialog).getByRole('radio', { name: 'Blue #1D4ED8' })).toBeChecked()
    expect(within(dialog).getByTestId('color-preview')).toHaveStyle({ backgroundColor: '#1D4ED8' })
    fireEvent.change(within(dialog).getByLabelText('Custom colour'), { target: { value: '#aa00bb' } })
    expect(within(dialog).getByLabelText('Hex colour')).toHaveValue('#AA00BB')
    expect(within(dialog).getByRole('radio', { name: 'Blue #1D4ED8' })).not.toBeChecked()
    await user.type(within(dialog).getByLabelText('Name'), 'Party')
    await user.click(within(dialog).getByRole('button', { name: 'Add category' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data).color).toBe('#AA00BB')
  })
})

describe('CategoryFormModal public flag', () => {
  it('sends is_public from the checkbox and prefills it when editing', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: [] })
    mock.onPost('/api/v1/holiday-categories').reply(201, { success: true, data: { id: 'c4' } })
    renderPage(<HolidayCategoriesPage />)
    await user.click(await screen.findByRole('button', { name: 'Add category' }))
    const dialog = await screen.findByRole('dialog')
    const box = within(dialog).getByRole('checkbox', { name: /Public holiday/ })
    expect(box).not.toBeChecked()
    await user.click(box)
    await user.type(within(dialog).getByLabelText('Name'), 'National')
    await user.click(within(dialog).getByRole('button', { name: 'Add category' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ name: 'National', is_public: true })
  })

  it('shows a Public indicator and prefills the edit form', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-categories').reply(200, {
      success: true,
      data: [{ id: 'p1', name: 'National', color: '#15803D', is_active: true, is_public: true }],
    })
    renderPage(<HolidayCategoriesPage />)
    expect(await screen.findByText('Public')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /National/ }))
    await user.click(await screen.findByRole('menuitem', { name: /Edit/ }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByRole('checkbox', { name: /Public holiday/ })).toBeChecked()
  })
})

describe('HolidaySettingsPage', () => {
  it('loads and saves settings, omitting a blank default year', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-calendar/settings').reply(200, {
      success: true,
      data: { default_view: 'calendar', default_year: 2026, week_start_day: 0 },
    })
    mock.onPut('/api/v1/holiday-calendar/settings').reply(200, { success: true, data: {} })
    renderPage(<HolidaySettingsPage />)
    const year = await screen.findByLabelText('Default year')
    await waitFor(() => expect(year).toHaveValue('2026'))
    await user.clear(year)
    await user.selectOptions(screen.getByLabelText('Default view'), 'list')
    await user.selectOptions(screen.getByLabelText('Week starts on'), 'Monday')
    await user.click(screen.getByRole('button', { name: 'Save settings' }))
    await waitFor(() => expect(mock.history.put).toHaveLength(1))
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ default_view: 'list', week_start_day: 1 })
  })

  it('rejects an out-of-range year', async () => {
    const user = userEvent.setup()
    mock.onGet('/api/v1/holiday-calendar/settings').reply(200, { success: true, data: { default_view: 'list', week_start_day: 1 } })
    renderPage(<HolidaySettingsPage />)
    const year = await screen.findByLabelText('Default year')
    await user.type(year, '1999')
    await user.click(screen.getByRole('button', { name: 'Save settings' }))
    expect(await screen.findByText(/Enter a year between 2000 and 2100/)).toBeInTheDocument()
    expect(mock.history.put).toHaveLength(0)
  })
})
