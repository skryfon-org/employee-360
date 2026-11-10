import { useState } from 'react'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Tabs } from './Tabs'

const items = [
  { value: 'a', label: 'Alpha' },
  { value: 'b', label: 'Beta' },
]

describe('Tabs', () => {
  it('renders tabs with the active one selected and its panel', () => {
    render(<Tabs items={items} value="b" onValueChange={() => {}} label="Sections">content b</Tabs>)
    expect(screen.getByRole('tablist', { name: 'Sections' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Beta' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Alpha' })).toHaveAttribute('aria-selected', 'false')
    expect(screen.getByRole('tabpanel')).toHaveTextContent('content b')
  })

  it('calls onValueChange on click and arrow keys', async () => {
    const onChange = vi.fn()
    function Harness() {
      const [v, setV] = useState('a')
      return <Tabs items={items} value={v} onValueChange={(n) => { onChange(n); setV(n) }} label="Sections">{v}</Tabs>
    }
    render(<Harness />)
    fireEvent.mouseDown(screen.getByRole('tab', { name: 'Beta' }), { button: 0, ctrlKey: false })
    expect(onChange).toHaveBeenLastCalledWith('b')
    fireEvent.keyDown(screen.getByRole('tab', { name: 'Beta' }), { key: 'ArrowLeft' })
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith('a'))
  })
})
