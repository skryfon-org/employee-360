import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { UserBadge, humanizeRole, pickPrimaryRole } from './UserBadge'

describe('UserBadge', () => {
  it('shows email and humanized role', () => {
    render(<UserBadge email="a@x.com" role="super_admin" />)
    expect(screen.getAllByText('Super Admin').length).toBeGreaterThan(0)
    expect(screen.getByText('a@x.com')).toBeInTheDocument()
  })
  it('prefers the name over the email', () => {
    render(<UserBadge email="a@x.com" name="Ada Lovelace" />)
    expect(screen.getByText('Ada Lovelace')).toBeInTheDocument()
    expect(screen.queryByText('a@x.com')).toBeNull()
  })
  it('omits the role when not provided and renders nothing without email', () => {
    const { container, rerender } = render(<UserBadge email="a@x.com" />)
    expect(screen.queryByText('Admin')).toBeNull()
    rerender(<UserBadge />)
    expect(container).toBeEmptyDOMElement()
  })
  it('humanizes and picks the highest role', () => {
    expect(humanizeRole('employee')).toBe('Employee')
    expect(pickPrimaryRole(['employee', 'admin'])).toBe('admin')
    expect(pickPrimaryRole(['admin', 'super_admin'])).toBe('super_admin')
    expect(pickPrimaryRole([])).toBeUndefined()
  })
})
