import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Breadcrumbs } from './Breadcrumbs'
import { buildBreadcrumbs, humanise } from './breadcrumbs-utils'

const config = {
  '/': 'Dashboard',
  '/invitations': 'Invitations',
  '/invitations/new': 'Invite user',
  '/people/:id': (p: Record<string, string>) => `Person ${p.id}`,
}

describe('buildBreadcrumbs', () => {
  it('returns a single root crumb for / and aliases', () => {
    expect(buildBreadcrumbs('/', config)).toEqual([{ to: '/', label: 'Dashboard' }])
    expect(buildBreadcrumbs('/dashboard', config, ['/dashboard'])).toHaveLength(1)
  })
  it('builds nested trails', () => {
    expect(buildBreadcrumbs('/invitations/new', config).map((c) => c.label)).toEqual(['Dashboard', 'Invitations', 'Invite user'])
  })
  it('resolves dynamic params and falls back to humanised segments', () => {
    expect(buildBreadcrumbs('/people/42', config).map((c) => c.label)).toEqual(['Dashboard', 'People', 'Person 42'])
    expect(buildBreadcrumbs('/invitations/some-thing', config).at(-1)?.label).toBe('Some thing')
    expect(humanise('a%20b_c')).toBe('A b c')
  })
})

describe('Breadcrumbs', () => {
  it('links ancestors and marks the last crumb as the current page', () => {
    render(
      <MemoryRouter>
        <Breadcrumbs items={buildBreadcrumbs('/invitations/new', config)} />
      </MemoryRouter>,
    )
    const nav = screen.getByRole('navigation', { name: 'Breadcrumb' })
    const items = within(nav).getAllByRole('listitem')
    expect(items.length).toBeGreaterThanOrEqual(3)
    expect(within(nav).getByRole('link', { name: 'Dashboard' })).toHaveAttribute('href', '/')
    expect(within(nav).getByRole('link', { name: 'Invitations' })).toHaveAttribute('href', '/invitations')
    expect(within(nav).queryByRole('link', { name: 'Invite user' })).not.toBeInTheDocument()
    expect(within(nav).getByText('Invite user')).toHaveAttribute('aria-current', 'page')
    expect(nav.querySelectorAll('[aria-hidden="true"]').length).toBeGreaterThan(0)
  })
  it('shows a single plain Dashboard crumb on the root', () => {
    render(
      <MemoryRouter>
        <Breadcrumbs items={buildBreadcrumbs('/', config)} />
      </MemoryRouter>,
    )
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.getByText('Dashboard')).toHaveAttribute('aria-current', 'page')
  })
})
