import { describe, expect, it } from 'vitest'
import { continueList, parseNotes, toggleList } from './notesFormat'

describe('toggleList', () => {
  it('applies a bullet to the current line with no selection', () => {
    const r = toggleList('one\ntwo', 5, 5, 'bullet')
    expect(r.text).toBe('one\n- two')
    expect(r.selectionStart).toBe(7)
  })

  it('toggles a bullet off when all lines have it', () => {
    const r = toggleList('- a\n- b', 0, 7, 'bullet')
    expect(r.text).toBe('a\nb')
    expect(r.selectionStart).toBe(0)
    expect(r.selectionEnd).toBe(3)
  })

  it('numbers multiple selected lines', () => {
    const r = toggleList('a\nb\nc', 0, 5, 'numbered')
    expect(r.text).toBe('1. a\n2. b\n3. c')
    expect(r.selectionEnd).toBe(r.text.length)
  })

  it('converts bullet to numbered and back', () => {
    const n = toggleList('- a\n- b', 0, 7, 'numbered')
    expect(n.text).toBe('1. a\n2. b')
    expect(toggleList(n.text, 0, n.text.length, 'bullet').text).toBe('- a\n- b')
  })

  it('adds markers to the remaining lines when only some have them', () => {
    expect(toggleList('- a\nb', 0, 5, 'bullet').text).toBe('- a\n- b')
  })

  it('only touches selected lines', () => {
    expect(toggleList('a\nb\nc', 2, 3, 'bullet').text).toBe('a\n- b\nc')
  })

  it('works on empty text', () => {
    const r = toggleList('', 0, 0, 'bullet')
    expect(r).toEqual({ text: '- ', selectionStart: 2, selectionEnd: 2 })
  })
})

describe('continueList', () => {
  it('continues a bullet', () => {
    const r = continueList('- a', 3, 3)
    expect(r).toEqual({ text: '- a\n- ', selectionStart: 6, selectionEnd: 6 })
  })

  it('continues a numbered list with the next number', () => {
    expect(continueList('1. a\n2. b', 9, 9)?.text).toBe('1. a\n2. b\n3. ')
  })

  it('splits mid-line', () => {
    expect(continueList('- ab', 3, 3)?.text).toBe('- a\n- b')
  })

  it('ends the list on an empty item', () => {
    expect(continueList('- a\n- ', 6, 6)).toEqual({ text: '- a\n', selectionStart: 4, selectionEnd: 4 })
    expect(continueList('1. ', 3, 3)?.text).toBe('')
  })

  it('returns null for non-list lines and selections', () => {
    expect(continueList('plain', 5, 5)).toBeNull()
    expect(continueList('- a', 0, 3)).toBeNull()
    expect(continueList('- a', 1, 1)).toBeNull()
  })
})

describe('parseNotes', () => {
  it('groups consecutive list lines', () => {
    expect(parseNotes('hi\n- a\n- b\n1. x\n2. y')).toEqual([
      { type: 'text', text: 'hi' },
      { type: 'ul', items: ['a', 'b'] },
      { type: 'ol', items: ['x', 'y'] },
    ])
  })
})
