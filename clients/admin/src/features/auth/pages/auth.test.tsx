import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, getSession, setSession } from '@employee360/api-client'
import App from '../../../App'
import { FORGOT_SUCCESS_MESSAGE } from './ForgotPasswordPage'
import { useAuthStore } from '../../../stores/authStore'

let mock: MockAdapter

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

const loginOk = (role: string) => ({
  success: true,
  data: {
    access_token: 'a',
    refresh_token: 'r',
    user: { id: 'u1', email: 'a@x.com', tenant_id: 't1', roles: [{ name: role }] },
  },
})

beforeEach(() => {
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/health').reply(200, { success: true, data: { status: 'ok', app: 'e', database: 'ok' } })
  clearSession()
  useAuthStore.getState().clear()
  localStorage.clear()
})
afterEach(() => mock.restore())

describe('login', () => {
  it('redirects unauthenticated users to login', () => {
    renderAt('/')
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
  })

  it('signs in and lands in the admin shell', async () => {
    mock.onPost('/api/v1/auth/login').reply(200, loginOk('admin'))
    renderAt('/login')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Password'), 'pw')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('button', { name: 'Sign out' })).toBeInTheDocument()
    expect(getSession()?.accessToken).toBe('a')
    expect(useAuthStore.getState().user?.roles).toEqual(['admin'])
  })

  it('shows one generic error for bad credentials', async () => {
    mock.onPost('/api/v1/auth/login').reply(401, { error: { message: 'user not found' } })
    renderAt('/login')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Password'), 'pw')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid email or password.')
    expect(screen.queryByText(/not found/i)).toBeNull()
  })

  it('rejects non-admin accounts with the same generic error', async () => {
    mock.onPost('/api/v1/auth/login').reply(200, loginOk('employee'))
    mock.onPost('/api/v1/auth/logout').reply(200, {})
    renderAt('/login')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Password'), 'pw')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid email or password.')
    expect(useAuthStore.getState().user).toBeNull()
  })
})

describe('login errors', () => {
  it.each([500, 429])('shows a distinct message for status %i', async (status) => {
    mock.onPost('/api/v1/auth/login').reply(status, {})
    renderAt('/login')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Password'), 'pw')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Unable to sign in. Please try again.')
  })
  it('shows a distinct message on network failure', async () => {
    mock.onPost('/api/v1/auth/login').networkError()
    renderAt('/login')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Password'), 'pw')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Unable to sign in. Please try again.')
  })
  it('keeps the credentials message for 403', async () => {
    mock.onPost('/api/v1/auth/login').reply(403, {})
    renderAt('/login')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Password'), 'pw')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid email or password.')
  })
})

describe('reload persistence', () => {
  const session = { accessToken: 'a', refreshToken: 'r', tenantId: 't1' }
  const adminUser = { id: 'u1', email: 'a@x.com', roles: ['admin'] }

  it('renders the admin shell from a persisted session and user', async () => {
    setSession(session)
    localStorage.setItem(
      'employee360.admin.auth',
      JSON.stringify({ state: { user: adminUser }, version: 0 }),
    )
    await useAuthStore.persist.rehydrate()
    renderAt('/')
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument()
  })
  it('redirects to login with a token but no user', () => {
    setSession(session)
    renderAt('/')
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
  })
  it('redirects to login with a user but no token', () => {
    useAuthStore.setState({ user: adminUser })
    renderAt('/')
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
  })
  it('redirects to login when the persisted user lacks an admin role', () => {
    setSession(session)
    useAuthStore.setState({ user: { ...adminUser, roles: ['employee'] } })
    renderAt('/')
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
  })
})

describe('forgot password', () => {
  it.each([200, 404, 500])('shows the same generic message for API status %i', async (status) => {
    mock.onPost('/api/v1/auth/forgot-password').reply(status, {})
    renderAt('/forgot-password')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.click(screen.getByRole('button', { name: 'Send reset link' }))
    expect(await screen.findByRole('status')).toHaveTextContent(FORGOT_SUCCESS_MESSAGE)
  })
  it('shows the same message on network failure', async () => {
    mock.onPost('/api/v1/auth/forgot-password').networkError()
    renderAt('/forgot-password')
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.click(screen.getByRole('button', { name: 'Send reset link' }))
    expect(await screen.findByRole('status')).toHaveTextContent(FORGOT_SUCCESS_MESSAGE)
  })
})

