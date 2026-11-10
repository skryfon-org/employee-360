import type { CityIndex, CoreData, CountryRec, StateRec } from './dataset'

export interface AddressValue {
  country: string
  state: string
  city: string
}
export type AddressField = keyof AddressValue

const norm = (s: string) => s.trim().toLocaleLowerCase()

export function findCountry(core: CoreData, name: string): CountryRec | undefined {
  const n = norm(name)
  return n ? core.countries.find((c) => norm(c.name) === n) : undefined
}

export function findState(core: CoreData, country: CountryRec | undefined, name: string): StateRec | undefined {
  const n = norm(name)
  if (!country || !n) return undefined
  return core.statesByCountry.get(country.isoCode)?.find((s) => norm(s.name) === n)
}

/**
 * Patch to apply when one address field changes: changing the country clears
 * state and city, changing the state clears the city.
 */
export function cascadePatch(field: AddressField, value: string): Partial<AddressValue> {
  if (field === 'country') return { country: value, state: '', city: '' }
  if (field === 'state') return { state: value, city: '' }
  return { city: value }
}

export type FieldMode = 'select' | 'text'

export interface AddressOptions {
  countries: string[]
  states: string[]
  cities: string[]
  /** Which control each field renders: dropdown, or free text when the dataset has no data. */
  stateMode: FieldMode
  cityMode: FieldMode
  /** The select is shown but cannot be used yet (parent not chosen, or cities still loading). */
  stateDisabled: boolean
  cityDisabled: boolean
  /** Canonical-cased value to show in each select (stored values are matched case-insensitively). */
  countryValue: string
  stateValue: string
  cityValue: string
}

/** Appends a stored value that is not in the dataset so it stays visible and selectable. */
function withCurrent(list: string[], current: string): string[] {
  return current && !list.includes(current) ? [...list, current] : list
}

/**
 * Derives the option lists and control modes for the current selection.
 * `core` null = still loading; `cityIndex` null = cities not loaded yet.
 */
export function computeAddressOptions(core: CoreData | null, cityIndex: CityIndex | null, v: AddressValue): AddressOptions {
  const country = core ? findCountry(core, v.country) : undefined
  const stateList = country && core ? (core.statesByCountry.get(country.isoCode) ?? []) : []
  const state = core ? findState(core, country, v.state) : undefined
  const countries = core ? core.countries.map((c) => c.name) : []
  const countryValue = country?.name ?? v.country.trim()

  const hasCountry = v.country.trim() !== ''
  // Free text when the (known or custom) country has no state list.
  const stateMode: FieldMode = hasCountry && core && stateList.length === 0 ? 'text' : 'select'
  const stateDisabled = stateMode === 'select' && (!hasCountry || !core)
  const states = withCurrent(
    stateList.map((s) => s.name),
    state?.name ?? v.state.trim(),
  )
  const stateValue = state?.name ?? v.state.trim()

  const hasState = v.state.trim() !== ''
  const cityList = state && cityIndex ? (cityIndex.get(`${state.countryCode}|${state.isoCode}`) ?? []) : []
  const citiesPending = !!state && !cityIndex
  let cityMode: FieldMode = 'select'
  if (stateMode === 'text') cityMode = 'text'
  else if (hasState && !state) cityMode = 'text' // custom stored state
  else if (state && cityIndex && cityList.length === 0) cityMode = 'text'
  const cityDisabled = cityMode === 'select' && (!hasState || citiesPending)
  const cityValue = v.city.trim()
  const cities = withCurrent(cityList, cityValue)

  return {
    countries: withCurrent(countries, countryValue),
    states,
    cities,
    stateMode,
    cityMode,
    stateDisabled,
    cityDisabled,
    countryValue,
    stateValue,
    cityValue,
  }
}
