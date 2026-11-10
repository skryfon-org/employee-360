// Static datasets shipped by `country-state-city`, imported lazily as separate chunks.
declare module 'country-state-city/lib/assets/country.json' {
  const countries: { name: string; isoCode: string }[]
  export default countries
}
declare module 'country-state-city/lib/assets/state.json' {
  const states: { name: string; isoCode: string; countryCode: string }[]
  export default states
}
declare module 'country-state-city/lib/assets/city.json' {
  /** Tuples of [name, countryCode, stateCode, latitude, longitude]. */
  const cities: [string, string, string, string, string][]
  export default cities
}
