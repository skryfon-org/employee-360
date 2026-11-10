import { Link, useNavigate } from 'react-router-dom'
import { EmptyState, RowActionsMenu } from '@employee360/ui'
import type { Project } from '../queries/projectQueries'
import { ProjectStatusBadge } from './ProjectStatusBadge'

export function ProjectsTable({ projects, filtered = false }: { projects: Project[]; filtered?: boolean }) {
  const navigate = useNavigate()

  if (projects.length === 0) {
    return (
      <EmptyState title={filtered ? 'No matching projects' : 'No projects yet'} description={filtered ? 'Try a different search or status filter.' : 'Create a project to start allocating employees.'} />
    )
  }

  return (
    <>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[46rem] text-left">
          <thead className="bg-slate-100 text-xs font-medium text-slate-600">
            <tr>
              <th scope="col" className="px-4 py-3">Project</th>
              <th scope="col" className="px-4 py-3">Code</th>
              <th scope="col" className="px-4 py-3">Start date</th>
              <th scope="col" className="px-4 py-3">End date</th>
              <th scope="col" className="px-4 py-3">Status</th>
              <th scope="col" className="px-4 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {projects.map((p) => (
              <tr key={p.id} className="border-t border-slate-200 text-sm text-slate-900 hover:bg-slate-50">
                <td className="px-4 py-3 font-medium">
                  <Link to={`/projects/${p.id}`} className="text-slate-900 hover:text-slate-700">
                    {p.name}
                  </Link>
                </td>
                <td className="px-4 py-3">{p.code}</td>
                <td className="px-4 py-3 text-slate-600">{p.start_date}</td>
                <td className="px-4 py-3 text-slate-600">{p.end_date ?? 'ongoing'}</td>
                <td className="px-4 py-3">
                  <ProjectStatusBadge status={p.status} />
                </td>
                <td className="px-4 py-3">
                  <div className="flex justify-end">
                    <RowActionsMenu
                      label={`project ${p.name}`}
                      actions={[
                        { label: 'View', onSelect: () => navigate(`/projects/${p.id}`) },
                        { label: 'Edit', onSelect: () => navigate(`/projects/${p.id}/edit`) },
                      ]}
                    />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}
