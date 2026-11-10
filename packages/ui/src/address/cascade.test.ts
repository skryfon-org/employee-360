import { describe, expect, it } from 'vitest'
import { computeAddressOptions, cascadePatch } from './cascade'
import type { CityIndex, CoreData } from './dataset'

const core: CoreData = {
  countries: [
    { name: 'India', isoCode: 'IN' },
    { name: 'Monaco', isoCode: 'MC' },
  ],
  statesByCountry: new Map([
    [
      'IN',
      [
        { name: 'Kerala', isoCode: 'KL', countryCode: 'IN' },
        { name: 'Goa', isoCode: 'GA', countryCode: 'IN' },
      ],
    ],
  ]),
}
const cities: CityIndex = new Map([['IN|KL', ['Kochi', 'Thrissur']]])
const v = (country = '', state = '', city = '') => ({ country, state, city })

describe('cascadePatch', () => {
  it('country change resets state and city', () => {
    expect(cascadePatch('country', 'India')).toEqual({ country: 'India', state: '', city: '' })
  })
  it('state change resets city only', () => {
    expect(cascadePatch('state', 'Kerala')).toEqual({ state: 'Kerala', city: '' })
  })
  it('city change touches nothing else', () => {
    expect(cascadePatch('city', 'Kochi')).toEqual({ city: 'Kochi' })
  })
})

describe('computeAddressOptions', () => {
  it('disables state and city until their parent is chosen', () => {
    const o = computeAddressOptions(core, null, v())
    expect(o.stateMode).toBe('select')
    expect(o.stateDisabled).toBe(true)
    expect(o.cityDisabled).toBe(true)
    expect(o.countries).toEqual(['India', 'Monaco'])
  })
  it('lists the states of the chosen country and keeps city disabled', () => {
    const o = computeAddressOptions(core, null, v('India'))
    expect(o.states).toEqual(['Kerala', 'Goa'])
    expect(o.stateDisabled).toBe(false)
    expect(o.cityDisabled).toBe(true)
  })
  it('lists the cities of the chosen state; disabled while the city data loads', () => {
    expect(computeAddressOptions(core, null, v('India', 'Kerala')).cityDisabled).toBe(true)
    const o = computeAddressOptions(core, cities, v('India', 'Kerala'))
    expect(o.cities).toEqual(['Kochi', 'Thrissur'])
    expect(o.cityMode).toBe('select')
    expect(o.cityDisabled).toBe(false)
  })
  it('matches stored values case-insensitively to the canonical option', () => {
    const o = computeAddressOptions(core, cities, v('india', 'KERALA', 'Kochi'))
    expect(o.countryValue).toBe('India')
    expect(o.stateValue).toBe('Kerala')
    expect(o.cities).toEqual(['Kochi', 'Thrissur'])
  })
  it('keeps a stored city that is not in the dataset selectable', () => {
    const o = computeAddressOptions(core, cities, v('India', 'Kerala', 'Smallville'))
    expect(o.cities).toEqual(['Kochi', 'Thrissur', 'Smallville'])
    expect(o.cityValue).toBe('Smallville')
  })
  it('keeps a stored unknown country visible and falls back to free text below it', () => {
    const o = computeAddressOptions(core, cities, v('Narnia', 'Cair', 'Paravel'))
    expect(o.countries).toContain('Narnia')
    expect(o.countryValue).toBe('Narnia')
    expect(o.stateMode).toBe('text')
    expect(o.cityMode).toBe('text')
  })
  it('falls back to free text for a country without states', () => {
    const o = computeAddressOptions(core, cities, v('Monaco'))
    expect(o.stateMode).toBe('text')
    expect(o.stateDisabled).toBe(false)
    expect(o.cityMode).toBe('text')
  })
  it('keeps an unknown stored state and uses free-text city under it', () => {
    const o = computeAddressOptions(core, cities, v('India', 'Atlantis'))
    expect(o.states).toContain('Atlantis')
    expect(o.cityMode).toBe('text')
  })
  it('falls back to free-text city when the state has no cities', () => {
    const o = computeAddressOptions(core, cities, v('India', 'Goa'))
    expect(o.cityMode).toBe('text')
  })
})
