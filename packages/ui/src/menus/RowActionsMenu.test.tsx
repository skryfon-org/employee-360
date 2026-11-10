import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { RowActionsMenu } from './RowActionsMenu'

function setup() {
  const onEdit = vi.fn()
  const onDelete = vi.fn()
  const onOff = vi.fn()
  render(
    <div>
      <RowActionsMenu
        label="Sales"
        actions={[
          { label: 'Edit', onSelect: onEdit },
          { label: 'Locked', onSelect: onOff, disabled: true },
          { label: 'Delete', onSelect: onDelete, danger: true },
        ]}
      />
      <p>outside</p>
    </div>,
  )
  return { onEdit, onDelete, onOff, trigger: screen.getByRole('button', { name: 'Actions for Sales' }) }
}

describe('RowActionsMenu', () => {
  it('exposes menu semantics and opens/closes via the trigger', () => {
    const { trigger } = setup()
    expect(trigger).toHaveAttribute('aria-haspopup', 'menu')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getAllByRole('menuitem')).toHaveLength(3)
    fireEvent.click(trigger)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('closes on Escape and returns focus to the trigger', () => {
    const { trigger } = setup()
    fireEvent.click(trigger)
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    expect(trigger).toHaveFocus()
  })

  it('closes on outside click', () => {
    const { trigger } = setup()
    fireEvent.click(trigger)
    fireEvent.mouseDown(screen.getByText('outside'))
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('navigates with arrow keys, skipping disabled items', () => {
    const { trigger } = setup()
    fireEvent.click(trigger)
    const [edit, locked, del] = screen.getAllByRole('menuitem')
    expect(edit).toHaveFocus()
    expect(locked).toBeDisabled()
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'ArrowDown' })
    expect(del).toHaveFocus()
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'ArrowDown' })
    expect(edit).toHaveFocus()
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'ArrowUp' })
    expect(del).toHaveFocus()
  })

  it('runs the action, closes and restores focus on item click', () => {
    const { trigger, onDelete, onOff } = setup()
    fireEvent.click(trigger)
    fireEvent.click(screen.getByRole('menuitem', { name: 'Locked' }))
    expect(onOff).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }))
    expect(onDelete).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    expect(trigger).toHaveFocus()
  })

  describe('placement', () => {
    const rect = (over: Partial<DOMRect>) => ({ top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0, x: 0, y: 0, toJSON: () => ({}), ...over }) as DOMRect

    function place(trigger: Partial<DOMRect>, innerHeight: number) {
      vi.stubGlobal('innerHeight', innerHeight)
      const spy = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
        if (this.getAttribute('role') === 'menu') return rect({ height: 120, width: 176 })
        return rect(trigger)
      })
      const { trigger: btn } = setup()
      fireEvent.click(btn)
      const style = screen.getByRole('menu').style
      spy.mockRestore()
      vi.unstubAllGlobals()
      return style
    }

    it('opens downward, right-aligned, by default', () => {
      const style = place({ top: 100, bottom: 132, left: 500, right: 532 }, 800)
      expect(style.top).toBe('136px')
      expect(style.left).toBe('356px')
    })

    it('flips up only when there is no room below and more above', () => {
      const style = place({ top: 700, bottom: 732, left: 500, right: 532 }, 760)
      expect(style.top).toBe(`${700 - 4 - 120}px`)
    })

    it('stays below when neither side fits but below has more room', () => {
      const style = place({ top: 20, bottom: 52, left: 500, right: 532 }, 120)
      expect(style.top).toBe('56px')
    })
  })
})
