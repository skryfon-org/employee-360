import type { StatusLogEntry } from './queries/statusLogQueries'

/** Select value (`project:<id>` / `category:<name>`) for an existing entry. */
export const targetOf = (e: StatusLogEntry) => (e.project_id ? `project:${e.project_id}` : `category:${e.activity_category ?? ''}`)

export const humanize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

export const entryName = (e: StatusLogEntry) =>
  e.project_id ? `${e.project_name ?? 'Project'}${e.project_code ? ` (${e.project_code})` : ''}` : humanize(e.activity_category ?? 'other')
