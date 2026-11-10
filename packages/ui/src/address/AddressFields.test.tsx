import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AddressFields, type AddressFieldsValue } from './AddressFields'
import { cascadePatch } from './cascade'

function Harness({ initial, spy }: { initial?: Partial<AddressFieldsValue>; spy?: (v: AddressFieldsValue) => void }) {
  const [v, setV] = useState<AddressFieldsValue>({ country: '', state: '', city: '', pincode: '', ...initial })
  return (
    <AddressFields
      value={v}
      required={{ country: true }}
      onChange={(p) => {
        const next = { ...v, ...p }
        spy?.(next)
        setV(next)
      }}
    />
  )
}

const pick = async (label: string, option: string) => {
  const input = screen.getByLabelText(new RegExp(`^${label}`))
  await waitFor(() => expect(input).toBeEnabled(), { timeout: 15000 })
  await userEvent.click(input)
  await userEvent.type(input, option)
  await userEvent.click(await screen.findByRole('option', { name: option }, { timeout: 15000 }))
}

describe('AddressFields', () => {
  it('cascades India -> Kerala -> city and resets children on parent change', async () => {
    render(<Harness />)
    expect(screen.getByLabelText('State')).toBeDisabled()
    expect(screen.getByLabelText('City')).toBeDisabled()

    await pick('Country', 'India')
    await waitFor(() => expect(screen.getByLabelText('State')).toBeEnabled())
    expect(screen.getByLabelText('City')).toBeDisabled()

    await pick('State', 'Kerala')
    await waitFor(() => expect(screen.getByLabelText('City')).toBeEnabled(), { timeout: 15000 })
    await pick('City', 'Kozhikode')
    expect(screen.getByLabelText('City')).toHaveValue('Kozhikode')

    // Changing the state resets the city.
    await pick('State', 'Goa')
    expect(screen.getByLabelText('City')).not.toHaveValue('Kozhikode')

    // Changing the country resets state and city.
    await pick('State', 'Kerala')
    await pick('Country', 'Monaco')
    expect(screen.getByLabelText('State')).toHaveValue('')
  }, 30000)

  it('prefills stored values and keeps ones missing from the dataset', async () => {
    render(<Harness initial={{ country: 'India', state: 'Kerala', city: 'Smallville', pincode: '682001' }} />)
    await waitFor(() => expect(screen.getByLabelText(/^Country/)).toHaveValue('India'))
    await waitFor(() => expect(screen.getByLabelText('State')).toHaveValue('Kerala'))
    await waitFor(() => expect(screen.getByLabelText('City')).toHaveValue('Smallville'), { timeout: 15000 })
    await userEvent.click(screen.getByLabelText('City'))
    expect(await screen.findByRole('option', { name: 'Kozhikode' })).toBeInTheDocument()
    expect(screen.getByLabelText('Pincode')).toHaveValue('682001')
  }, 30000)

  it('keeps a stored country that is not in the dataset and uses free text below it', async () => {
    render(<Harness initial={{ country: 'Narnia', state: 'Cair', city: 'Paravel' }} />)
    await waitFor(() => expect(screen.getByLabelText(/^Country/)).toHaveValue('Narnia'))
    expect(screen.getByLabelText('State')).toHaveValue('Cair')
    expect(screen.getByLabelText('State').tagName).toBe('INPUT')
    expect(screen.getByLabelText('City').tagName).toBe('INPUT')
  })

  it('reports cascade patches through onChange', async () => {
    const spy = vi.fn()
    render(<Harness initial={{ country: 'India', state: 'Kerala' }} spy={spy} />)
    await pick('Country', 'Monaco')
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ ...cascadePatch('country', 'Monaco') }))
  })

  it('shows field errors with role=alert', () => {
    render(<AddressFields value={{ country: '', state: '', city: '', pincode: '1' }} onChange={() => {}} errors={{ pincode: 'Enter a valid 6-digit PIN code' }} />)
    expect(screen.getByRole('alert')).toHaveTextContent('Enter a valid 6-digit PIN code')
    expect(screen.getByLabelText('Pincode')).toHaveAttribute('aria-invalid', 'true')
  })
})
