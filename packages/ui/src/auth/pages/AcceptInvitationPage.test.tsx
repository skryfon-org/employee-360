import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, getSession, setSession } from '@employee360/api-client'
import { AcceptInvitationPage, type AcceptInvitationPageProps } from './AcceptInvitationPage'

const VALIDATE = '/api/v1/invitations/validate'
const ACCEPT = '/api/v1/invitations/accept'
const PASSWORD = 'securePassword123'

const baseProps: AcceptInvitationPageProps = {
  currentApp: 'admin',
  signInLabel: 'Sign in to Admin',
  otherPortalName: 'Employee Portal',
  otherPortalNameInAcceptedHint: 'Employee Portal (accepted hint)',
}

let mock: MockAdapter

function renderPage(path: string, props: Partial<AcceptInvitationPageProps> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <AcceptInvitationPage {...baseProps} {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  mock = new MockAdapter(apiClient)
})
afterEach(() => {
  mock.restore()
  vi.unstubAllGlobals()
})

describe('AcceptInvitationPage (shared)', () => {
  it('shows the invalid-link state when no token is present', () => {
    renderPage('/accept-invitation')
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Invalid invitation link')
    expect(screen.getByRole('alert')).toHaveTextContent(
      'This invitation link is invalid. Please check the link in your invitation email and try again.',
    )
    expect(screen.getByRole('link', { name: 'Return to sign in' })).toHaveAttribute('href', '/login')
    expect(mock.history.get).toHaveLength(0)
  })

  it('accepts the invitation with token + password, calls onAccepted and shows the sign-in state', async () => {
    mock.onGet(VALIDATE, { params: { token: 'tok-1' } }).reply(200, {
      success: true,
      data: { valid: true, email: 'new@acme.com', role: 'admin' },
    })
    mock.onPost(ACCEPT).reply(200, { success: true, data: { message: 'invitation accepted' } })
    setSession({ accessToken: 'old-access', refreshToken: 'old-refresh', tenantId: 'old-tenant' })
    const onAccepted = vi.fn()

    renderPage('/accept-invitation?token=tok-1', { onAccepted })

    expect(await screen.findByRole('heading', { level: 1, name: 'Set your password' })).toBeInTheDocument()
    expect(screen.getByText('new@acme.com')).toBeInTheDocument()

    // Mismatched confirmation is rejected client-side and nothing is posted.
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: PASSWORD } })
    fireEvent.change(screen.getByLabelText('Confirm password'), { target: { value: 'differentPassword1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Set password & accept' }))
    expect(await screen.findByText('Passwords do not match')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
    expect(onAccepted).not.toHaveBeenCalled()

    fireEvent.change(screen.getByLabelText('Confirm password'), { target: { value: PASSWORD } })
    fireEvent.click(screen.getByRole('button', { name: 'Set password & accept' }))

    expect(await screen.findByRole('heading', { level: 1, name: 'Invitation accepted' })).toBeInTheDocument()
    expect(
      screen.getByText(
        'Your invitation has been accepted and your password has been set. You can now sign in to your account.',
      ),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sign in to Admin' })).toHaveAttribute('href', '/login')
    expect(mock.history.post).toHaveLength(1)
    expect(JSON.parse(mock.history.post[0].data)).toEqual({ token: 'tok-1', password: PASSWORD })
    expect(onAccepted).toHaveBeenCalledTimes(1)
    expect(getSession()).toBeNull()
  })

  it.each([
    {
      code: 'INVALID_TOKEN',
      status: 400,
      title: 'Invalid invitation link',
      message: 'This invitation link is invalid. Please check the link in your invitation email and try again.',
      signIn: false,
    },
    {
      code: 'INVITATION_EXPIRED',
      status: 410,
      title: 'Invitation expired',
      message: 'This invitation has expired. Ask your administrator to resend the invitation.',
      signIn: false,
    },
    {
      code: 'INVITATION_REVOKED',
      status: 403,
      title: 'Invitation revoked',
      message: 'This invitation has been revoked. Please contact your administrator.',
      signIn: false,
    },
    {
      code: 'INVITATION_ACCEPTED',
      status: 409,
      title: 'Invitation already accepted',
      message:
        'This invitation has already been accepted. You can sign in with your password. If you were invited to the Employee Portal (accepted hint), sign in there instead.',
      signIn: true,
    },
  ])('renders the distinct $title state for validate error code $code', async (c) => {
    // The server message is deliberately misleading: the state must come from the code only.
    mock.onGet(VALIDATE).reply(c.status, { success: false, error: { code: c.code, message: 'token expired revoked' } })
    renderPage('/accept-invitation?token=bad')

    expect(await screen.findByRole('alert')).toHaveTextContent(c.message)
    expect(screen.getByRole('alert').textContent).toBe(c.message)
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(c.title)
    expect(screen.queryByLabelText('Password')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Set password & accept' })).toBeNull()

    if (c.signIn) {
      expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/login')
      expect(screen.queryByRole('link', { name: 'Return to sign in' })).toBeNull()
    } else {
      expect(screen.getByRole('link', { name: 'Return to sign in' })).toHaveAttribute('href', '/login')
      expect(screen.queryByRole('link', { name: 'Sign in' })).toBeNull()
    }
  })

  it('falls back to the server message for an unknown error code', async () => {
    mock.onGet(VALIDATE).reply(500, { error: { code: 'INTERNAL', message: 'token expired conflict' } })
    renderPage('/accept-invitation?token=t')
    expect(await screen.findByRole('alert')).toHaveTextContent('token expired conflict')
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Accept invitation')
  })

  it('shows the ACCEPTED state when accept returns INVITATION_ACCEPTED after validation (race)', async () => {
    mock.onGet(VALIDATE).reply(200, { success: true, data: { email: 'new@acme.com', role: 'admin' } })
    mock.onPost(ACCEPT).reply(409, { error: { code: 'INVITATION_ACCEPTED', message: 'x' } })
    const onAccepted = vi.fn()
    renderPage('/accept-invitation?token=race', { onAccepted })

    await screen.findByText('Set your password')
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: PASSWORD } })
    fireEvent.change(screen.getByLabelText('Confirm password'), { target: { value: PASSWORD } })
    fireEvent.click(screen.getByRole('button', { name: 'Set password & accept' }))

    expect(await screen.findByRole('heading', { level: 1, name: 'Invitation already accepted' })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('If you were invited to the Employee Portal (accepted hint), sign in there instead.')
    expect(screen.getByRole('link', { name: 'Sign in' })).toBeInTheDocument()
    expect(onAccepted).not.toHaveBeenCalled()
  })

  describe('cross-app redirect', () => {
    const replace = vi.fn()
    beforeEach(() => {
      replace.mockClear()
      vi.stubGlobal('location', { ...window.location, replace })
    })

    it('redirects an employee invitation opened in admin to employeeAppUrl, preserving the token', async () => {
      mock.onGet(VALIDATE).reply(200, { success: true, data: { email: 'x@y.com', role: 'employee' } })
      renderPage('/accept-invitation?token=tok%201', {
        adminAppUrl: 'http://admin.test',
        employeeAppUrl: 'http://employee.test/',
      })

      expect(await screen.findByRole('heading', { level: 1, name: 'Redirecting' })).toBeInTheDocument()
      expect(screen.getByText(/This invitation is for the Employee Portal\. Redirecting you now/)).toBeInTheDocument()
      expect(screen.getByRole('link', { name: 'Continue' })).toHaveAttribute(
        'href',
        'http://employee.test/accept-invitation?token=tok%201',
      )
      await waitFor(() => expect(replace).toHaveBeenCalledWith('http://employee.test/accept-invitation?token=tok%201'))
      expect(replace).toHaveBeenCalledTimes(1)
      expect(screen.queryByLabelText('Password')).toBeNull()
    })

    it('redirects an admin invitation opened in the employee app to adminAppUrl using the provided otherPortalName', async () => {
      mock.onGet(VALIDATE).reply(200, { success: true, data: { email: 'x@y.com', role: 'admin' } })
      renderPage('/accept-invitation?token=tok', {
        currentApp: 'employee',
        otherPortalName: 'Admin Portal',
        adminAppUrl: 'http://admin.test',
        employeeAppUrl: 'http://employee.test',
      })

      expect(await screen.findByText(/This invitation is for the Admin Portal\. Redirecting you now/)).toBeInTheDocument()
      await waitFor(() => expect(replace).toHaveBeenCalledWith('http://admin.test/accept-invitation?token=tok'))
    })

    it('shows an explanatory notice and does not redirect when the target URL is not configured', async () => {
      mock.onGet(VALIDATE).reply(200, { success: true, data: { email: 'x@y.com', role: 'employee' } })
      renderPage('/accept-invitation?token=tok', { adminAppUrl: 'http://admin.test', employeeAppUrl: '' })

      expect(await screen.findByRole('heading', { level: 1, name: 'Open the Employee Portal' })).toBeInTheDocument()
      expect(screen.getByRole('status').textContent).toBe(
        'This invitation is for the Employee Portal; open the link from your invitation email in that app (the Employee Portal address is not configured here).',
      )
      expect(replace).not.toHaveBeenCalled()
      expect(screen.queryByLabelText('Password')).toBeNull()
    })

    it('does not redirect when the role belongs to the current portal', async () => {
      mock.onGet(VALIDATE).reply(200, { success: true, data: { email: 'x@y.com', role: 'admin' } })
      renderPage('/accept-invitation?token=tok', { employeeAppUrl: 'http://employee.test' })

      expect(await screen.findByRole('heading', { level: 1, name: 'Set your password' })).toBeInTheDocument()
      expect(replace).not.toHaveBeenCalled()
    })
  })
})
