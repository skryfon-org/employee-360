import type { HTMLAttributes, ReactNode } from 'react'

/**
 * Top strip of a list card that holds the list's filters. Place it as the first
 * child of `<Card padded={false}>`. Pass grid/flex layout through `className`.
 */
export function FilterBar({
  children,
  className = '',
  ...rest
}: { children: ReactNode; className?: string } & Omit<HTMLAttributes<HTMLElement>, 'className' | 'children'>) {
  return (
    <div {...rest} className={`border-b border-slate-200 p-4 ${className}`.trim()}>
      {children}
    </div>
  )
}
