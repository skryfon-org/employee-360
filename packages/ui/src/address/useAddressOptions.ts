import { useEffect, useState } from 'react'
import { computeAddressOptions, findCountry, findState, type AddressOptions, type AddressValue } from './cascade'
import { loadCityIndex, loadCoreData, type CityIndex, type CoreData } from './dataset'

/** Lazy-loads the bundled dataset (no network at runtime beyond the app's own chunks) and derives dropdown options. */
export function useAddressOptions(value: AddressValue): AddressOptions & { loading: boolean } {
  const [core, setCore] = useState<CoreData | null>(null)
  const [cityIndex, setCityIndex] = useState<CityIndex | null>(null)
  const [failed, setFailed] = useState(false)

  // Countries are needed to render the dropdown at all, so load them right away.
  useEffect(() => {
    if (core) return
    let live = true
    loadCoreData().then(
      (c) => live && setCore(c),
      () => live && setFailed(true),
    )
    return () => {
      live = false
    }
  }, [core])

  const needCities = !!core && !!findState(core, findCountry(core, value.country), value.state)
  useEffect(() => {
    if (!needCities || cityIndex) return
    let live = true
    loadCityIndex().then(
      (c) => live && setCityIndex(c),
      () => live && setFailed(true),
    )
    return () => {
      live = false
    }
  }, [needCities, cityIndex])

  const opts = computeAddressOptions(core, cityIndex, value)
  // If the dataset failed to load, degrade to free text so the form stays usable.
  if (failed) return { ...opts, stateMode: 'text', cityMode: 'text', stateDisabled: false, cityDisabled: false, loading: false }
  return { ...opts, loading: !core }
}
