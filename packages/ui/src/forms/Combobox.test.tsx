import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Combobox, COMBOBOX_MAX_RENDERED, filterOptions } from './Combobox'

const OPTIONS = ['Alpha', 'Beta', 'Gamma', 'Alphabet']

function Harness({ options = OPTIONS, initial = '', disabled, onChange }: { options?: string[]; initial?: string; disabled?: boolean; onChange?: (v: string) => void }) {
  const [v, setV] = useState(initial)
  return (
    <Combobox
      id="thing"
      label="Thing"
      value={v}
      options={options}
      disabled={disabled}
      error={undefined}
      onChange={(x) => {
        onChange?.(x)
        setV(x)
      }}
    />
  )
}

const rect = (r: Partial<DOMRect>) => ({ x: 0, y: 0, left: 0, right: 0, width: 0, height: 0, top: 0, bottom: 0, toJSON() {}, ...r }) as DOMRect

afterEach(() => vi.restoreAllMocks())

describe('filterOptions', () => {
  it('puts prefix matches first and is case-insensitive', () => {
    expect(filterOptions(['Beta', 'Alphabet', 'alpha', 'Omega'], 'ALP')).toEqual(['Alphabet', 'alpha'])
    expect(filterOptions(['xa', 'ax'], 'a')).toEqual(['ax', 'xa'])
    expect(filterOptions(OPTIONS, '  ')).toEqual(OPTIONS)
  })
})

describe('Combobox', () => {
  it('exposes combobox semantics and wires the label', async () => {
    render(<Harness />)
    const input = screen.getByLabelText('Thing')
    expect(input).toHaveAttribute('role', 'combobox')
    expect(input).toHaveAttribute('aria-expanded', 'false')
    await userEvent.click(input)
    expect(input).toHaveAttribute('aria-expanded', 'true')
    const list = screen.getByRole('listbox')
    expect(input).toHaveAttribute('aria-controls', list.id)
    expect(screen.getAllByRole('option')).toHaveLength(4)
    expect(input.getAttribute('aria-activedescendant')).toBe(screen.getAllByRole('option')[0].id)
  })

  it('opens below the input at its width by default', async () => {
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.tagName === 'INPUT') return rect({ top: 100, bottom: 136, left: 20, width: 300, right: 320, height: 36 })
      return rect({ height: 200 })
    })
    render(<Harness />)
    await userEvent.click(screen.getByLabelText('Thing'))
    const list = screen.getByRole('listbox')
    expect(list.style.top).toBe('140px')
    expect(list.style.left).toBe('20px')
    expect(list.style.width).toBe('300px')
    expect(list.style.maxHeight).toBe('240px')
    expect(list.style.position).toBe('fixed')
  })

  it('flips above only when there is not enough room below and more above', async () => {
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(600)
    const input = (top: number) => rect({ top, bottom: top + 36, left: 0, width: 300, right: 300, height: 36 })
    let inputRect = input(540) // 20px below, 536px above
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      return this.tagName === 'INPUT' ? inputRect : rect({ height: 200 })
    })
    const { unmount } = render(<Harness />)
    await userEvent.click(screen.getByLabelText('Thing'))
    expect(screen.getByRole('listbox').style.top).toBe(`${540 - 4 - 200}px`)
    unmount()

    inputRect = input(10) // little room above too: stays below
    render(<Harness />)
    await userEvent.click(screen.getByLabelText('Thing'))
    expect(screen.getByRole('listbox').style.top).toBe('50px')
  })

  it('filters while typing and selects with click', async () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)
    const input = screen.getByLabelText('Thing')
    await userEvent.click(input)
    await userEvent.type(input, 'gam')
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Gamma'])
    await userEvent.click(screen.getByRole('option', { name: 'Gamma' }))
    expect(onChange).toHaveBeenCalledWith('Gamma')
    expect(input).toHaveValue('Gamma')
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('shows a no-matches row', async () => {
    render(<Harness />)
    await userEvent.type(screen.getByLabelText('Thing'), 'zzz')
    expect(screen.getByText('No matches')).toBeInTheDocument()
    expect(screen.queryAllByRole('option')).toHaveLength(0)
  })

  it('supports ArrowUp/Down, Home, End, Enter and Escape', async () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)
    const input = screen.getByLabelText('Thing')
    input.focus()
    await userEvent.keyboard('{ArrowDown}')
    expect(input).toHaveAttribute('aria-expanded', 'true')
    await userEvent.keyboard('{ArrowDown}{ArrowDown}')
    expect(input.getAttribute('aria-activedescendant')).toBe(screen.getAllByRole('option')[2].id)
    await userEvent.keyboard('{Home}')
    expect(input.getAttribute('aria-activedescendant')).toBe(screen.getAllByRole('option')[0].id)
    await userEvent.keyboard('{End}')
    expect(input.getAttribute('aria-activedescendant')).toBe(screen.getAllByRole('option')[3].id)
    await userEvent.keyboard('{ArrowUp}{Enter}')
    expect(onChange).toHaveBeenCalledWith('Gamma')
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()

    await userEvent.keyboard('{ArrowDown}{Escape}')
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    expect(input).toHaveValue('Gamma')
  })

  it('closes on outside click without changing the value', async () => {
    render(
      <div>
        <Harness initial="Beta" />
        <button>outside</button>
      </div>,
    )
    await userEvent.click(screen.getByLabelText('Thing'))
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'outside' }))
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Thing')).toHaveValue('Beta')
  })

  it('clears the value when the text is emptied and focus leaves', async () => {
    const onChange = vi.fn()
    render(<Harness initial="Beta" onChange={onChange} />)
    const input = screen.getByLabelText('Thing')
    await userEvent.click(input)
    await userEvent.clear(input)
    await userEvent.tab()
    expect(onChange).toHaveBeenCalledWith('')
  })

  it('does not open when disabled', async () => {
    render(<Harness disabled />)
    const input = screen.getByLabelText('Thing')
    expect(input).toBeDisabled()
    await userEvent.click(input)
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('caps rendered options and tells the user to narrow the list', async () => {
    const many = Array.from({ length: 5000 }, (_, i) => `City ${i}`)
    render(<Harness options={many} />)
    await userEvent.click(screen.getByLabelText('Thing'))
    expect(screen.getAllByRole('option')).toHaveLength(COMBOBOX_MAX_RENDERED)
    expect(screen.getByText(/Type to narrow results/)).toBeInTheDocument()
  })

  it('marks invalid and links the error', () => {
    render(<Combobox id="x" label="X" value="" options={[]} onChange={() => {}} error="Required" />)
    const input = screen.getByLabelText('X')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('alert')).toHaveTextContent('Required')
    expect(input).toHaveAttribute('aria-describedby', 'x-error')
  })
})
