import { Combobox } from '../forms/Combobox'
import { FormField } from '../auth/components/FormField'
import { cascadePatch, type AddressField } from './cascade'
import { useAddressOptions } from './useAddressOptions'

export interface AddressFieldsValue {
  country: string
  state: string
  city: string
  pincode: string
}

export interface AddressFieldsProps {
  value: AddressFieldsValue
  /** Called with every field that changed (a country change also clears state and city). */
  onChange: (patch: Partial<AddressFieldsValue>) => void
  onBlur?: (field: keyof AddressFieldsValue) => void
  errors?: Partial<Record<keyof AddressFieldsValue, string>>
  required?: Partial<Record<keyof AddressFieldsValue, boolean>>
  disabled?: boolean
}

/**
 * Country / State / City dropdowns (cascading, backed by a lazily loaded bundled dataset) plus the postal code.
 * Renders four grid cells; place it inside the form's grid. Values are plain name strings, as the API expects.
 * Falls back to free-text inputs where the dataset has no states or cities for the selection.
 */
export function AddressFields({ value, onChange, onBlur, errors = {}, required = {}, disabled }: AddressFieldsProps) {
  const o = useAddressOptions(value)
  const change = (field: AddressField, v: string) => onChange(cascadePatch(field, v))

  const stateHint = !value.country.trim() ? 'Select a country first' : 'Select state'
  const cityHint = !value.state.trim() ? (o.stateMode === 'select' ? 'Select a state first' : 'City') : o.cityDisabled ? 'Loading cities…' : 'Select city'

  return (
    <>
      <Combobox
        id="country"
        label="Country"
        value={o.countryValue}
        options={o.countries}
        placeholder={o.loading ? 'Loading…' : 'Select country'}
        required={required.country}
        error={errors.country}
        disabled={disabled}
        onChange={(v) => change('country', v)}
        onBlur={() => onBlur?.('country')}
      />
      {o.stateMode === 'select' ? (
        <Combobox
          id="state"
          label="State"
          value={o.stateValue}
          options={o.states}
          placeholder={stateHint}
          required={required.state}
          error={errors.state}
          disabled={disabled || o.stateDisabled}
          onChange={(v) => change('state', v)}
          onBlur={() => onBlur?.('state')}
        />
      ) : (
        <FormField
          label="State"
          name="state"
          autoComplete="address-level1"
          required={required.state}
          error={errors.state}
          disabled={disabled}
          value={value.state}
          onChange={(e) => change('state', e.target.value)}
          onBlur={() => onBlur?.('state')}
        />
      )}
      {o.cityMode === 'select' ? (
        <Combobox
          id="city"
          label="City"
          value={o.cityValue}
          options={o.cities}
          placeholder={cityHint}
          required={required.city}
          error={errors.city}
          disabled={disabled || o.cityDisabled}
          onChange={(v) => change('city', v)}
          onBlur={() => onBlur?.('city')}
        />
      ) : (
        <FormField
          label="City"
          name="city"
          autoComplete="address-level2"
          required={required.city}
          error={errors.city}
          disabled={disabled}
          value={value.city}
          onChange={(e) => change('city', e.target.value)}
          onBlur={() => onBlur?.('city')}
        />
      )}
      <FormField
        label="Pincode"
        name="pincode"
        autoComplete="postal-code"
        required={required.pincode}
        error={errors.pincode}
        disabled={disabled}
        value={value.pincode}
        onChange={(e) => onChange({ pincode: e.target.value })}
        onBlur={() => onBlur?.('pincode')}
      />
    </>
  )
}
