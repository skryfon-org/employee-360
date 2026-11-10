import { useEffect, useState } from 'react'
import { Toaster } from 'sonner'

const base = 'flex w-full items-start gap-3 rounded-sm border border-l-4 py-3 pl-4 pr-8 text-sm'

/** Flat, design-system styled sonner toaster. Mount once at the app root. */
export function AppToaster() {
  // Phones: top-center keeps toasts clear of bottom content and thumb-reach actions.
  const [narrow, setNarrow] = useState(() => window.matchMedia?.('(max-width: 639px)').matches ?? false)
  useEffect(() => {
    const mq = window.matchMedia?.('(max-width: 639px)')
    if (!mq) return
    const on = () => setNarrow(mq.matches)
    mq.addEventListener('change', on)
    return () => mq.removeEventListener('change', on)
  }, [])
  return (
    <Toaster
      position={narrow ? 'top-center' : 'bottom-right'}
      mobileOffset={16}
      closeButton
      visibleToasts={5}
      toastOptions={{
        unstyled: true,
        classNames: {
          toast: `${base} bg-slate-100 border-slate-300 text-slate-900`,
          success: 'bg-green-50 border-green-300 border-l-green-600 text-green-700',
          error: 'bg-red-50 border-red-200 border-l-red-600 text-red-700',
          warning: 'bg-amber-50 border-amber-300 border-l-amber-500 text-amber-700',
          info: 'bg-slate-100 border-slate-300 border-l-slate-500 text-slate-900',
          title: 'text-sm font-medium',
          // sonner defaults the close button to the top-left; pin it to the top-right corner
          closeButton:
            '!left-auto !right-0 ![transform:translate(35%,-35%)] border border-slate-300 bg-white text-slate-900 rounded-full focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900',
        },
      }}
    />
  )
}
