import { beforeEach, describe, expect, it } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import App from '../../../App'
import { useAuthStore } from '../../../stores/authStore'

let mock: MockAdapter
let settings: Record<string, unknown>

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
  settings = {
    employee_code_prefix: 'EMP', employee_code_padding: 4, employee_code_start_number: 1,
    employee_code_next_number: 1, employee_codes_issued: false, employee_code_preview: 'EMP0001', weekly_off_days: [0, 6],
  }
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/organization/settings').reply(() => [200, { success: true, data: settings }])
  clearSession()
  useAuthStore.getState().clear()
})

describe('Organization > Employee codes tab', () => {
  it('renders the tab for admins without fetching the super_admin-only tenant', async () => {
    signIn()
    renderAt('/settings/organization/employee-codes')
    expect(await screen.findByLabelText(/Code prefix/)).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Employee codes' })).toHaveAttribute('aria-selected', 'true')
    expect(mock.history.get.some((r) => r.url === '/api/v1/tenant')).toBe(false)
  })

  it('redirects the old /settings/employee-codes path to the tab', async () => {
    signIn()
    renderAt('/settings/employee-codes')
    expect(await screen.findByLabelText(/Code prefix/)).toBeInTheDocument()
  })

  it('updates the live preview as the user types and uppercases the prefix', async () => {
    signIn()
    renderAt('/settings/organization/employee-codes')
    const prefix = await screen.findByLabelText(/Code prefix/)
    await userEvent.clear(prefix)
    await userEvent.type(prefix, 'skryfon')
    expect(prefix).toHaveValue('SKRYFON')
    const start = screen.getByLabelText(/Starting number/)
    await userEvent.clear(start)
    await userEvent.type(start, '40001')
    expect(screen.getByTestId('code-preview')).toHaveTextContent('SKRYFON40001')
    expect(screen.getByText(/applies to new employees only/)).toBeInTheDocument()
  })

  it('shows client-side validation errors and does not submit', async () => {
    signIn()
    renderAt('/settings/organization/employee-codes')
    const prefix = await screen.findByLabelText(/Code prefix/)
    await userEvent.clear(prefix)
    await userEvent.type(prefix, 'EMP1')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText(/must not end in a digit/)).toBeInTheDocument()
    expect(mock.history.put).toHaveLength(0)
  })

  it('saves the format and sends numbers', async () => {
    signIn()
    mock.onPut('/api/v1/organization/settings').reply((cfg) => {
      settings = { ...settings, ...JSON.parse(cfg.data), employee_code_preview: 'ACME0001' }
      return [200, { success: true, data: settings }]
    })
    renderAt('/settings/organization/employee-codes')
    const prefix = await screen.findByLabelText(/Code prefix/)
    await userEvent.clear(prefix)
    await userEvent.type(prefix, 'acme')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.put).toHaveLength(1))
    expect(JSON.parse(mock.history.put[0].data)).toEqual({
      employee_code_prefix: 'ACME', employee_code_padding: 4, employee_code_start_number: 1,
    })
  })

  it('constrains the starting number once codes were issued and shows the counter', async () => {
    settings = { ...settings, employee_codes_issued: true, employee_code_next_number: 12, employee_code_start_number: 1 }
    signIn()
    renderAt('/settings/organization/employee-codes')
    expect(await screen.findByTestId('counter')).toHaveTextContent('12')
    const start = screen.getByLabelText(/Starting number/)
    await userEvent.clear(start)
    await userEvent.type(start, '5')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText(/at least 12/)).toBeInTheDocument()
    expect(mock.history.put).toHaveLength(0)
  })

  it('surfaces server validation errors inline', async () => {
    signIn()
    mock.onPut('/api/v1/organization/settings').reply(400, { success: false, error: { message: 'employee_code_padding: prefix length plus padding must not exceed 20 characters' } })
    renderAt('/settings/organization/employee-codes')
    const pad = await screen.findByLabelText(/Digit padding/)
    await userEvent.clear(pad)
    await userEvent.type(pad, '5')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    const err = await screen.findByText('prefix length plus padding must not exceed 20 characters')
    expect(err).toBeInTheDocument()
    expect(err.textContent).not.toMatch(/employee_code_padding:/)
    expect(screen.getByLabelText(/Digit padding/)).toHaveAccessibleDescription(/must not exceed 20/)
  })
})

