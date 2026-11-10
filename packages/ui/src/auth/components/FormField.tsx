import type { InputHTMLAttributes, Ref } from 'react'

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label: string
  /** Shows a red asterisk after the label and sets aria-required on the input. */
  required?: boolean
  error?: string
  /** Helper text shown under the input; replaced by the error when present. */
  hint?: string
  ref?: Ref<HTMLInputElement>
}

export function FormField({ label, required, error, hint, id, ref, ...rest }: Props) {
  const inputId = id ?? rest.name
  const errId = `${inputId}-error`
  const hintId = `${inputId}-hint`
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={inputId} className="text-xs font-medium text-slate-900">
        {label}
        {required && (
          <span className="text-red-600" aria-hidden="true">
            {' '}
            *
          </span>
        )}
      </label>
      <input
        id={inputId}
        ref={ref}
        aria-required={required || undefined}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errId : hint ? hintId : undefined}
        className="h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base sm:text-sm md:h-9 text-slate-900 placeholder:text-slate-500 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600"
        {...rest}
      />
      {error ? (
        <p id={errId} role="alert" className="text-xs text-red-700">
          {error}
        </p>
      ) : (
        hint && (
          <p id={hintId} className="text-xs text-ink-muted">
            {hint}
          </p>
        )
      )}
    </div>
  )
}
