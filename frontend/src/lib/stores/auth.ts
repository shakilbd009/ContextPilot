// Auth store for client-side display state.
//
// F4 fix: the auth store NEVER reads or writes session cookies.
// The session_id and X-User-ID cookies are HttpOnly (set by the
// server), so document.cookie cannot see them anyway. Instead, the
// store only tracks the user-facing display state (display name,
// email) and an `isAuthenticated` flag that the SvelteKit hooks
// layer maintains from the server-side cookie. The /api/auth/logout
// helper above triggers the server-side cookie expiry; the store
// just clears its own in-memory display state.
import { writable, get } from 'svelte/store';
import { login as apiLogin, signup as apiSignup, logout as apiLogout } from '$lib/api/auth';

export const isAuthenticated = writable(false);
export const currentUser = writable<{ name: string; email: string } | null>(null);

export function setCurrentUser(user: { name: string; email: string } | null) {
  currentUser.set(user);
  isAuthenticated.set(user !== null);
}

export function login(name: string, email: string) {
  setCurrentUser({ name, email });
}

export async function loginAndStore(
  email: string,
  password: string
): Promise<void> {
  const res = await apiLogin(email, password);
  setCurrentUser({ name: res.name ?? email, email: res.email });
}

export async function signupAndStore(
  name: string,
  email: string,
  password: string
): Promise<void> {
  const res = await apiSignup(name, email, password);
  setCurrentUser({ name: res.name ?? name, email: res.email });
}

export async function logout(): Promise<void> {
  try {
    await apiLogout();
  } catch {
    // Even if the server-side logout fails (e.g. backend down),
    // we still clear local display state. The HttpOnly cookie
    // would persist until the browser closes, but the user is
    // navigated away regardless.
  }
  setCurrentUser(null);
  // Backward-compat: keep the synchronous clear for any existing
  // import sites that expect it. No document.cookie write —
  // the HttpOnly session cookie is server-managed.
  currentUser.set(null);
  isAuthenticated.set(false);
}

// Re-export the snapshot helper for tests that want to assert
// on the in-memory state.
export function snapshot() {
  return { currentUser: get(currentUser), isAuthenticated: get(isAuthenticated) };
}
