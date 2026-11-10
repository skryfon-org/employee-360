/// <reference path="./json-modules.d.ts" />

export interface CountryRec {
  name: string
  isoCode: string
}
export interface StateRec {
  name: string
  isoCode: string
  countryCode: string
}
export interface CoreData {
  countries: CountryRec[]
  statesByCountry: Map<string, StateRec[]>
}
/** City names keyed by `${countryCode}|${stateCode}`. */
export type CityIndex = Map<string, string[]>

let corePromise: Promise<CoreData> | undefined
let cityPromise: Promise<CityIndex> | undefined

/** Countries and states (~650 KB), fetched as their own chunk on first use. */
export function loadCoreData(): Promise<CoreData> {
  corePromise ??= Promise.all([
    import('country-state-city/lib/assets/country.json'),
    import('country-state-city/lib/assets/state.json'),
  ]).then(([c, s]) => {
    const statesByCountry = new Map<string, StateRec[]>()
    for (const st of s.default) {
      const list = statesByCountry.get(st.countryCode)
      if (list) list.push(st)
      else statesByCountry.set(st.countryCode, [st])
    }
    return { countries: c.default, statesByCountry }
  })
  corePromise.catch(() => {
    corePromise = undefined
  })
  return corePromise
}

/** Cities (~8 MB raw), fetched as a separate chunk only once a state is picked. */
export function loadCityIndex(): Promise<CityIndex> {
  cityPromise ??= import('country-state-city/lib/assets/city.json').then((m) => {
    const idx: CityIndex = new Map()
    for (const [name, cc, sc] of m.default) {
      const key = `${cc}|${sc}`
      const list = idx.get(key)
      if (list) list.push(name)
      else idx.set(key, [name])
    }
    return idx
  })
  cityPromise.catch(() => {
    cityPromise = undefined
  })
  return cityPromise
}
