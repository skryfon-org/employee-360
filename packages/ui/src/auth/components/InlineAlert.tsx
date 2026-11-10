import type { ReactNode } from 'react'

export function InlineAlert({
  tone,
  children,
}: {
  tone: 'error' | 'neutral'
  children: ReactNode
}) {
  const cls =
    tone === 'error'
      ? 'bg-red-50 border-red-200 text-red-700'
      : 'bg-slate-100 border-slate-300 text-slate-900'
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={`flex gap-2 rounded-sm border p-3 text-sm ${cls}`}
    >
      <svg className="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        {tone === 'error' ? (
          <path d="M10 2a8 8 0 100 16 8 8 0 000-16zm-1 4h2v5H9V6zm0 6h2v2H9v-2z" />
        ) : (
          <path d="M10 2a8 8 0 100 16 8 8 0 000-16zm-1 3h2v2H9V5zm0 4h2v6H9V9z" />
        )}
      </svg>
      <p>{children}</p>
    </div>
  )
}
