import { useId, useLayoutEffect, useRef, useState } from 'react'
import { parseNotes } from '../notesFormat'

// jsdom has no layout, so also treat long or multi-line notes as overflowing.
const FALLBACK_LENGTH = 100

function NotesBody({ text }: { text: string }) {
  return (
    <>
      {parseNotes(text).map((b, i) => {
        if (b.type === 'ul') {
          return (
            <ul key={i} className="list-disc pl-5">
              {b.items.map((item, j) => (
                <li key={j}>{item}</li>
              ))}
            </ul>
          )
        }
        if (b.type === 'ol') {
          return (
            <ol key={i} className="list-decimal pl-5">
              {b.items.map((item, j) => (
                <li key={j}>{item}</li>
              ))}
            </ol>
          )
        }
        return (
          <p key={i} className="min-h-[1lh]">
            {b.text}
          </p>
        )
      })}
    </>
  )
}

export function ExpandableNotes({ notes }: { notes?: string | null }) {
  const [expanded, setExpanded] = useState(false)
  const [overflowing, setOverflowing] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const id = useId()
  const text = notes ?? ''
  const heuristic = text.length > FALLBACK_LENGTH || text.includes('\n')
  const hasList = parseNotes(text).some((b) => b.type !== 'text')

  useLayoutEffect(() => {
    const el = ref.current
    if (!el || expanded) return
    const measure = () => setOverflowing(el.scrollHeight > el.clientHeight)
    measure()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(measure)
    observer.observe(el)
    return () => observer.disconnect()
  }, [text, expanded])

  if (!text) return <>-</>

  const showToggle = expanded || overflowing || heuristic
  const cls = `break-words ${expanded ? '' : 'line-clamp-2'}`

  return (
    <div>
      {hasList ? (
        <div ref={ref} id={id} className={cls}>
          <NotesBody text={text} />
        </div>
      ) : (
        <div ref={ref} id={id} className={`whitespace-pre-line ${cls}`}>
          {text}
        </div>
      )}
      {showToggle && (
        <button
          type="button"
          aria-expanded={expanded}
          aria-controls={id}
          onClick={() => setExpanded((v) => !v)}
          className="mt-1 text-xs font-medium text-slate-900 hover:text-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2"
        >
          {expanded ? 'View less' : 'View more'}
        </button>
      )}
    </div>
  )
}