describe('Organization > Welcome email tab', () => {
  it('is a separate tab and the Employee codes form no longer has the message', async () => {
    signIn()
    renderAt('/settings/organization/employee-codes')
    expect(await screen.findByLabelText(/Code prefix/)).toBeInTheDocument()
    expect(screen.queryByLabelText(/Welcome message/)).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('tab', { name: 'Welcome email' }))
    expect(await screen.findByLabelText(/Welcome message/)).toBeInTheDocument()
    expect(screen.queryByLabelText(/Code prefix/)).not.toBeInTheDocument()
  })

  it('shows the saved welcome message with a counter and sends edits', async () => {
    settings = { ...settings, welcome_message: 'Hello team' }
    signIn()
    mock.onPut('/api/v1/organization/settings').reply((cfg) => [200, { success: true, data: { ...settings, ...JSON.parse(cfg.data) } }])
    renderAt('/settings/organization/welcome-email')
    const box = await screen.findByLabelText(/Welcome message/)
    expect(box).toHaveValue('Hello team')
    expect(screen.getByText(/10\/2000 characters/)).toBeInTheDocument()
    expect(screen.getByText(/Leave blank to use the\s+default message/)).toBeInTheDocument()
    await userEvent.type(box, '!')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.put).toHaveLength(1))
    expect(JSON.parse(mock.history.put[0].data)).toEqual({
      employee_code_prefix: 'EMP', employee_code_padding: 4, employee_code_start_number: 1,
      welcome_message: 'Hello team!',
    })
  })

  it('treats a null welcome message as blank and sends an empty string to clear', async () => {
    settings = { ...settings, welcome_message: null }
    signIn()
    mock.onPut('/api/v1/organization/settings').reply(200, { success: true, data: settings })
    renderAt('/settings/organization/welcome-email')
    expect(await screen.findByLabelText(/Welcome message/)).toHaveValue('')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.put).toHaveLength(1))
    expect(JSON.parse(mock.history.put[0].data).welcome_message).toBe('')
  })

  it('rejects an over-long welcome message client-side and maps the server error to the field', async () => {
    signIn()
    renderAt('/settings/organization/welcome-email')
    const box = await screen.findByLabelText(/Welcome message/)
    await userEvent.click(box)
    await userEvent.paste('a'.repeat(2001))
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText(/at most 2000 characters/)).toBeInTheDocument()
    expect(mock.history.put).toHaveLength(0)

    mock.onPut('/api/v1/organization/settings').reply(400, { success: false, error: { message: 'welcome_message: must not exceed 2000 characters' } })
    await userEvent.clear(box)
    await userEvent.type(box, 'ok')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    const err = await screen.findByText('must not exceed 2000 characters')
    expect(box).toHaveAccessibleDescription(err.textContent ?? '')
  })
})


describe('Organization > Working days tab', () => {
  const open = async () => {
    signIn()
    renderAt('/settings/organization/working-days')
    await screen.findByRole('button', { name: 'Sunday' })
  }

  it('defaults to Sat/Sun pressed and saves the new selection with the current code format', async () => {
    mock.onPut('/api/v1/organization/settings').reply(200, { success: true, data: settings })
    await open()
    expect(screen.getByRole('button', { name: 'Sunday' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Monday' })).toHaveAttribute('aria-pressed', 'false')
    await userEvent.click(screen.getByRole('button', { name: 'Friday' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(mock.history.put).toHaveLength(1))
    expect(JSON.parse(mock.history.put[0].data)).toEqual({
      employee_code_prefix: 'EMP', employee_code_padding: 4, employee_code_start_number: 1, weekly_off_days: [0, 5, 6],
    })
  })

  it('blocks saving with no days selected', async () => {
    await open()
    await userEvent.click(screen.getByRole('button', { name: 'Sunday' }))
    await userEvent.click(screen.getByRole('button', { name: 'Saturday' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText(/at least one weekly off day/)).toBeInTheDocument()
    expect(mock.history.put).toHaveLength(0)
  })

  it('blocks saving with all seven days selected', async () => {
    await open()
    for (const d of ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday']) {
      await userEvent.click(screen.getByRole('button', { name: d }))
    }
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText(/at most six/)).toBeInTheDocument()
    expect(mock.history.put).toHaveLength(0)
  })

  it('shows the server validation error', async () => {
    mock.onPut('/api/v1/organization/settings').reply(400, { success: false, error: { message: 'weekly_off_days: invalid' } })
    await open()
    await userEvent.click(screen.getByRole('button', { name: 'Friday' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/weekly_off_days/)
  })
})
