import type { HTMLAttributes, ReactNode } from 'react'

/** Flat neutral placeholder block. Decorative: hidden from assistive tech (the owning SkeletonRegion announces loading). */
export function Skeleton({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`animate-pulse motion-reduce:animate-none rounded-sm bg-slate-200 ${className}`} />
}

/** A stack of text-line placeholders; the last line is shorter. */
export function SkeletonText({ lines = 2, className = '' }: { lines?: number; className?: string }) {
  return (
    <div aria-hidden="true" className={`flex flex-col gap-2 ${className}`}>
      {Array.from({ length: lines }, (_, i) => (
        <Skeleton key={i} className={`h-3 ${i === lines - 1 && lines > 1 ? 'w-2/3' : 'w-full'}`} />
      ))}
    </div>
  )
}

/**
 * Owns a loading region: role="status" + aria-busy with visually-hidden text, so screen readers
 * (and tests) have something to find while the Skeleton blocks inside stay aria-hidden.
 */
export function SkeletonRegion({
  label = 'Loading…',
  children,
  className = '',
  ...rest
}: { label?: string; children: ReactNode; className?: string } & Omit<HTMLAttributes<HTMLDivElement>, 'className' | 'children' | 'role'>) {
  return (
    <div {...rest} role="status" aria-busy="true" className={className}>
      <span className="sr-only">{label}</span>
      {children}
    </div>
  )
}
