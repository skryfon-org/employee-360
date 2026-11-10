import type { ReactNode, Ref, SelectHTMLAttributes } from 'react'

const SELECT_CLS =
  'h-11 w-full rounded-sm border border-slate-300 bg-white px-3 text-base text-slate-900 focus:border-slate-900 focus:outline-none focus:ring-1 focus:ring-slate-900 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 aria-[invalid=true]:border-red-600 sm:text-sm md:h-9'

interface SelectFieldProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label: string
  /** Shows a red asterisk after the label and sets aria-required on the select. */
  required?: boolean
  error?: string
  children: ReactNode
  ref?: Ref<HTMLSelectElement>
}

export function SelectField({ label, required, error, id, name, ref, children, ...rest }: SelectFieldProps) {
  const selectId = id ?? name
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={selectId} className="text-xs font-medium text-slate-900">
        {label}
        {required && (
          <span className="text-red-600" aria-hidden="true">
            {' '}
            *
          </span>
        )}
      </label>
      <select
        id={selectId}
        name={name}
        ref={ref}
        aria-required={required || undefined}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${selectId}-error` : undefined}
        className={SELECT_CLS}
        {...rest}
      >
        {children}
      </select>
      {error && (
        <p id={`${selectId}-error`} role="alert" className="text-xs text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}

export function Section({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-4">
      <div>
        <h2 className="text-base font-semibold text-slate-900">{title}</h2>
        {description && <p className="mt-1 text-xs text-slate-600">{description}</p>}
      </div>
      {children}
    </section>
  )
}

export const secondaryButtonCls =
  'h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500 md:h-9'
export const dangerButtonCls =
  'h-11 rounded-sm border border-red-300 bg-white px-4 text-sm font-medium text-red-600 hover:bg-red-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:text-slate-500 md:h-9'

/** Read-only label/value grid used by the view mode of each section. */
export function DefinitionList({
  rows,
  className = 'grid grid-cols-1 gap-4 sm:grid-cols-2',
}: {
  rows: [string, ReactNode][]
  className?: string
}) {
  return (
    <dl className={className}>
      {rows.map(([label, value]) => (
        <div key={label} className="flex flex-col gap-1">
          <dt className="text-xs font-medium text-slate-600">{label}</dt>
          <dd className="break-words text-sm text-slate-900">{value || '-'}</dd>
        </div>
      ))}
    </dl>
  )
}
