// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  clearSession,
  getSession,
  setSession,
  setSessionStorage,
  subscribeSession,
  type Session,
} from './session.ts';
import { createLocalStorageSessionAdapter } from './sessionStorage.ts';

const KEY = 'test.session';
const s1: Session = { accessToken: 'a1', refreshToken: 'r1', tenantId: 't1', expiresAt: 'x' };
const s2: Session = { accessToken: 'a2', refreshToken: 'r2', tenantId: 't1' };

function fireStorage(init: StorageEventInit): void {
  window.dispatchEvent(new StorageEvent('storage', init));
}

describe('createLocalStorageSessionAdapter', () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => {
    setSessionStorage(null);
    clearSession();
    vi.restoreAllMocks();
  });

  it('round-trips save/load/clear', () => {
    const a = createLocalStorageSessionAdapter(KEY);
    expect(a.load()).toBeNull();
    a.save(s1);
    expect(a.load()).toEqual(s1);
    a.clear();
    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it('returns null for corrupt or malformed JSON', () => {
    const a = createLocalStorageSessionAdapter(KEY);
    localStorage.setItem(KEY, '{nope');
    expect(a.load()).toBeNull();
    localStorage.setItem(KEY, JSON.stringify({ accessToken: 1 }));
    expect(a.load()).toBeNull();
  });

  it('swallows quota errors on save', () => {
    const a = createLocalStorageSessionAdapter(KEY);
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('full', 'QuotaExceededError');
    });
    expect(() => a.save(s1)).not.toThrow();
  });

  it('is a no-op when localStorage access throws', () => {
    const a = createLocalStorageSessionAdapter(KEY);
    vi.spyOn(window, 'localStorage', 'get').mockImplementation(() => {
      throw new Error('blocked');
    });
    expect(a.load()).toBeNull();
    expect(() => a.save(s1)).not.toThrow();
    expect(() => a.clear()).not.toThrow();
  });

  it('hydrates the in-memory session via setSessionStorage and persists writes', () => {
    localStorage.setItem(KEY, JSON.stringify(s1));
    setSessionStorage(createLocalStorageSessionAdapter(KEY));
    expect(getSession()).toEqual(s1);
    setSession(s2);
    expect(JSON.parse(localStorage.getItem(KEY) ?? '{}')).toEqual(s2);
  });

  describe('cross-tab storage event sync', () => {
    it('updates memory and notifies on rotation in another tab', () => {
      setSessionStorage(createLocalStorageSessionAdapter(KEY));
      const listener = vi.fn();
      subscribeSession(listener);
      fireStorage({ key: KEY, newValue: JSON.stringify(s2) });
      expect(getSession()).toEqual(s2);
      expect(listener).toHaveBeenLastCalledWith(s2);
    });

    it('clears memory on logout in another tab (removeItem and storage.clear)', () => {
      setSessionStorage(createLocalStorageSessionAdapter(KEY));
      setSession(s1);
      fireStorage({ key: KEY, newValue: null });
      expect(getSession()).toBeNull();
      setSession(s1);
      fireStorage({ key: null, newValue: null });
      expect(getSession()).toBeNull();
    });

    it('ignores other keys and does not write back to storage', () => {
      setSessionStorage(createLocalStorageSessionAdapter(KEY));
      setSession(s1);
      const spy = vi.spyOn(Storage.prototype, 'setItem');
      fireStorage({ key: 'other', newValue: JSON.stringify(s2) });
      expect(getSession()).toEqual(s1);
      fireStorage({ key: KEY, newValue: JSON.stringify(s2) });
      expect(spy).not.toHaveBeenCalled();
    });

    it('stops listening after the adapter is removed', () => {
      setSessionStorage(createLocalStorageSessionAdapter(KEY));
      setSessionStorage(null);
      setSession(s1);
      fireStorage({ key: KEY, newValue: JSON.stringify(s2) });
      expect(getSession()).toEqual(s1);
    });
  });
});
