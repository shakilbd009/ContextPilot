// Auth store for client-side auth state
import { writable } from 'svelte/store';

export const isAuthenticated = writable(false);
export const currentUser = writable<{ name: string; email: string } | null>(null);

export function login(name: string, email: string) {
  currentUser.set({ name, email });
  isAuthenticated.set(true);
}

export function logout() {
  currentUser.set(null);
  isAuthenticated.set(false);
  // Clear session cookie
  document.cookie = 'session_id=; Max-Age=0; path=/';
}