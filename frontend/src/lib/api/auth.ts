// Auth API client. Talks to the SvelteKit /api/auth/* proxies, which
// forward to the Go auth endpoint and copy the HttpOnly+Secure+
// SameSite=Strict Set-Cookie headers back to the browser.
//
// The client never reads or writes the session cookies directly —
// that was the F4 bug. The browser stores them automatically in
// response to the Set-Cookie header, and the SvelteKit hooks server
// reads them via event.cookies.get() for protected-route checks.

export interface AuthResponse {
  userId: string;
  name?: string;
  email: string;
}

export class AuthError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = 'AuthError';
    this.status = status;
  }
}

export async function login(
  email: string,
  password: string,
  fetchFn: typeof fetch = fetch
): Promise<AuthResponse> {
  const res = await fetchFn('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const detail = await res.json().catch(() => ({ message: 'Login failed' }));
    throw new AuthError(detail.message ?? 'Login failed', res.status);
  }
  return (await res.json()) as AuthResponse;
}

export async function signup(
  name: string,
  email: string,
  password: string,
  fetchFn: typeof fetch = fetch
): Promise<AuthResponse> {
  const res = await fetchFn('/api/auth/signup', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, email, password }),
  });
  if (!res.ok) {
    const detail = await res.json().catch(() => ({ message: 'Signup failed' }));
    throw new AuthError(detail.message ?? 'Signup failed', res.status);
  }
  return (await res.json()) as AuthResponse;
}

export async function logout(
  fetchFn: typeof fetch = fetch
): Promise<void> {
  await fetchFn('/api/auth/logout', { method: 'POST' });
}
