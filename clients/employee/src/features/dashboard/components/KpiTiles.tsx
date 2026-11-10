import type { SVGProps } from 'react'
import { StatCards, type Stat } from '@employee360/ui'
import type { EmployeeDashboard } from '@employee360/api-client'

function Icon(props: SVGProps<SVGSVGElement>) {
  return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} strokeLinecap="round" strokeLinejoin="round" {...props} />
}
const ClockIcon = (p: SVGProps<SVGSVGElement>) => (<Icon {...p}><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></Icon>)
const CalendarIcon = (p: SVGProps<SVGSVGElement>) => (<Icon {...p}><rect x="3" y="5" width="18" height="16" rx="2" /><path d="M3 10h18M8 3v4M16 3v4" /></Icon>)
const BriefcaseIcon = (p: SVGProps<SVGSVGElement>) => (<Icon {...p}><rect x="3" y="7" width="18" height="13" rx="2" /><path d="M9 7V4h6v3" /></Icon>)
const CheckIcon = (p: SVGProps<SVGSVGElement>) => (<Icon {...p}><circle cx="12" cy="12" r="9" /><path d="m8 12 3 3 5-6" /></Icon>)

export function KpiTiles({ stats }: { stats?: EmployeeDashboard['stats'] }) {
  const tiles: Stat[] = [
    { label: 'Pending leave', value: stats?.pending_leave ?? 0, description: 'Requests awaiting a decision', icon: ClockIcon, to: '/leave' },
    { label: 'Upcoming leave days', value: stats?.upcoming_leave_days ?? 0, description: 'Approved, not yet taken', icon: CalendarIcon, to: '/leave' },
    { label: 'Allocation', value: `${stats?.allocation_percent ?? 0}%`, description: 'Across active projects today', icon: BriefcaseIcon, to: '/projects' },
    { label: 'Hours, last 7 days', value: stats?.hours_this_week ?? 0, description: 'Logged over the last 7 days', icon: ClockIcon, to: '/status' },
  ]
  // null/undefined = not an approver: the tile is hidden rather than shown as 0.
  if (stats?.pending_approvals != null) {
    tiles.push({ label: 'Pending team approvals', value: stats.pending_approvals, description: 'Leave requests to review', icon: CheckIcon, to: '/leave' })
  }
  return <StatCards label="Summary" stats={tiles} />
}
