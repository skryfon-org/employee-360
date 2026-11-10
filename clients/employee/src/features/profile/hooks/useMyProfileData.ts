import { useOutletContext } from 'react-router-dom'
import type { MyProfile } from '../queries/profileQueries'

/** The profile loaded once by ProfileLayout, shared with every section page through Outlet context. */
export function useMyProfileData() {
  return useOutletContext<MyProfile>()
}
