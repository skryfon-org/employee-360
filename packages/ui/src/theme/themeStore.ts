import { create } from 'zustand'

export type ThemeMode = 'light' | 'dark'

/** Shared by both clients; the inline script in each index.html reads the same key before first paint. */
export const THEME_STORAGE_KEY = 'employee360.theme'

const DARK_QUERY = '(prefers-color-scheme: dark)'

const hasWindow = () => typeof window !== 'undefined'

function readSaved(): ThemeMode | null {
  if (!hasWindow()) return null
  try {
    const v = window.localStorage.getItem(THEME_STORAGE_KEY)
    return v === 'light' || v === 'dark' ? v : null
  } catch {
    return null
  }
}

function systemMode(): ThemeMode {
  if (!hasWindow() || typeof window.matchMedia !== 'function') return 'light'
  try {
    return window.matchMedia(DARK_QUERY).matches ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

/** Puts the `dark` class and `color-scheme` on <html>. */
export function applyTheme(mode: ThemeMode): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.classList.toggle('dark', mode === 'dark')
  root.style.colorScheme = mode
}

export function resolveInitialTheme(): ThemeMode {
  return readSaved() ?? systemMode()
}

interface ThemeState {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
  toggle: () => void
}

export const useThemeStore = create<ThemeState>()((set, get) => {
  const initial = resolveInitialTheme()
  applyTheme(initial)
  const setMode = (mode: ThemeMode) => {
    applyTheme(mode)
    if (hasWindow()) {
      try {
        window.localStorage.setItem(THEME_STORAGE_KEY, mode)
      } catch {
        /* storage unavailable: theme still applies for this session */
      }
    }
    set({ mode })
  }
  return { mode: initial, setMode, toggle: () => setMode(get().mode === 'dark' ? 'light' : 'dark') }
})

let stopWatching: (() => void) | null = null

/**
 * Follows live OS theme changes, but only while the user has no saved choice (an explicit
 * toggle always wins). Idempotent: any previous subscription is replaced, so repeated calls
 * (HMR, tests) never stack listeners. Returns an unsubscribe function; a no-op without a
 * DOM / matchMedia. Falls back to the legacy addListener API on old Safari.
 */
export function watchSystemTheme(): () => void {
  stopWatching?.()
  stopWatching = null
  if (!hasWindow() || typeof window.matchMedia !== 'function') return () => {}
  let mql: MediaQueryList
  try {
    mql = window.matchMedia(DARK_QUERY)
  } catch {
    return () => {}
  }
  const onChange = (e: { matches: boolean }) => {
    if (readSaved() !== null) return
    const mode: ThemeMode = e.matches ? 'dark' : 'light'
    applyTheme(mode)
    useThemeStore.setState({ mode })
  }
  const modern = typeof mql.addEventListener === 'function'
  if (modern) mql.addEventListener('change', onChange)
  else mql.addListener?.(onChange)
  const stop = () => {
    if (modern) mql.removeEventListener('change', onChange)
    else mql.removeListener?.(onChange)
    if (stopWatching === stop) stopWatching = null
  }
  stopWatching = stop
  return stop
}

watchSystemTheme()

// Vite HMR: the re-evaluated module starts with no handle, so release the old listener here.
// Typed locally because this package does not depend on vite/client types.
const hot = (import.meta as ImportMeta & { hot?: { dispose: (cb: () => void) => void } }).hot
hot?.dispose(() => stopWatching?.())
