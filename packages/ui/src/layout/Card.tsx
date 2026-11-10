import type { ReactNode } from 'react'

export function Card({
  children,
  className = '',
  padded = true,
  ...rest
}: {
  children: ReactNode
  className?: string
  padded?: boolean
} & Omit<React.HTMLAttributes<HTMLDivElement>, 'className' | 'children'>) {
  return (
    <div {...rest} className={`rounded-sm border border-line bg-surface ${padded ? 'p-4 sm:p-6' : ''} ${className}`}>
      {children}
    </div>
  )
}
