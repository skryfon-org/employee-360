import { beforeEach, describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { EmployeeShell } from './EmployeeShell'
import { useUiStore } from '../../stores/uiStore'
import { useAuthStore } from '../../stores/authStore'

function renderShell(path = '/holidays') {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route element={<EmployeeShell />}>
            <Route path="/holidays" element={<p>holidays page</p>} />
            <Route path="/" element={<p>home</p>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const sidebar = () => screen.getByRole('complementary', { name: 'Sidebar' })

beforeEach(() => {
  useUiStore.setState({ sidebarCollapsed: false, sidebarMobileOpen: false })
  useAuthStore.setState({ user: { id: 'u1', email: 'emp@x.com', roles: ['employee'] } })
})

describe('Employee Sidebar', () => {
  it('renders Home and Holidays, highlights the active one, and shows the user', () => {
    renderShell()
    const holidays = within(sidebar()).getByRole('link', { name: 'Holidays' })
    expect(holidays).toHaveAttribute('href', '/holidays')
    expect(holidays).toHaveAttribute('aria-current', 'page')
    expect(within(sidebar()).getByRole('link', { name: 'Home' })).not.toHaveAttribute('aria-current')
    expect(within(sidebar()).getByText('emp@x.com')).toBeInTheDocument()
    expect(within(sidebar()).getByRole('button', { name: 'Sign out' })).toBeInTheDocument()
    expect(screen.getByText('holidays page')).toBeInTheDocument()
  })

  it('highlights Home only on the root route', () => {
    renderShell('/')
    expect(within(sidebar()).getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page')
    expect(within(sidebar()).getByRole('link', { name: 'Holidays' })).not.toHaveAttribute('aria-current')
  })

  it('toggles collapsed state in the ui store', async () => {
    renderShell()
    await userEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }))
    expect(useUiStore.getState().sidebarCollapsed).toBe(true)
    expect(sidebar()).toHaveAttribute('data-collapsed', 'true')
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
    await userEvent.click(within(sidebar()).getByRole('link', { name: 'Home' }))
    expect(useUiStore.getState().sidebarMobileOpen).toBe(false)
  })
})

describe('Employee header', () => {
  it('renders a header with the page title and menu button', () => {
    renderShell()
    expect(screen.getByRole('banner')).toBeInTheDocument()
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Holidays')
    expect(screen.getByRole('button', { name: 'Open menu' })).toBeInTheDocument()
  })

  it('shows only Home on the root route', () => {
    renderShell('/')
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Home')
  })
})
