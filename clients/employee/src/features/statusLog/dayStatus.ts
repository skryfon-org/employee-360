import type { GithubComSkryfonOrgEmployee360BackendInternalTypesStatuslogMyDayStatusResponse as DayStatusItem } from '@employee360/api-client'

export type { DayStatusItem }

export interface DayStatusInfo {
  /** Short badge text. */
  label: string
  /** Longer tooltip / accessible description. */
  title: string
  /** `info` = nothing to log and not a problem; `warning` = the day still needs a log. */
  tone: 'info' | 'warning'
  /** Note for the selected-day panel. */
  note: string
}

/** Why a day has no log. Returns undefined for logged days, truly missing days, and when the status is unknown. */
export function describeDayStatus(s?: DayStatusItem): DayStatusInfo | undefined {
  if (!s) return undefined
  switch (s.day_status) {
    case 'holiday': {
      const name = s.holiday_name
      return {
        label: 'Holiday',
        title: name ? `Public holiday: ${name}` : 'Public holiday',
        tone: 'info',
        note: name ? `Public holiday: ${name} — no log needed.` : 'Public holiday — no log needed.',
      }
    }
    case 'weekly_off':
      return { label: 'Weekly off', title: 'Weekly off', tone: 'info', note: 'Weekly off — no log needed.' }
    case 'on_leave': {
      const type = s.leave_type_name
      return {
        label: 'On leave',
        title: type ? `On leave: ${type}` : 'On leave',
        tone: 'info',
        note: type ? `On approved leave (${type}) — no log needed.` : 'On approved leave — no log needed.',
      }
    }
    case 'missing':
      if (s.half_day) {
        const type = s.leave_type_name
        return {
          label: 'Missing',
          title: type ? `Half-day leave (${type}): nothing logged` : 'Half-day leave: nothing logged',
          tone: 'warning',
          note: 'Half-day leave — please log your other half.',
        }
      }
      return undefined
    default:
      return undefined
  }
}
