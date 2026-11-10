import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { THEME_STORAGE_KEY, resolveInitialTheme, useThemeStore, watchSystemTheme } from './themeStore'
import { ThemeToggle } from './ThemeToggle'
import adminHtml from '../../../../clients/admin/index.html?raw'
import employeeHtml from '../../../../clients/employee/index.html?raw'

beforeEach(() => {
  window.localStorage.clear()
  useThemeStore.getState().setMode('light')
  window.localStorage.clear()
})

describe('theme store', () => {
  it('applies the dark class and persists the choice', () => {
    useThemeStore.getState().setMode('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark')
    useThemeStore.getState().toggle()
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
  })
})

function mockSystemDark(dark: boolean) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockImplementation((query: string) => ({
      matches: query === '(prefers-color-scheme: dark)' ? dark : false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  )
}

describe('system theme default', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('follows prefers-color-scheme when nothing is saved', () => {
    mockSystemDark(true)
    expect(resolveInitialTheme()).toBe('dark')
    mockSystemDark(false)
    expect(resolveInitialTheme()).toBe('light')
  })

  it('falls back to light when matchMedia is unavailable', () => {
    vi.stubGlobal('matchMedia', undefined)
    expect(resolveInitialTheme()).toBe('light')
  })

  it('lets a saved choice win over the system preference', () => {
    mockSystemDark(true)
    window.localStorage.setItem(THEME_STORAGE_KEY, 'light')
    expect(resolveInitialTheme()).toBe('light')
    mockSystemDark(false)
    window.localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    expect(resolveInitialTheme()).toBe('dark')
  })

  it('ignores an invalid saved value', () => {
    mockSystemDark(true)
    window.localStorage.setItem(THEME_STORAGE_KEY, 'purple')
    expect(resolveInitialTheme()).toBe('dark')
  })

  it.each([
    ['admin', adminHtml],
    ['employee', employeeHtml],
  ])('%s index.html inline script matches the store logic', (_app, html) => {
    const script = /<script>([\s\S]*?)<\/script>/.exec(html)?.[1] ?? ''
    expect(script).toContain(THEME_STORAGE_KEY)
    for (const [dark, saved, expected] of [
      [true, null, 'dark'],
      [false, null, 'light'],
      [true, 'light', 'light'],
      [false, 'dark', 'dark'],
    ] as const) {
      mockSystemDark(dark)
      window.localStorage.clear()
      if (saved) window.localStorage.setItem(THEME_STORAGE_KEY, saved)
      document.documentElement.classList.remove('dark')
      new Function(script)()
      expect(document.documentElement.classList.contains('dark')).toBe(expected === 'dark')
      expect(document.documentElement.style.colorScheme).toBe(expected)
    }
  })
})

describe('live system theme changes', () => {
  afterEach(() => vi.unstubAllGlobals())

  function mockListenable() {
    let listener: ((e: { matches: boolean }) => void) | undefined
    const removeEventListener = vi.fn()
    vi.stubGlobal(
      'matchMedia',
      vi.fn().mockImplementation(() => ({
        matches: false,
        addEventListener: (_: string, l: (e: { matches: boolean }) => void) => (listener = l),
        removeEventListener,
      })),
    )
    return { fire: (matches: boolean) => listener?.({ matches }), removeEventListener }
  }

  it('follows the system when the user has no saved choice', () => {
    const m = mockListenable()
    const stop = watchSystemTheme()
    m.fire(true)
    expect(useThemeStore.getState().mode).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    m.fire(false)
    expect(useThemeStore.getState().mode).toBe('light')
    stop()
    expect(m.removeEventListener).toHaveBeenCalled()
  })

  it('ignores system changes once the user has chosen', () => {
    const m = mockListenable()
    watchSystemTheme()
    useThemeStore.getState().setMode('light')
    m.fire(true)
    expect(useThemeStore.getState().mode).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('replaces the previous subscription instead of stacking listeners', () => {
    const add = vi.fn()
    const remove = vi.fn()
    vi.stubGlobal('matchMedia', vi.fn().mockImplementation(() => ({ matches: false, addEventListener: add, removeEventListener: remove })))
    watchSystemTheme()
    watchSystemTheme()
    expect(add).toHaveBeenCalledTimes(2)
    expect(remove).toHaveBeenCalledTimes(1)
  })

  it('falls back to addListener / removeListener on old Safari', () => {
    let listener: ((e: { matches: boolean }) => void) | undefined
    const removeListener = vi.fn()
    vi.stubGlobal(
      'matchMedia',
      vi.fn().mockImplementation(() => ({
        matches: false,
        addListener: (l: (e: { matches: boolean }) => void) => (listener = l),
        removeListener,
      })),
    )
    const stop = watchSystemTheme()
    listener?.({ matches: true })
    expect(useThemeStore.getState().mode).toBe('dark')
    stop()
    expect(removeListener).toHaveBeenCalled()
  })

  it('is a safe no-op without matchMedia', () => {
    vi.stubGlobal('matchMedia', undefined)
    expect(() => watchSystemTheme()()).not.toThrow()
  })
})

describe('ThemeToggle', () => {
  it('is an accessible button that flips the theme', async () => {
    render(<ThemeToggle />)
    await userEvent.click(screen.getByRole('button', { name: 'Switch to dark theme' }))
    expect(useThemeStore.getState().mode).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(screen.getByRole('button', { name: 'Switch to light theme' })).toBeInTheDocument()
  })
})
