import type { HTMLAttributes, ReactNode } from 'react'

/** Centred empty state for lists: bold title, muted description, optional action (e.g. Reset button). */
export function EmptyState({
  title,
  description,
  action,
  className = '',
  ...rest
}: {
  title: ReactNode
  description?: ReactNode
  action?: ReactNode
  className?: string
} & Omit<HTMLAttributes<HTMLDivElement>, 'className' | 'title'>) {
  return (
    <div {...rest} className={`flex flex-col items-center gap-1 px-4 py-8 text-center ${className}`.trim()}>
      <p className="text-sm font-medium text-slate-900">{title}</p>
      {description && <p className="text-sm text-slate-600">{description}</p>}
      {action}
    </div>
  )
}
