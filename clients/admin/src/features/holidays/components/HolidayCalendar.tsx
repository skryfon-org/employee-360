import { HolidayCalendar as SharedHolidayCalendar, type CalendarCursor, type HolidayCalendarProps } from '@employee360/ui'
import type { Holiday } from '../queries/holidayQueries'
import { useHolidayViewStore } from '../../../stores/holidayViewStore'

export type { CalendarCursor }

type Props = Omit<HolidayCalendarProps, 'scope' | 'onScopeChange' | 'holidays' | 'onSelect' | 'onDateClick'> & {
  holidays: Holiday[]
  onSelect: (h: Holiday) => void
  onDateClick: (isoDate: string) => void
}

/** Admin wrapper: wires the shared calendar to the admin view store and enables editing interactions. */
export function HolidayCalendar({ holidays, onSelect, ...rest }: Props) {
  const scope = useHolidayViewStore((s) => s.calendarScope)
  const setScope = useHolidayViewStore((s) => s.setCalendarScope)
  const byId = new Map(holidays.map((h) => [h.id, h]))
  return (
    <SharedHolidayCalendar
      {...rest}
      holidays={holidays}
      scope={scope}
      onScopeChange={setScope}
      onSelect={(h) => {
        const holiday = byId.get(h.id)
        if (holiday) onSelect(holiday)
      }}
    />
  )
}
