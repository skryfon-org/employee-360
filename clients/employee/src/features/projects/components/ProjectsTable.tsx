import type { UserAllocation } from '../queries/projectQueries'

export function ProjectsTable({ rows }: { rows: UserAllocation[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[36rem] text-left">
        <thead className="bg-slate-100 text-xs font-medium text-slate-600">
          <tr>
            <th scope="col" className="px-4 py-3">Project</th>
            <th scope="col" className="px-4 py-3">Allocation</th>
            <th scope="col" className="px-4 py-3">Role</th>
            <th scope="col" className="px-4 py-3">Start</th>
            <th scope="col" className="px-4 py-3">End</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((a) => (
            <tr key={a.id} className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50">
              <td className="px-4 py-3 font-medium">
                {a.project_name}
                <span className="ml-2 text-xs font-normal text-slate-600">{a.project_code}</span>
              </td>
              <td className="px-4 py-3">{a.allocation_percent}%</td>
              <td className="px-4 py-3 text-slate-600">{a.role_on_project || '-'}</td>
              <td className="px-4 py-3">{a.start_date}</td>
              <td className="px-4 py-3">{a.end_date ?? 'Ongoing'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
