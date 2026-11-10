/**
 * Auth session owned by this package. The Axios interceptors read from it;
 * `login`/`refresh` write to it. Apps mirror it into Zustand by subscribing
 * (see `subscribeSession`) — the tokens are NOT duplicated server state, just
 * the credentials the transport layer needs.
 *
 * Persistence is ON by default: at package load (browser only) a
 * localStorage adapter is installed and the in-memory session is hydrated
 * from it, including cross-tab `storage` event sync. Opt out with
 * `setSessionStorage(null)` (memory only) or override with
 * `setSessionStorage(customAdapter)`.
 *
 * Security tradeoff: the refresh token in localStorage is readable by any
 * script running on the origin (XSS). Mitigate with no untrusted HTML; apps
 * that cannot accept this should opt out. Consuming apps should also ship a
 * strict Content-Security-Policy (no inline/eval scripts, tight script-src).
 */
import { createLocalStorageSessionAdapter } from './sessionStorage.ts';

export interface Session {
  accessToken: string;
  refreshToken: string;
  /** Resolved once at login from the user's tenant; never switched mid-session. */
  tenantId: string | null;
  /** ISO timestamp from the backend, informational only. */
  expiresAt?: string;
}

export interface SessionStorageAdapter {
  load: () => Session | null;
  save: (session: Session) => void;
  clear: () => void;
  /**
   * Optional cross-tab hook. The adapter calls `onExternalChange` whenever
   * another context changed the persisted session (`null` = cleared). Returns
   * an unsubscribe function. Invoked by `setSessionStorage`.
   */
  subscribe?: (onExternalChange: (session: Session | null) => void) => () => void;
}

type Listener = (session: Session | null) => void;

let current: Session | null = null;
let storage: SessionStorageAdapter | null = null;
let unsubscribeStorage: (() => void) | null = null;
const listeners = new Set<Listener>();

function notify(): void {
  listeners.forEach((l) => l(current));
}

export function getSession(): Session | null {
  return current;
}

export function setSession(session: Session): void {
  current = session;
  storage?.save(session);
  notify();
}

export function clearSession(): void {
  current = null;
  storage?.clear();
  notify();
}

/** Subscribe to session changes. Returns an unsubscribe function. */
export function subscribeSession(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/**
 * Re-read the persisted session (source of truth across tabs). Falls back to
 * the in-memory session when no adapter is installed.
 */
export function readPersistedSession(): Session | null {
  return storage ? storage.load() : current;
}

/**
 * Adopt a session that was written elsewhere (another tab): update memory and
 * notify subscribers, but do NOT write back to storage.
 */
export function applyExternalSession(session: Session | null): void {
  current = session;
  notify();
}

/** Install (or remove) a persistence adapter and hydrate from it. */
export function setSessionStorage(adapter: SessionStorageAdapter | null): void {
  unsubscribeStorage?.();
  unsubscribeStorage = null;
  storage = adapter;
  if (adapter) {
    current = adapter.load();
    unsubscribeStorage = adapter.subscribe?.(applyExternalSession) ?? null;
    notify();
  }
}

// Default-on persistence. Runs at module evaluation, i.e. before any request
// can be made through the client (client.ts imports this module), so the
// first request already sees the hydrated session. Skipped without a window
// (SSR / node); tests in node are therefore unaffected.
if (typeof window !== 'undefined') {
  setSessionStorage(createLocalStorageSessionAdapter());
}
