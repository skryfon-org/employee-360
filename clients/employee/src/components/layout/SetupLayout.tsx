import { Outlet, useNavigate } from 'react-router-dom'
import { BrandMark } from '@employee360/ui'
import { useLogoutMutation } from '../../features/auth/queries/authMutations'

/** Minimal chrome for the welcome and profile-completion screens (no navigation to gated pages). */
export function SetupLayout() {
  const logout = useLogoutMutation()
  const navigate = useNavigate()
  return (
    <div className="flex min-h-dvh flex-col bg-slate-50">
      <header className="flex h-14 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 sm:px-6">
        <BrandMark name="Employee360" subtitle="Employee Portal" />
        <button
          type="button"
          onClick={() => logout.mutate(undefined, { onSettled: () => navigate('/login', { replace: true }) })}
          className="h-11 rounded-sm border border-slate-300 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
        >
          Sign out
        </button>
      </header>
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <Outlet />
      </main>
    </div>
  )
}
