export interface NotesState {
  text: string
  selectionStart: number
  selectionEnd: number
}

export type ListKind = 'bullet' | 'numbered'

const BULLET = /^- /
const NUMBERED = /^\d+\. /

function markerLength(line: string, kind: ListKind): number {
  const m = (kind === 'bullet' ? BULLET : NUMBERED).exec(line)
  return m ? m[0].length : 0
}

function anyMarkerLength(line: string): number {
  return markerLength(line, 'bullet') || markerLength(line, 'numbered')
}

/** Start/end offsets of the lines touched by the selection. */
function lineRange(text: string, start: number, end: number): { from: number; to: number } {
  const from = text.lastIndexOf('\n', start - 1) + 1
  // A selection ending right after a newline does not include the next line.
  const effectiveEnd = end > start && text[end - 1] === '\n' ? end - 1 : end
  const nl = text.indexOf('\n', effectiveEnd)
  return { from, to: nl === -1 ? text.length : nl }
}

/** Toggle / convert the selected lines (or the current line) to a list. */
export function toggleList(text: string, selectionStart: number, selectionEnd: number, kind: ListKind): NotesState {
  const { from, to } = lineRange(text, selectionStart, selectionEnd)
  const lines = text.slice(from, to).split('\n')
  const allHave = lines.every((l) => markerLength(l, kind) > 0)

  let n = 0
  const out: string[] = []
  const deltas: number[] = []
  for (const line of lines) {
    const old = anyMarkerLength(line)
    const body = line.slice(old)
    let next: string
    if (allHave) {
      next = body
    } else if (kind === 'bullet') {
      next = `- ${body}`
    } else {
      n += 1
      next = `${n}. ${body}`
    }
    out.push(next)
    deltas.push(next.length - line.length)
  }

  const replaced = out.join('\n')
  const newText = text.slice(0, from) + replaced + text.slice(to)

  const adjust = (pos: number, isEnd: boolean): number => {
    let offset = from
    let delta = 0
    for (let i = 0; i < lines.length; i++) {
      const lineStart = offset
      const lineEnd = offset + lines[i].length
      const oldMarker = anyMarkerLength(lines[i])
      const newMarker = anyMarkerLength(out[i])
      if (pos > lineEnd) {
        delta += deltas[i]
      } else if (pos >= lineStart) {
        const col = pos - lineStart
        const inBody = Math.max(col - oldMarker, 0)
        const base = lineStart + delta
        return Math.min(base + newMarker + inBody, base + out[i].length)
      }
      offset = lineEnd + 1
    }
    return pos + delta + (isEnd ? 0 : 0)
  }

  return { text: newText, selectionStart: adjust(selectionStart, false), selectionEnd: adjust(selectionEnd, true) }
}

/**
 * Handle Enter at a collapsed caret. Returns null when the default newline
 * behaviour should apply (not on a list line, or a selection exists).
 */
export function continueList(text: string, selectionStart: number, selectionEnd: number): NotesState | null {
  if (selectionStart !== selectionEnd) return null
  const lineStart = text.lastIndexOf('\n', selectionStart - 1) + 1
  const nl = text.indexOf('\n', selectionStart)
  const lineEnd = nl === -1 ? text.length : nl
  const line = text.slice(lineStart, lineEnd)

  const bullet = markerLength(line, 'bullet')
  const numbered = markerLength(line, 'numbered')
  const marker = bullet || numbered
  if (!marker || selectionStart < lineStart + marker) return null

  // Empty item: remove the marker and end the list.
  if (line.length === marker) {
    const newText = text.slice(0, lineStart) + text.slice(lineEnd)
    return { text: newText, selectionStart: lineStart, selectionEnd: lineStart }
  }

  const nextMarker = bullet ? '- ' : `${parseInt(line, 10) + 1}. `
  const insert = `\n${nextMarker}`
  const newText = text.slice(0, selectionStart) + insert + text.slice(selectionStart)
  const caret = selectionStart + insert.length
  return { text: newText, selectionStart: caret, selectionEnd: caret }
}

export type NotesBlock = { type: 'text'; text: string } | { type: 'ul'; items: string[] } | { type: 'ol'; items: string[] }

/** Group lines into plain text lines and consecutive list runs. */
export function parseNotes(text: string): NotesBlock[] {
  const blocks: NotesBlock[] = []
  for (const line of text.split('\n')) {
    const type: 'ul' | 'ol' | null = BULLET.test(line) ? 'ul' : NUMBERED.test(line) ? 'ol' : null
    if (!type) {
      blocks.push({ type: 'text', text: line })
      continue
    }
    const item = line.slice(anyMarkerLength(line))
    const last = blocks[blocks.length - 1]
    if (last && last.type === type) last.items.push(item)
    else blocks.push(type === 'ul' ? { type, items: [item] } : { type, items: [item] })
  }
  return blocks
}
