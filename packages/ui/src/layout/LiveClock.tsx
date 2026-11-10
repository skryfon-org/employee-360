import { useEffect, useState } from 'react'
import { formatClockParts } from './clock-utils'

export interface LiveClockProps {
  className?: string
}

/** Ticking clock in the user's locale: time on top, date underneath. Time only below `sm`. */
export function LiveClock({ className = '' }: LiveClockProps) {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(id)
  }, [])
  const { date, time } = formatClockParts(now)
  return (
    <time
      data-testid="live-clock"
      dateTime={now.toISOString()}
      className={`ml-auto flex shrink-0 flex-col items-end whitespace-nowrap leading-tight tabular-nums ${className}`}
    >
      <span className="text-sm font-semibold text-slate-900 sm:text-base">{time}</span>
      <span className="hidden text-xs text-slate-500 sm:block">{date}</span>
    </time>
  )
}