describe('reset password', () => {
  it('validates match and posts token + new password', async () => {
    mock.onPost('/api/v1/auth/reset-password').reply(200, {})
    renderAt('/reset-password?token=tok')
    await userEvent.type(screen.getByLabelText('New password'), 'password123')
    await userEvent.type(screen.getByLabelText('Confirm new password'), 'different1')
    await userEvent.click(screen.getByRole('button', { name: 'Reset password' }))
    expect(await screen.findByText('Passwords do not match')).toBeInTheDocument()
    await userEvent.clear(screen.getByLabelText('Confirm new password'))
    await userEvent.type(screen.getByLabelText('Confirm new password'), 'password123')
    await userEvent.click(screen.getByRole('button', { name: 'Reset password' }))
    expect(await screen.findByText(/has been reset/i)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ token: 'tok', new_password: 'password123' })
  })
  it('shows invalid-link state without a token', () => {
    renderAt('/reset-password')
    expect(screen.getByRole('alert')).toHaveTextContent(/invalid/i)
  })
  it('clears any active session on success', async () => {
    setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' })
    useAuthStore.setState({ user: { id: 'u1', email: 'a@x.com', roles: ['admin'] } })
    mock.onPost('/api/v1/auth/reset-password').reply(200, {})
    renderAt('/reset-password?token=tok')
    await userEvent.type(screen.getByLabelText('New password'), 'password123')
    await userEvent.type(screen.getByLabelText('Confirm new password'), 'password123')
    await userEvent.click(screen.getByRole('button', { name: 'Reset password' }))
    expect(await screen.findByText(/has been reset/i)).toBeInTheDocument()
    expect(getSession()).toBeNull()
    expect(useAuthStore.getState().user).toBeNull()
  })
})

