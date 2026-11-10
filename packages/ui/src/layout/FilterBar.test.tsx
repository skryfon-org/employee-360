import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { EmptyState } from './EmptyState'
import { FilterBar } from './FilterBar'

describe('FilterBar', () => {
  it('renders children with the strip styling and forwards props', () => {
    render(
      <FilterBar role="search" className="grid" data-testid="fb">
        <span>child</span>
      </FilterBar>,
    )
    const el = screen.getByTestId('fb')
    expect(el).toHaveAttribute('role', 'search')
    expect(el.className).toContain('border-b')
    expect(el.className).toContain('grid')
    expect(screen.getByText('child')).toBeInTheDocument()
  })
})

describe('EmptyState', () => {
  it('renders title, description and action', () => {
    render(<EmptyState title="No items" description="Nothing here." action={<button>Reset</button>} role="status" />)
    expect(screen.getByRole('status')).toBeInTheDocument()
    expect(screen.getByText('No items')).toBeInTheDocument()
    expect(screen.getByText('Nothing here.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).toBeInTheDocument()
  })
  it('omits description when absent', () => {
    render(<EmptyState title="Only title" />)
    expect(screen.getByText('Only title')).toBeInTheDocument()
  })
})
