import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { InfoIcon } from './InfoIcon'

describe('InfoIcon', () => {
  it('renders a decorative svg with the given class', () => {
    const { container } = render(<InfoIcon className="h-5 w-5" />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('aria-hidden', 'true')
    expect(svg).toHaveClass('h-5')
  })
})
