import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import { Switch } from './Switch'

function Harness() {
  const [on, setOn] = useState(true)
  return <Switch checked={on} onChange={setOn} label="Active" description="Helper text" />
}

describe('Switch', () => {
  it('exposes switch role, toggles by click and keyboard', () => {
    render(<Harness />)
    const sw = screen.getByRole('switch', { name: 'Active' })
    expect(sw).toHaveAttribute('aria-checked', 'true')
    expect(sw).toHaveAccessibleDescription('Helper text')
    fireEvent.click(sw)
    expect(sw).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(sw) // native button: Space/Enter dispatch click
    expect(sw).toHaveAttribute('aria-checked', 'true')
  })
})
