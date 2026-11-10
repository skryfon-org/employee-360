import { useEffect } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getErrorMessage } from '@employee360/api-client'
import { InlineAlert, Skeleton, SkeletonRegion } from '@employee360/ui'
import { useMeQuery } from '../../features/profile/queries/profileQueries'
import { useProfileGateStore } from '../../stores/profileGateStore'

export const WELCOME_PATH = '/welcome'
export const COMPLETE_PROFILE_PATH = '/complete-profile'

/**
 * Route guard that sits inside RequireAuth. Users who must complete a profile (profile_required,
 * i.e. never admins) are redirected to the welcome screen (first login) or profile completion
 * until GET /me reports profile_completed. A 403 `profile_incomplete` from any API call raises
 * the same gate. Everyone else is kept off the welcome / completion pages.
 */
export function ProfileGate() {
  const { data: me, isPending, isError, error, refetch, isFetching } = useMeQuery()
  const flagged = useProfileGateStore((s) => s.profileIncomplete)
  const welcomeDismissed = useProfileGateStore((s) => s.welcomeDismissed)
  const setFlag = useProfileGateStore((s) => s.setProfileIncomplete)
  const { pathname } = useLocation()

  // Once a fresh /me confirms completion, a stale 403 flag must not keep the gate shut.
  useEffect(() => {
    if (flagged && me?.profile_completed && !isFetching) setFlag(false)
  }, [flagged, me?.profile_completed, isFetching, setFlag])

  if (isPending) {
    return (
      <div className="mx-auto w-full max-w-3xl p-6">
        <SkeletonRegion label="Loading your account…" className="flex flex-col gap-4">
          <Skeleton className="h-7 w-64" />
          <Skeleton className="h-40 w-full" />
        </SkeletonRegion>
      </div>
    )
  }
  if (isError || !me) {
    return (
      <div className="mx-auto flex w-full max-w-3xl flex-col items-start gap-3 p-6">
        <InlineAlert tone="error">{getErrorMessage(error, 'Could not load your account.')}</InlineAlert>
        <button
          type="button"
          onClick={() => void refetch()}
          className="h-11 rounded-sm border border-slate-300 bg-white px-4 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
        >
          Retry
        </button>
      </div>
    )
  }

  const gated = Boolean(me.profile_required) && (!me.profile_completed || flagged)
  if (!gated) {
    return pathname === WELCOME_PATH || pathname === COMPLETE_PROFILE_PATH ? <Navigate to="/" replace /> : <Outlet />
  }
  const target = me.first_login && !welcomeDismissed ? WELCOME_PATH : COMPLETE_PROFILE_PATH
  return pathname === target ? <Outlet /> : <Navigate to={target} replace />
}
