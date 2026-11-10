import { forwardRef, useId } from 'react'

export interface SwitchProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label: string
  description?: string
  id?: string
  name?: string
  disabled?: boolean
  onBlur?: () => void
}

/** Accessible toggle switch. Native button gives Space/Enter keyboard operation. RHF-friendly via Controller. */
export const Switch = forwardRef<HTMLButtonElement, SwitchProps>(function Switch(
  { checked, onChange, label, description, id, name, disabled, onBlur },
  ref,
) {
  const autoId = useId()
  const baseId = id ?? autoId
  const labelId = `${baseId}-label`
  const descId = `${baseId}-desc`
  return (
    <div className="flex items-start gap-3">
      <button
        ref={ref}
        id={baseId}
        name={name}
        type="button"
        role="switch"
        aria-checked={checked}
        aria-labelledby={labelId}
        aria-describedby={description ? descId : undefined}
        disabled={disabled}
        onBlur={onBlur}
        onClick={() => onChange(!checked)}
        className={`relative mt-0.5 inline-flex h-5 w-9 shrink-0 items-center rounded-full border border-transparent focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-200 ${
          checked ? 'bg-slate-900' : 'bg-slate-400'
        }`}
      >
        <span
          aria-hidden="true"
          data-switch-thumb
          className={`inline-block h-4 w-4 rounded-full bg-white ${checked ? 'translate-x-4' : 'translate-x-0.5'}`}
        />
      </button>
      <div className="flex flex-col text-xs">
        <span id={labelId} onClick={() => !disabled && onChange(!checked)} className="cursor-pointer font-medium text-slate-900">
          {label}
        </span>
        {description && (
          <span id={descId} className="font-normal text-slate-600">
            {description}
          </span>
        )}
      </div>
    </div>
  )
})
