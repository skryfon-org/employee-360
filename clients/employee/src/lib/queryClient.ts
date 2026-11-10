import { MutationCache, QueryCache, QueryClient, type DefaultOptions } from '@tanstack/react-query'
import { getErrorCode } from '@employee360/api-client'
import { useProfileGateStore } from '../stores/profileGateStore'

/** Error code the backend returns (HTTP 403) while the employee's mandatory profile is incomplete. */
export const PROFILE_INCOMPLETE = 'profile_incomplete'

/** Root of the /me query keys; invalidating it refetches the gating status. */
export const ME_KEY = ['me'] as const

export function createQueryClient(defaultOptions?: DefaultOptions): QueryClient {
  const handle = (err: unknown) => {
    if (getErrorCode(err) !== PROFILE_INCOMPLETE) return
    // Gate immediately, then let /me confirm the real status.
    useProfileGateStore.getState().setProfileIncomplete(true)
    void client.invalidateQueries({ queryKey: ME_KEY })
  }
  const client: QueryClient = new QueryClient({
    queryCache: new QueryCache({ onError: handle }),
    mutationCache: new MutationCache({ onError: handle }),
    defaultOptions,
  })
  return client
}
