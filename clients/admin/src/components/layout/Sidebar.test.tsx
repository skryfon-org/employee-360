import { beforeEach, describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AdminShell } from './AdminShell'
import { useUiStore } from '../../stores/uiStore'
import { useAuthStore } from '../../stores/authStore'

function renderShell(path = '/invitations') {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route element={<AdminShell />}>
            <Route path="/invitations/new" element={<p>new page</p>} />
            <Route path="/invitations" element={<p>invitations page</p>} />
            <Route path="/" element={<p>home</p>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  useUiStore.setState({ sidebarCollapsed: false, sidebarMobileOpen: false })
  useAuthStore.setState({ user: { id: 'u1', email: 'admin@x.com', roles: ['admin'] } })
})

describe('Sidebar', () => {
  it('renders nav links, highlights the active one, and shows the user', () => {
    renderShell()
    const link = screen.getByRole('link', { name: 'Invitations' })
    expect(link).toHaveAttribute('href', '/invitations')
    expect(link).toHaveAttribute('aria-current', 'page')
    const sidebar = screen.getByRole('complementary', { name: 'Sidebar' })
    expect(within(sidebar).getByText('admin@x.com')).toBeInTheDocument()
    expect(within(sidebar).getAllByText('Admin').length).toBeGreaterThan(1)
    expect(within(sidebar).getByRole('button', { name: 'Sign out' })).toBeInTheDocument()
    expect(within(screen.getByRole('banner')).queryByRole('button', { name: 'Sign out' })).toBeNull()
  })

  it('shows the full name instead of the email when available', () => {
    useAuthStore.setState({ user: { id: 'u1', email: 'admin@x.com', firstName: 'Ada', lastName: 'Lovelace', roles: ['admin'] } })
    renderShell()
    const sidebar = screen.getByRole('complementary', { name: 'Sidebar' })
    expect(within(sidebar).getByText('Ada Lovelace')).toBeInTheDocument()
    expect(within(sidebar).queryByText('admin@x.com')).toBeNull()
  })

  it('keeps Invitations highlighted on /invitations/new', () => {
    renderShell('/invitations/new')
    const sidebar = screen.getByRole('complementary', { name: 'Sidebar' })
    expect(within(sidebar).getByRole('link', { name: 'Invitations' })).toHaveAttribute('aria-current', 'page')
  })

  it('does not mark the link active on other routes', () => {
    renderShell('/')
    expect(screen.getByRole('link', { name: 'Invitations' })).not.toHaveAttribute('aria-current')
  })

  it('toggles collapsed state in the ui store', async () => {
    renderShell()
    await userEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }))
    expect(useUiStore.getState().sidebarCollapsed).toBe(true)
    expect(screen.getByRole('complementary', { name: 'Sidebar' })).toHaveAttribute('data-collapsed', 'true')
    await userEvent.click(screen.getByRole('button', { name: 'Expand sidebar' }))
    expect(useUiStore.getState().sidebarCollapsed).toBe(false)
  })

  it('opens and closes the mobile drawer', async () => {
    renderShell()
    await userEvent.click(screen.getByRole('button', { name: 'Open menu' }))
    expect(useUiStore.getState().sidebarMobileOpen).toBe(true)
    await userEvent.click(screen.getByTestId('sidebar-backdrop'))
    expect(useUiStore.getState().sidebarMobileOpen).toBe(false)
    await userEvent.click(screen.getByRole('button', { name: 'Open menu' }))
    await userEvent.click(screen.getByRole('link', { name: 'Invitations' }))
    expect(useUiStore.getState().sidebarMobileOpen).toBe(false)
  })
})

describe('AdminShell user role', () => {
  it('shows the humanized role next to the email', () => {
    useAuthStore.setState({ accessToken: 't', user: { id: 'u', email: 'a@x.com', roles: ['employee', 'super_admin'] } })
    renderShell('/')
    expect(within(screen.getByRole('complementary', { name: 'Sidebar' })).getByText('Super Admin')).toBeInTheDocument()
  })
})

describe('AdminShell breadcrumbs', () => {
  it('renders the trail for a nested route with the last crumb as current page', () => {
    renderShell('/invitations/new')
    const nav = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(nav.querySelector('a[href="/invitations"]')).toHaveTextContent('Invitations')
    expect(nav.querySelector('[aria-current="page"]')).toHaveTextContent('Invite user')
  })
  it('shows only Dashboard on the root', () => {
    renderShell('/')
    const nav = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(nav.querySelectorAll('li')).toHaveLength(1)
    expect(nav.querySelector('[aria-current="page"]')).toHaveTextContent('Dashboard')
  })
})
