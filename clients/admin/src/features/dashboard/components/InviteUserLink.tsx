import { Link } from 'react-router-dom'

export function InviteUserLink() {
  return (
    <Link
      to="/invitations/new"
      className="inline-flex h-11 items-center rounded-sm bg-slate-900 px-4 text-sm font-semibold text-white hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 md:h-9"
    >
      Invite user
    </Link>
  )
}
