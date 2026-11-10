import type { ReactNode } from 'react'
import { BrandMark } from '../../layout/BrandMark'

export function AuthLayout({ title, children, product = 'Employee360' }: { title: string; children: ReactNode; product?: string }) {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-6 bg-canvas px-4 py-8">
      <BrandMark name={product} />
      <main className="w-full max-w-sm rounded-sm border border-line bg-surface p-6 sm:p-8">
        <h1 className="mb-6 text-xl font-semibold text-ink">{title}</h1>
        {children}
      </main>
      <p className="text-xs text-ink-muted">Open-source employee platform</p>
    </div>
  )
}