describe('accept invitation', () => {
  it('shows invalid-link state without a token', () => {
    renderAt('/accept-invitation')
    expect(screen.getByRole('alert')).toHaveTextContent(/this invitation link is invalid/i)
    expect(screen.getByRole('link', { name: /return to sign in/i })).toBeInTheDocument()
  })

  it('validates token on load before showing password fields and completes acceptance', async () => {
    mock.onGet('/api/v1/invitations/validate', { params: { token: 'inv-token-123' } }).reply(200, {
      success: true,
      data: { valid: true, email: 'new@acme.com', role: 'admin' },
    })
    mock.onPost('/api/v1/invitations/accept').reply(200, { success: true, data: { message: 'invitation accepted' } })
    renderAt('/accept-invitation?token=inv-token-123')

    // Password fields appear once token is validated
    expect(await screen.findByRole('heading', { name: /set your password/i })).toBeInTheDocument()

    // Password length validation
    await userEvent.type(screen.getByLabelText(/^password$/i), 'short')
    await userEvent.type(screen.getByLabelText(/^confirm password$/i), 'short')
    await userEvent.click(screen.getByRole('button', { name: /set password & accept/i }))
    expect(await screen.findByText('Password must be at least 8 characters')).toBeInTheDocument()

    // Password mismatch validation
    await userEvent.clear(screen.getByLabelText(/^password$/i))
    await userEvent.type(screen.getByLabelText(/^password$/i), 'securePassword123')
    await userEvent.clear(screen.getByLabelText(/^confirm password$/i))
    await userEvent.type(screen.getByLabelText(/^confirm password$/i), 'mismatchPassword')
    await userEvent.click(screen.getByRole('button', { name: /set password & accept/i }))
    expect(await screen.findByText('Passwords do not match')).toBeInTheDocument()

    // Valid submission
    await userEvent.clear(screen.getByLabelText(/^confirm password$/i))
    await userEvent.type(screen.getByLabelText(/^confirm password$/i), 'securePassword123')
    await userEvent.click(screen.getByRole('button', { name: /set password & accept/i }))

    expect(await screen.findByRole('heading', { name: /invitation accepted/i })).toBeInTheDocument()
    expect(screen.getByText(/your invitation has been accepted/i)).toBeInTheDocument()
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ token: 'inv-token-123', password: 'securePassword123' })

    // Clicking sign in lands on login page with acceptance alert
    await userEvent.click(screen.getByRole('link', { name: /sign in to admin/i }))
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument()
    expect(screen.getByText(/your invitation was accepted\. you can now sign in\./i)).toBeInTheDocument()

    // AC: the invitee can then actually log in with the role-appropriate login
    mock.onPost('/api/v1/auth/login').reply(200, loginOk('admin'))
    await userEvent.type(screen.getByLabelText('Email'), 'a@x.com')
    await userEvent.type(screen.getByLabelText('Password'), 'securePassword123')
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('button', { name: 'Sign out' })).toBeInTheDocument()
    expect(useAuthStore.getState().user?.roles).toEqual(['admin'])
    expect(JSON.parse(mock.history.post.find((r) => r.url === '/api/v1/auth/login')!.data)).toMatchObject({
      email: 'a@x.com',
      password: 'securePassword123',
    })
  })

  it.each([
    ['INVALID_TOKEN', 400, /this invitation link is invalid/i],
    ['INVITATION_EXPIRED', 410, /ask your administrator to resend/i],
    ['INVITATION_REVOKED', 403, /invitation has been revoked.*contact your administrator/i],
    ['INVITATION_ACCEPTED', 409, /already been accepted/i],
  ])('renders a distinct state for validate error %s', async (code, status, text) => {
    mock.onGet('/api/v1/invitations/validate').reply(status, { success: false, error: { code, message: 'server text' } })
    renderAt('/accept-invitation?token=bad-token')

    expect(await screen.findByRole('alert')).toHaveTextContent(text)
    expect(screen.queryByLabelText(/^password$/i)).toBeNull()
    expect(screen.queryByRole('button', { name: /set password & accept/i })).toBeNull()
    // Only the accepted state offers a primary Sign in button; expired offers no sign-in CTA.
    const signInButton = screen.queryByRole('link', { name: /^sign in$/i })
    if (code === 'INVITATION_ACCEPTED') expect(signInButton).toBeInTheDocument()
    else expect(signInButton).toBeNull()
  })

  it('uses distinct headings per error code', async () => {
    const headings: string[] = []
    for (const [code, status] of [
      ['INVITATION_EXPIRED', 410],
      ['INVITATION_REVOKED', 403],
      ['INVITATION_ACCEPTED', 409],
      ['INVALID_TOKEN', 400],
    ] as const) {
      mock.reset()
      mock.onGet('/api/v1/invitations/validate').reply(status, { error: { code, message: 'm' } })
      const { unmount } = renderAt(`/accept-invitation?token=${code}`)
      await screen.findByRole('alert')
      headings.push(screen.getByRole('heading', { level: 1 }).textContent ?? '')
      unmount()
    }
    expect(new Set(headings).size).toBe(4)
  })

  it('does not classify by message text when the code is unknown', async () => {
    mock.onGet('/api/v1/invitations/validate').reply(500, { error: { code: 'INTERNAL', message: 'token expired conflict' } })
    renderAt('/accept-invitation?token=t')
    expect(await screen.findByRole('alert')).toHaveTextContent('token expired conflict')
    expect(screen.queryByText(/ask your administrator to resend/i)).toBeNull()
  })

  it('shows the invitee email on the set-password form', async () => {
    mock.onGet('/api/v1/invitations/validate').reply(200, {
      success: true,
      data: { valid: true, email: 'new@acme.com', role: 'admin' },
    })
    renderAt('/accept-invitation?token=ok')
    expect(await screen.findByText('new@acme.com')).toBeInTheDocument()
    expect(screen.getByText(/setting a password for/i)).toBeInTheDocument()
  })

  it('keeps unknown roles on the current app', async () => {
    mock.onGet('/api/v1/invitations/validate').reply(200, { success: true, data: { email: 'x@y.com', role: 'manager' } })
    renderAt('/accept-invitation?token=ok')
    expect(await screen.findByRole('heading', { name: /set your password/i })).toBeInTheDocument()
  })

  describe('role-based host switching', () => {
    const replace = vi.fn()
    beforeEach(() => {
      replace.mockClear()
      vi.stubGlobal('location', { ...window.location, replace })
    })
    afterEach(() => {
      vi.unstubAllGlobals()
      vi.unstubAllEnvs()
    })

    it('redirects a employee invitation to the employee app preserving the token', async () => {
      vi.stubEnv('VITE_EMPLOYEE_APP_URL', 'http://employee.test/')
      mock.onGet('/api/v1/invitations/validate').reply(200, { success: true, data: { email: 'x@y.com', role: 'employee' } })
      renderAt('/accept-invitation?token=tok%201')
      await waitFor(() => expect(replace).toHaveBeenCalledWith('http://employee.test/accept-invitation?token=tok%201'))
      expect(screen.queryByLabelText(/^password$/i)).toBeNull()
    })

    it('shows a clear message instead of redirecting when the target URL is not configured', async () => {
      vi.stubEnv('VITE_EMPLOYEE_APP_URL', '')
      mock.onGet('/api/v1/invitations/validate').reply(200, { success: true, data: { email: 'x@y.com', role: 'employee' } })
      renderAt('/accept-invitation?token=tok')
      expect(await screen.findByText(/this invitation is for the Employee Portal/i)).toBeInTheDocument()
      expect(replace).not.toHaveBeenCalled()
      expect(screen.queryByLabelText(/^password$/i)).toBeNull()
    })

    it('does not redirect when the role belongs to this app', async () => {
      vi.stubEnv('VITE_EMPLOYEE_APP_URL', 'http://employee.test')
      mock.onGet('/api/v1/invitations/validate').reply(200, { success: true, data: { email: 'x@y.com', role: 'admin' } })
      renderAt('/accept-invitation?token=tok')
      expect(await screen.findByRole('heading', { name: /set your password/i })).toBeInTheDocument()
      expect(replace).not.toHaveBeenCalled()
    })
  })

  it('handles INVITATION_ACCEPTED returned by accept (race after validate)', async () => {
    mock.onGet('/api/v1/invitations/validate').reply(200, { success: true, data: { email: 'new@acme.com', role: 'admin' } })
    mock.onPost('/api/v1/invitations/accept').reply(409, { error: { code: 'INVITATION_ACCEPTED', message: 'x' } })
    renderAt('/accept-invitation?token=race')
    await screen.findByRole('heading', { name: /set your password/i })
    await userEvent.type(screen.getByLabelText(/^password$/i), 'securePassword123')
    await userEvent.type(screen.getByLabelText(/^confirm password$/i), 'securePassword123')
    await userEvent.click(screen.getByRole('button', { name: /set password & accept/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/already been accepted/i)
    expect(screen.getByRole('link', { name: /^sign in$/i })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent(/invited to the employee portal, sign in there instead/i)
  })

  it('keeps the server message for password-validation errors on accept', async () => {
    mock.onGet('/api/v1/invitations/validate').reply(200, { success: true, data: { email: 'new@acme.com', role: 'admin' } })
    mock.onPost('/api/v1/invitations/accept').reply(400, { error: { code: 'BAD_REQUEST', message: 'password is too common' } })
    renderAt('/accept-invitation?token=pw')
    await screen.findByRole('heading', { name: /set your password/i })
    await userEvent.type(screen.getByLabelText(/^password$/i), 'securePassword123')
    await userEvent.type(screen.getByLabelText(/^confirm password$/i), 'securePassword123')
    await userEvent.click(screen.getByRole('button', { name: /set password & accept/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('password is too common')
    expect(screen.getByRole('heading', { name: /set your password/i })).toBeInTheDocument()
  })

  it('clears active session state on successful acceptance', async () => {
    mock.onGet('/api/v1/invitations/validate', { params: { token: 'valid-tok' } }).reply(200, {
      success: true,
      data: { valid: true, email: 'new@acme.com', role: 'admin' },
    })
    mock.onPost('/api/v1/invitations/accept').reply(200, { success: true, data: { message: 'invitation accepted' } })
    setSession({ accessToken: 'old-access', refreshToken: 'old-refresh', tenantId: 'old-tenant' })
    useAuthStore.setState({ user: { id: 'old-u', email: 'old@x.com', roles: ['admin'] } })

    renderAt('/accept-invitation?token=valid-tok')
    expect(await screen.findByRole('heading', { name: /set your password/i })).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText(/^password$/i), 'securePassword123')
    await userEvent.type(screen.getByLabelText(/^confirm password$/i), 'securePassword123')
    await userEvent.click(screen.getByRole('button', { name: /set password & accept/i }))

    expect(await screen.findByRole('heading', { name: /invitation accepted/i })).toBeInTheDocument()
    expect(getSession()).toBeNull()
    expect(useAuthStore.getState().user).toBeNull()
  })
})
