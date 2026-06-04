// Tests for the auth store. The store is the F4 fix's "control
// surface" — it's the only thing the rest of the app touches to
// learn the auth state. Critical invariants:
//
//   1. The store NEVER reads or writes document.cookie. (It cannot
//      read the HttpOnly session cookie anyway, and writing it
//      would be a no-op for HttpOnly.) The store tracks only
//      client-side display state.
//   2. loginAndStore() calls the /api/auth/login proxy and stores
//      the returned user info.
//   3. logout() always clears local state, even if the server-side
//      logout request fails.
//   4. setCurrentUser(null) flips isAuthenticated to false; setting
//      a user flips it to true.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

// Mock the auth API client so we don't actually call fetch.
// The store's contract is "call the API, then update state" —
// we just need to verify the wiring.
vi.mock('$lib/api/auth', () => ({
  login: vi.fn(),
  signup: vi.fn(),
  logout: vi.fn(),
}));

import { login as apiLogin, signup as apiSignup, logout as apiLogout } from '$lib/api/auth';
import {
  isAuthenticated,
  currentUser,
  setCurrentUser,
  login,
  loginAndStore,
  signupAndStore,
  logout as storeLogout,
  snapshot,
} from '$lib/stores/auth';

describe('auth store', () => {
  beforeEach(() => {
    // Reset stores to a known starting state.
    setCurrentUser(null);
    vi.clearAllMocks();
  });
  afterEach(() => {
    setCurrentUser(null);
  });

  it('starts unauthenticated with no current user', () => {
    expect(get(isAuthenticated)).toBe(false);
    expect(get(currentUser)).toBeNull();
  });

  it('setCurrentUser(user) flips isAuthenticated to true', () => {
    setCurrentUser({ name: 'Alice', email: 'a@example.com' });
    expect(get(isAuthenticated)).toBe(true);
    expect(get(currentUser)).toEqual({ name: 'Alice', email: 'a@example.com' });
  });

  it('setCurrentUser(null) flips isAuthenticated to false', () => {
    setCurrentUser({ name: 'Alice', email: 'a@example.com' });
    setCurrentUser(null);
    expect(get(isAuthenticated)).toBe(false);
    expect(get(currentUser)).toBeNull();
  });

  it('login(name, email) sets the display state synchronously', () => {
    login('Alice', 'a@example.com');
    expect(get(currentUser)).toEqual({ name: 'Alice', email: 'a@example.com' });
    expect(get(isAuthenticated)).toBe(true);
  });

  it('loginAndStore calls the API and stores the returned user', async () => {
    vi.mocked(apiLogin).mockResolvedValue({
      userId: 'u-1',
      email: 'a@example.com',
      name: 'Alice',
    });

    await loginAndStore('a@example.com', 'hunter2');

    expect(apiLogin).toHaveBeenCalledWith('a@example.com', 'hunter2');
    expect(get(currentUser)).toEqual({ name: 'Alice', email: 'a@example.com' });
    expect(get(isAuthenticated)).toBe(true);
  });

  it('loginAndStore falls back to email-as-name when API returns no name', async () => {
    vi.mocked(apiLogin).mockResolvedValue({
      userId: 'u-1',
      email: 'a@example.com',
    });

    await loginAndStore('a@example.com', 'pw');

    expect(get(currentUser)?.name).toBe('a@example.com');
  });

  it('signupAndStore calls signup and stores the user', async () => {
    vi.mocked(apiSignup).mockResolvedValue({
      userId: 'u-2',
      email: 'new@example.com',
      name: 'New User',
    });

    await signupAndStore('New User', 'new@example.com', 'pw');

    expect(apiSignup).toHaveBeenCalledWith('New User', 'new@example.com', 'pw');
    expect(get(currentUser)).toEqual({ name: 'New User', email: 'new@example.com' });
  });

  it('storeLogout clears state and calls the API', async () => {
    setCurrentUser({ name: 'Alice', email: 'a@example.com' });
    vi.mocked(apiLogout).mockResolvedValue();

    await storeLogout();

    expect(apiLogout).toHaveBeenCalled();
    expect(get(isAuthenticated)).toBe(false);
    expect(get(currentUser)).toBeNull();
  });

  it('storeLogout clears local state even when the API call fails', async () => {
    setCurrentUser({ name: 'Alice', email: 'a@example.com' });
    vi.mocked(apiLogout).mockRejectedValue(new Error('network down'));

    // The store is supposed to be defensive: the user clicked
    // "log out" and expects to be logged out, regardless of
    // whether the backend is reachable. The HttpOnly cookie will
    // persist until the browser closes, but the in-memory state
    // must clear so the UI re-renders as logged-out.
    await storeLogout();

    expect(get(isAuthenticated)).toBe(false);
    expect(get(currentUser)).toBeNull();
  });

  it('snapshot returns the current state for tests', () => {
    setCurrentUser({ name: 'Alice', email: 'a@example.com' });
    const s = snapshot();
    expect(s.currentUser).toEqual({ name: 'Alice', email: 'a@example.com' });
    expect(s.isAuthenticated).toBe(true);
  });

  it('NEVER touches document.cookie — the F4 invariant', async () => {
    // If any code path in the store reads or writes
    // document.cookie, that is the F4 bug. We assert it stays
    // untouched across the full lifecycle.
    const cookieSpy = vi.fn();
    const originalDescriptor = Object.getOwnPropertyDescriptor(document, 'cookie');
    Object.defineProperty(document, 'cookie', {
      get: () => {
        cookieSpy('get');
        return '';
      },
      set: () => {
        cookieSpy('set');
      },
      configurable: true,
    });

    try {
      vi.mocked(apiLogin).mockResolvedValue({
        userId: 'u-1',
        email: 'a@example.com',
        name: 'Alice',
      });
      await loginAndStore('a@example.com', 'pw');
      setCurrentUser(null);
      vi.mocked(apiLogout).mockResolvedValue();
      await storeLogout();

      expect(cookieSpy).not.toHaveBeenCalled();
    } finally {
      if (originalDescriptor) {
        Object.defineProperty(document, 'cookie', originalDescriptor);
      }
    }
  });
});
