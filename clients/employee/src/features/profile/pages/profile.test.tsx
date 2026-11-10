import { useToast } from '@employee360/ui'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import MockAdapter from 'axios-mock-adapter'
import { apiClient, clearSession, setSession } from '@employee360/api-client'
import App from '../../../App'
import { createQueryClient } from '../../../lib/queryClient'
import { useAuthStore } from '../../../stores/authStore'
import { useProfileGateStore } from '../../../stores/profileGateStore'

let mock: MockAdapter

interface MeState {
  roles: string[]
  profile_required: boolean
  profile_completed: boolean
  first_login: boolean
}
let me: MeState
let contacts: Record<string, unknown>[]
let profile: Record<string, unknown>
let placement: Record<string, unknown> = {}
let missing: string[]
let meFails = false

const incompleteMe: MeState = { roles: ['employee'], profile_required: true, profile_completed: false, first_login: true }

function renderAt(path: string) {
  return render(
    <QueryClientProvider client={createQueryClient({ queries: { retry: false }, mutations: { retry: false } })}>
      <MemoryRouter initialEntries={[path]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function signIn(roles = ['employee']) {
  setSession({ accessToken: 'a', refreshToken: 'r', tenantId: 't1' })
  useAuthStore.setState({ accessToken: 'a', user: { id: 'u1', email: 'emp@x.com', firstName: 'Jane', lastName: 'Doe', roles } })
}

const heading = (name: string | RegExp) => screen.findByRole('heading', { name })

beforeEach(() => {
  meFails = false
  me = { ...incompleteMe }
  contacts = []
  profile = {}
  placement = {}
  missing = ['phone', 'date_of_birth', 'gender', 'address_line1', 'city', 'country', 'emergency_contact']
  mock = new MockAdapter(apiClient)
  mock.onGet('/api/v1/me').reply(() => meFails ? [500, { success: false, error: { message: 'boom' } }] : [
    200,
    { success: true, data: { id: 'u1', email: 'emp@x.com', first_name: 'Jane', last_name: 'Doe', organization_name: 'Acme Corp', ...me } },
  ])
  mock.onGet('/api/v1/me/profile').reply(() => [
    200,
    { success: true, data: { ...placement, profile, emergency_contacts: contacts, missing_fields: missing, profile_completed: me.profile_completed } },
  ])
  mock.onGet('/api/v1/health').reply(200, { success: true, data: { status: 'ok', app: 'employee360', database: 'ok' } })
  mock.onGet('/api/v1/dashboard/employee').reply(200, { success: true, data: { greeting: { first_name: 'Jane', welcome_message: '', today: { date: '2026-10-30', kind: 'working_day' } }, stats: {}, leave: { recent: [] }, holidays: [], projects: [], week: { days: [], missing_days: 0 } } })
  clearSession()
  useAuthStore.getState().clear()
  useProfileGateStore.getState().reset()
  localStorage.clear()
})
afterEach(() => {
  mock.restore()
  useToast().dismiss() // sonner's store is module-global; don't leak toasts into the next test
})

// The real country/state/city dataset (~8 MB of JSON) takes seconds to parse and index in jsdom; use a tiny fixture.
vi.mock('@employee360/ui/src/address/dataset', () => ({
  loadCoreData: async () => ({
    countries: [
      { name: 'India', isoCode: 'IN' },
      { name: 'Monaco', isoCode: 'MC' },
      { name: 'United States', isoCode: 'US' },
    ],
    statesByCountry: new Map([
      [
        'IN',
        [
          { name: 'Goa', isoCode: 'GA', countryCode: 'IN' },
          { name: 'Kerala', isoCode: 'KL', countryCode: 'IN' },
          { name: 'Maharashtra', isoCode: 'MH', countryCode: 'IN' },
        ],
      ],
    ]),
  }),
  loadCityIndex: async () =>
    new Map([
      ['IN|GA', ['Panaji']],
      ['IN|KL', ['Kozhikode']],
      ['IN|MH', ['Mumbai', 'Pune']],
    ]),
}))

/** Picks an option in a (lazily populated) address combobox: wait until enabled, type to filter, click the option. */
async function pickOption(label: string, option: string) {
  const input = screen.getByLabelText(label)
  await waitFor(() => expect(input).toBeEnabled())
  await userEvent.click(input)
  await userEvent.type(input, option)
  await userEvent.click(await screen.findByRole('option', { name: option }))
}

async function pickRowAction(trigger: string, item: string) {
  await userEvent.click(await screen.findByRole('button', { name: trigger }))
  await userEvent.click(await screen.findByRole('menuitem', { name: item }))
}

describe('welcome and gating', () => {
  it('redirects an incomplete first-login employee to the welcome screen with the organization name', async () => {
    signIn()
    renderAt('/')
    expect(await heading('Welcome to Acme Corp')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Holidays' })).not.toBeInTheDocument()
  })

  it('continues from welcome to profile completion', async () => {
    signIn()
    renderAt('/holidays')
    await userEvent.click(await screen.findByRole('button', { name: 'Get started' }))
    expect(await heading('Complete your profile')).toBeInTheDocument()
    expect(await screen.findByRole('form', { name: 'Personal details' })).toBeInTheDocument()
    expect(screen.getByText('Still needed to finish your profile:')).toBeInTheDocument()
  })

  it('persists the welcome dismissal in sessionStorage until reset on sign-out', async () => {
    signIn()
    renderAt('/holidays')
    await userEvent.click(await screen.findByRole('button', { name: 'Get started' }))
    expect(sessionStorage.getItem('e360.welcomeDismissed')).toBe('1')
    useProfileGateStore.getState().reset()
    expect(sessionStorage.getItem('e360.welcomeDismissed')).toBeNull()
    expect(useProfileGateStore.getState().welcomeDismissed).toBe(false)
  })

  it('goes straight to profile completion when it is not a first login', async () => {
    me = { ...incompleteMe, first_login: false }
    signIn()
    renderAt('/')
    expect(await heading('Complete your profile')).toBeInTheDocument()
  })

  it('blocks every other route until the profile is complete', async () => {
    me = { ...incompleteMe, first_login: false }
    signIn()
    renderAt('/profile')
    expect(await heading('Complete your profile')).toBeInTheDocument()
  })

  it('lets a completed employee through and keeps them off the setup pages', async () => {
    me = { roles: ['employee'], profile_required: true, profile_completed: true, first_login: false }
    signIn()
    renderAt('/complete-profile')
    expect(await screen.findByText(/Good (morning|afternoon|evening), Jane/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'My profile' })).toHaveAttribute('href', '/profile')
  })

  it('never shows welcome or the gate to admins', async () => {
    me = { roles: ['admin'], profile_required: false, profile_completed: false, first_login: false }
    signIn(['admin'])
    renderAt('/welcome')
    expect(await screen.findByText(/Good (morning|afternoon|evening), Jane/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Get started' })).not.toBeInTheDocument()
  })

  it('redirects to profile completion when an API call returns 403 profile_incomplete', async () => {
    me = { roles: ['employee'], profile_required: true, profile_completed: true, first_login: false }
    signIn()
    mock.onGet('/api/v1/holiday-categories').reply(200, { success: true, data: [] })
    mock.onGet('/api/v1/holidays').reply(() => {
      // The server now says the profile is incomplete (e.g. the last contact was removed elsewhere).
      me = { ...incompleteMe, first_login: false }
      return [403, { success: false, error: { code: 'profile_incomplete', message: 'complete your profile' } }]
    })
    renderAt('/holidays')
    expect(await heading('Complete your profile')).toBeInTheDocument()
  })

  it('shows an error with retry when /me fails', async () => {
    signIn()
    meFails = true
    renderAt('/')
    expect(await screen.findByRole('alert')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})

describe('profile completion wizard', () => {
  beforeEach(() => {
    me = { ...incompleteMe, first_login: false }
    signIn()
  })

  it('validates mandatory personal fields before saving', async () => {
    renderAt('/complete-profile')
    await screen.findByRole('form', { name: 'Personal details' })
    await userEvent.click(screen.getByRole('button', { name: 'Save and continue' }))
    expect(await screen.findByText('Phone is required')).toBeInTheDocument()
    expect(screen.getByText('Date of birth is required')).toBeInTheDocument()
    expect(screen.getByText('Gender is required')).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)

    await userEvent.type(screen.getByLabelText('Phone *'), 'abc')
    await userEvent.type(screen.getByLabelText('Personal email'), 'nope')
    await userEvent.type(screen.getByLabelText('Date of birth *'), '1899-01-01')
    await userEvent.click(screen.getByRole('button', { name: 'Save and continue' }))
    expect(await screen.findByText('Enter a valid phone number')).toBeInTheDocument()
    expect(screen.getByText(/Enter a valid email address/)).toBeInTheDocument()
    expect(screen.getByText('Date of birth must be in the past and after 1900')).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
  })

  it('walks through personal, address and contacts, then releases the gate', async () => {
    mock.onPatch('/api/v1/me/profile').reply(200, { success: true, data: {} })
    mock.onPost('/api/v1/me/emergency-contacts').reply(() => {
      contacts = [{ id: 'c1', name: 'Bo', relationship: 'Sibling', phone: '5559876', is_primary: true }]
      missing = []
      me = { roles: ['employee'], profile_required: true, profile_completed: true, first_login: false }
      return [201, { success: true, data: contacts[0] }]
    })
    renderAt('/complete-profile')
    await screen.findByRole('form', { name: 'Personal details' })

    await userEvent.type(screen.getByLabelText('Phone *'), '5551234567')
    await userEvent.type(screen.getByLabelText('Date of birth *'), '1990-05-01')
    await userEvent.selectOptions(screen.getByLabelText('Gender *'), 'female')
    await userEvent.click(screen.getByRole('button', { name: 'Save and continue' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toMatchObject({ phone: '5551234567', date_of_birth: '1990-05-01', gender: 'female' })
    expect(JSON.parse(mock.history.patch[0].data)).not.toHaveProperty('city')

    await screen.findByRole('form', { name: 'Address' })
    await userEvent.type(screen.getByLabelText('Address line 1 *'), '1 Main St')
    await pickOption('Country *', 'India')
    await pickOption('State', 'Maharashtra')
    await pickOption('City *', 'Pune')
    await userEvent.click(screen.getByRole('button', { name: 'Save and continue' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(2))

    expect(await screen.findByRole('heading', { name: 'Emergency contacts' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Finish' })).toBeDisabled()
    await userEvent.click(screen.getByRole('button', { name: 'Add emergency contact' }))
    const form = screen.getByRole('form', { name: 'Add emergency contact' })
    expect(within(form).getByLabelText(/Primary contact/)).toBeChecked()
    expect(within(form).getByLabelText(/Primary contact/)).toBeDisabled()
    await userEvent.type(within(form).getByLabelText('Name *'), 'Bo')
    await userEvent.type(within(form).getByLabelText('Relationship *'), 'Sibling')
    await userEvent.type(within(form).getByLabelText('Phone *'), '5559876')
    await userEvent.click(within(form).getByRole('button', { name: 'Add contact' }))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({ name: 'Bo', is_primary: true })

    const finish = await screen.findByRole('button', { name: 'Finish' })
    await waitFor(() => expect(finish).toBeEnabled())
    await userEvent.click(finish)
    expect(await screen.findByText(/Good (morning|afternoon|evening), Jane/)).toBeInTheDocument()
  })

  it('sends the user back with the missing fields when the backend still reports some', async () => {
    contacts = [{ id: 'c1', name: 'Bo', relationship: 'Sibling', phone: '5559876', is_primary: true }]
    missing = ['phone']
    renderAt('/complete-profile')
    await screen.findByRole('form', { name: 'Personal details' })
    expect(screen.getByText('Phone', { selector: 'li' })).toBeInTheDocument()
    expect(screen.queryByText('At least one emergency contact')).not.toBeInTheDocument()
  })
})

describe('profile page', () => {
  beforeEach(() => {
    me = { roles: ['employee'], profile_required: true, profile_completed: true, first_login: false }
    missing = []
    profile = {
      phone: '5551234567', date_of_birth: '1990-05-01', gender: 'female', address_line1: '1 Main St', city: 'Pune', country: 'India',
      employee_code: 'E-100', employment_status: 'active', employment_type: 'full_time', work_location: 'Pune HQ',
    }
    contacts = [{ id: 'c1', name: 'Bo', relationship: 'Sibling', phone: '5559876', is_primary: true }]
    placement = { first_name: 'Janet', last_name: 'Doer', department_id: 'd1', department_name: 'Engineering', position_id: 'p1', position_name: 'Senior Engineer' }
    signIn()
  })

  it('redirects to the overview and shows the sub-navigation', async () => {
    renderAt('/profile')
    expect(await heading('My profile')).toBeInTheDocument()
    const nav = screen.getByRole('navigation', { name: 'Profile sections' })
    expect(within(nav).getAllByRole('link').map((l) => l.getAttribute('href'))).toEqual(
      ['overview', 'employment', 'personal', 'address', 'emergency-contacts'].map((p) => `/profile/${p}`),
    )
    await waitFor(() => expect(within(nav).getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page'))
  })

  it('shows employment read-only with no Edit button', async () => {
    renderAt('/profile/employment')
    const employment = (await screen.findByRole('heading', { name: 'Employment' })).closest('section') as HTMLElement
    expect(within(employment).getByText('E-100')).toBeInTheDocument()
    expect(within(employment).getByText('Pune HQ')).toBeInTheDocument()
    expect(within(employment).getByText('Engineering')).toBeInTheDocument()
    expect(within(employment).getByText('Senior Engineer')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Edit/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  })

  it('falls back to a dash for department and position when unassigned', async () => {
    placement = {}
    renderAt('/profile/employment')
    const employment = (await screen.findByRole('heading', { name: 'Employment' })).closest('section') as HTMLElement
    const dept = within(employment).getByText('Department').closest('div') as HTMLElement
    const pos = within(employment).getByText('Position').closest('div') as HTMLElement
    expect(within(dept).getByText('—')).toBeInTheDocument()
    expect(within(pos).getByText('—')).toBeInTheDocument()
  })

  it('shows department and position in the overview', async () => {
    renderAt('/profile/overview')
    expect(await screen.findByText('Engineering')).toBeInTheDocument()
    expect(screen.getByText('Senior Engineer')).toBeInTheDocument()
  })

  it('shows the profile name in the overview', async () => {
    renderAt('/profile/overview')
    expect(await screen.findByText('Janet Doer')).toBeInTheDocument()
  })

  it('shows names in personal view and patches them only when changed', async () => {
    mock.onPatch('/api/v1/me/profile').reply(200, { success: true, data: {} })
    renderAt('/profile/personal')
    expect(await screen.findByText('Janet')).toBeInTheDocument()
    expect(screen.getByText('Doer')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit personal details' }))
    expect(screen.getByLabelText('First name *')).toHaveValue('Janet')
    await userEvent.type(screen.getByLabelText('Nationality'), 'Indian')
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    const body = JSON.parse(mock.history.patch[0].data)
    expect(body).not.toHaveProperty('first_name')
    expect(body).not.toHaveProperty('last_name')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit personal details' }))
    const first = screen.getByLabelText('First name *')
    await userEvent.clear(first)
    await userEvent.type(first, '  Jan  ')
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(2))
    const body2 = JSON.parse(mock.history.patch[1].data)
    expect(body2.first_name).toBe('Jan')
    expect(body2).not.toHaveProperty('last_name')
  })

  it('confirms a saved profile edit with a toast', async () => {
    mock.onPatch('/api/v1/me/profile').reply(200, { success: true, data: {} })
    renderAt('/profile/personal')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit personal details' }))
    await userEvent.type(screen.getByLabelText('Nationality'), 'Indian')
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    expect(await screen.findByText('Profile updated.')).toBeInTheDocument()
  })

  it('requires a first name when editing', async () => {
    renderAt('/profile/personal')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit personal details' }))
    await userEvent.clear(screen.getByLabelText('First name *'))
    await userEvent.click(screen.getByRole('button', { name: 'Save personal details' }))
    expect(await screen.findByText('First name is required')).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
  })

  it('opens personal details in view mode and cancel discards edits', async () => {
    renderAt('/profile/personal')
    expect(await screen.findByText('5551234567')).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Personal details' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit personal details' }))
    expect(screen.getByLabelText('Phone *')).toHaveValue('5551234567')
    await userEvent.type(screen.getByLabelText('Nationality'), 'Indian')
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('form', { name: 'Personal details' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit personal details' }))
    expect(screen.getByLabelText('Nationality')).toHaveValue('')
    expect(mock.history.patch).toHaveLength(0)
  })

  it('saves address edits via PATCH /me/profile and returns to view mode', async () => {
    mock.onPatch('/api/v1/me/profile').reply(200, { success: true, data: {} })
    renderAt('/profile/address')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit address' }))
    await pickOption('State', 'Maharashtra')
    expect(screen.getByLabelText('City *')).toHaveValue('')
    await pickOption('City *', 'Mumbai')
    await userEvent.click(screen.getByRole('button', { name: 'Save address' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toMatchObject({ country: 'India', state: 'Maharashtra', city: 'Mumbai' })
    await waitFor(() => expect(screen.queryByRole('form', { name: 'Address' })).not.toBeInTheDocument())
  })

  it('cascades country -> state -> city, resets children and prefills stored values in edit mode', async () => {
    profile = { ...profile, country: 'India', state: 'Kerala', city: 'Kozhikode', pincode: '673001' }
    renderAt('/profile/address')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit address' }))
    await waitFor(() => expect(screen.getByLabelText('Country *')).toHaveValue('India'))
    await waitFor(() => expect(screen.getByLabelText('State')).toHaveValue('Kerala'))
    await waitFor(() => expect(screen.getByLabelText('City *')).toHaveValue('Kozhikode'))
    expect(screen.getByLabelText('Pincode')).toHaveValue('673001')

    await pickOption('State', 'Goa')
    expect(screen.getByLabelText('City *')).toHaveValue('')
    await pickOption('Country *', 'Monaco')
    expect(screen.getByLabelText('State')).toHaveValue('')
  })

  it('keeps a stored city that is not in the dataset and shows a specific PIN code error', async () => {
    profile = { ...profile, country: 'India', state: 'Kerala', city: 'Smallville', pincode: '6820' }
    mock.onPatch('/api/v1/me/profile').reply(200, { success: true, data: {} })
    renderAt('/profile/address')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit address' }))
    await waitFor(() => expect(screen.getByLabelText('City *')).toHaveValue('Smallville'))
    await userEvent.click(screen.getByRole('button', { name: 'Save address' }))
    expect(await screen.findByText('Enter a valid 6-digit PIN code')).toBeInTheDocument()
    expect(mock.history.patch).toHaveLength(0)
    await userEvent.clear(screen.getByLabelText('Pincode'))
    await userEvent.type(screen.getByLabelText('Pincode'), '682001')
    await waitFor(() => expect(screen.queryByText('Enter a valid 6-digit PIN code')).not.toBeInTheDocument())
    await userEvent.click(screen.getByRole('button', { name: 'Save address' }))
    await waitFor(() => expect(mock.history.patch).toHaveLength(1))
    expect(JSON.parse(mock.history.patch[0].data)).toMatchObject({ city: 'Smallville', pincode: '682001' })
  })

  it('revalidates the pincode when the country changes', async () => {
    profile = { ...profile, country: 'India', pincode: '90210' }
    renderAt('/profile/address')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit address' }))
    await waitFor(() => expect(screen.getByLabelText('Country *')).toHaveValue('India'))
    await userEvent.click(screen.getByRole('button', { name: 'Save address' }))
    expect(await screen.findByText('Enter a valid 6-digit PIN code')).toBeInTheDocument()
    await pickOption('Country *', 'United States')
    await waitFor(() => expect(screen.queryByText('Enter a valid 6-digit PIN code')).not.toBeInTheDocument())
  })

  it('lists contacts read-only until Edit enables add/edit/remove', async () => {
    renderAt('/profile/emergency-contacts')
    expect(await screen.findByText('Bo')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add emergency contact' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Actions for contact Bo' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Edit emergency contacts' }))
    await userEvent.click(screen.getByRole('button', { name: 'Actions for contact Bo' }))
    expect(screen.getByRole('menuitem', { name: 'Remove' })).toBeDisabled()
  })

  it('removes a contact after inline confirmation when more than one exists', async () => {
    contacts = [...contacts, { id: 'c2', name: 'Cy', relationship: 'Friend', phone: '5550000', is_primary: false }]
    mock.onDelete('/api/v1/me/emergency-contacts/c2').reply(200, { success: true })
    renderAt('/profile/emergency-contacts')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit emergency contacts' }))
    await pickRowAction('Actions for contact Cy', 'Remove')
    expect(mock.history.delete).toHaveLength(0)
    await userEvent.click(screen.getByRole('button', { name: 'Confirm remove Cy' }))
    await waitFor(() => expect(mock.history.delete).toHaveLength(1))
    expect(await screen.findByText('Emergency contact removed.')).toBeInTheDocument()
  })

  it('confirms adding and updating a contact with a toast', async () => {
    mock.onPost('/api/v1/me/emergency-contacts').reply(201, { success: true, data: { id: 'c3', name: 'Di', relationship: 'Aunt', phone: '5551111', is_primary: false } })
    mock.onPut('/api/v1/me/emergency-contacts/c1').reply(200, { success: true, data: contacts[0] })
    renderAt('/profile/emergency-contacts')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit emergency contacts' }))
    await userEvent.click(screen.getByRole('button', { name: 'Add emergency contact' }))
    const form = screen.getByRole('form', { name: 'Add emergency contact' })
    await userEvent.type(within(form).getByLabelText('Name *'), 'Di')
    await userEvent.type(within(form).getByLabelText('Relationship *'), 'Aunt')
    await userEvent.type(within(form).getByLabelText('Phone *'), '5551111')
    await userEvent.click(within(form).getByRole('button', { name: 'Add contact' }))
    expect(await screen.findByText('Emergency contact added.')).toBeInTheDocument()

    useToast().dismiss()
    await pickRowAction('Actions for contact Bo', 'Edit')
    const editForm = screen.getByRole('form', { name: 'Edit emergency contact' })
    await userEvent.type(within(editForm).getByLabelText('Relationship *'), 's')
    await userEvent.click(within(editForm).getByRole('button', { name: /^Save/ }))
    expect(await screen.findByText('Emergency contact updated.')).toBeInTheDocument()
  })

  it('disables adding past the 10 contact limit', async () => {
    contacts = Array.from({ length: 10 }, (_, i) => ({ id: `c${i}`, name: `Contact ${i}`, relationship: 'Friend', phone: '5550000', is_primary: i === 0 }))
    renderAt('/profile/emergency-contacts')
    await screen.findByText('Contact 9')
    await userEvent.click(screen.getByRole('button', { name: 'Edit emergency contacts' }))
    expect(screen.getByRole('button', { name: 'Add emergency contact' })).toBeDisabled()
  })

  it('validates a new contact', async () => {
    renderAt('/profile/emergency-contacts')
    await userEvent.click(await screen.findByRole('button', { name: 'Edit emergency contacts' }))
    await userEvent.click(screen.getByRole('button', { name: 'Add emergency contact' }))
    const form = screen.getByRole('form', { name: 'Add emergency contact' })
    await userEvent.click(within(form).getByRole('button', { name: 'Add contact' }))
    expect(await within(form).findByText('Name is required')).toBeInTheDocument()
    expect(within(form).getByText('Relationship is required')).toBeInTheDocument()
    expect(within(form).getByText('Phone is required')).toBeInTheDocument()
    expect(mock.history.post).toHaveLength(0)
  })
})
