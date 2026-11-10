import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Skeleton, SkeletonRegion, SkeletonText } from './Skeleton'

describe('Skeleton', () => {
  it('renders an aria-hidden flat pulsing block that respects reduced motion', () => {
    const { container } = render(<Skeleton className="h-4 w-8" />)
    const el = container.firstElementChild as HTMLElement
    expect(el).toHaveAttribute('aria-hidden', 'true')
    expect(el).toHaveClass('animate-pulse', 'motion-reduce:animate-none', 'rounded-sm', 'bg-slate-200', 'h-4', 'w-8')
  })

  it('SkeletonText renders the requested number of lines', () => {
    const { container } = render(<SkeletonText lines={3} />)
    expect(container.querySelectorAll('.animate-pulse')).toHaveLength(3)
  })

  it('SkeletonRegion is a busy status region with accessible text', () => {
    render(<SkeletonRegion label="Loading things"><Skeleton /></SkeletonRegion>)
    const region = screen.getByRole('status')
    expect(region).toHaveAttribute('aria-busy', 'true')
    expect(region).toHaveTextContent('Loading things')
  })

  it('SkeletonRegion defaults the label to Loading…', () => {
    render(<SkeletonRegion>x</SkeletonRegion>)
    expect(screen.getByRole('status')).toHaveTextContent('Loading…')
  })
})
