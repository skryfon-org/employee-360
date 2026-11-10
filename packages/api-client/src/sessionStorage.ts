import type { Session, SessionStorageAdapter } from './session.ts';

export const DEFAULT_SESSION_STORAGE_KEY = 'employee360.session';

function getStorage(): Storage | null {
  try {
    if (typeof window === 'undefined' || !window.localStorage) return null;
    return window.localStorage;
  } catch {
    return null; // access can throw (blocked site data, sandboxed iframe)
  }
}

function parseSession(raw: string | null): Session | null {
  if (!raw) return null;
  try {
    const v: unknown = JSON.parse(raw);
    if (typeof v !== 'object' || v === null) return null;
    const s = v as Record<string, unknown>;
    if (typeof s.accessToken !== 'string' || typeof s.refreshToken !== 'string') return null;
    if (!s.accessToken || !s.refreshToken) return null;
    return {
      accessToken: s.accessToken,
      refreshToken: s.refreshToken,
      tenantId: typeof s.tenantId === 'string' ? s.tenantId : null,
      expiresAt: typeof s.expiresAt === 'string' ? s.expiresAt : undefined,
    };
  } catch {
    return null;
  }
}

/**
 * localStorage-backed {@link SessionStorageAdapter}. Safe when storage is
 * unavailable (SSR, blocked, quota): every operation degrades to a no-op.
 * Installed by default at package load (see session.ts); use a custom key or
 * re-install with `setSessionStorage(createLocalStorageSessionAdapter(key))`.
 * Also listens to the `storage` event so rotation/logout in another tab
 * propagates to this tab's in-memory session.
 */
export function createLocalStorageSessionAdapter(
  key: string = DEFAULT_SESSION_STORAGE_KEY,
): SessionStorageAdapter {
  return {
    load() {
      try {
        return parseSession(getStorage()?.getItem(key) ?? null);
      } catch {
        return null;
      }
    },
    save(session) {
      try {
        getStorage()?.setItem(key, JSON.stringify(session));
      } catch {
        // quota exceeded / storage disabled: stay in-memory only
      }
    },
    clear() {
      try {
        getStorage()?.removeItem(key);
      } catch {
        // ignore
      }
    },
    subscribe(onExternalChange) {
      if (typeof window === 'undefined') return () => {};
      const handler = (e: StorageEvent): void => {
        // key === null means localStorage.clear() was called in another tab.
        if (e.key !== null && e.key !== key) return;
        onExternalChange(e.key === null ? null : parseSession(e.newValue));
      };
      window.addEventListener('storage', handler);
      return () => window.removeEventListener('storage', handler);
    },
  };
}
