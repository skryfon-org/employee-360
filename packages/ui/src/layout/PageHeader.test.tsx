import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PageContainer, PageHeader } from './PageHeader'

describe('PageHeader', () => {
  it('renders title, description and actions', () => {
    render(
      <PageContainer>
        <PageHeader title="Dashboard" description="Organisation overview" actions={<button>Go</button>} />
      </PageContainer>,
    )
    expect(screen.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeInTheDocument()
    expect(screen.getByText('Organisation overview')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Go' })).toBeInTheDocument()
  })
})
