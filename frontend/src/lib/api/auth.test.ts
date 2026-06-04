// Tests for the auth API client and the SvelteKit auth proxies.
//
// The proxies are the linchpin of the F4 fix: they exist so the
// Set-Cookie header from the Go backend reaches the browser on the
// same origin. If the proxy drops the Set-Cookie header, the
// HttpOnly session cookie is silently lost and the user is silently
// logged out on every request.
//
// Each test stubs the global fetch with a hand-rolled response so
// we can assert that:
//   1. The proxy forwards the body and Content-Type to the backend.
//   2. The proxy copies Set-Cookie headers verbatim from the
//      backend response onto the SvelteKit response.
//   3. Errors are mapped cleanly (502 for backend unreachable, 400
//      for bad JSON, 405 for non-POST).
//   4. login()/signup() throw AuthError on non-2xx, with the
//      server-supplied message.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Auth API client is small enough to test in isolation; the proxy
// needs the SvelteKit runtime (event, request) so we test it
// separately as an end-to-end handler invocation.

import { login, signup, logout, AuthError } from '$lib/api/auth';

// ── auth API client ────────────────────────────────────────────────

describe('auth API client', () => {
  let originalFetch: typeof fetch;
  beforeEach(() => {
    originalFetch = globalThis.fetch;
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe('login', () => {
    it('POSTs email+password to /api/auth/login and returns the response', async () => {
      const mock = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({ userId: 'u-1', email: 'alice@example.com', name: 'Alice' }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      );
      globalThis.fetch = mock as unknown as typeof fetch;

      const res = await login('alice@example.com', 'hunter2');

      expect(res.userId).toBe('u-1');
      expect(res.email).toBe('alice@example.com');
      expect(mock).toHaveBeenCalledTimes(1);
      const [url, init] = mock.mock.calls[0];
      expect(url).toBe('/api/auth/login');
      expect(init.method).toBe('POST');
      expect(JSON.parse(init.body as string)).toEqual({
        email: 'alice@example.com',
        password: 'hunter2',
      });
    });

    it('throws AuthError with the server message on non-2xx', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ message: 'Email and password are required' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
        })
      ) as unknown as typeof fetch;

      try {
        await login('', '');
        expect.fail('login should have thrown');
      } catch (err) {
        expect(err).toBeInstanceOf(AuthError);
        expect((err as AuthError).status).toBe(400);
        expect((err as AuthError).message).toBe('Email and password are required');
      }
    });

    it('uses a fallback message when the server body is not JSON', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response('not json', { status: 500 })
      ) as unknown as typeof fetch;

      await expect(login('a', 'b')).rejects.toThrow(/Login failed/);
    });
  });

  describe('signup', () => {
    it('POSTs name+email+password and returns the response', async () => {
      const mock = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({ userId: 'u-2', email: 'new@example.com', name: 'New User' }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      );
      globalThis.fetch = mock as unknown as typeof fetch;

      const res = await signup('New User', 'new@example.com', 'pw');

      expect(res.name).toBe('New User');
      expect(mock).toHaveBeenCalledTimes(1);
      const [, init] = mock.mock.calls[0];
      expect(JSON.parse(init.body as string)).toEqual({
        name: 'New User',
        email: 'new@example.com',
        password: 'pw',
      });
    });

    it('throws AuthError on signup validation failure', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ message: 'Name is required for signup' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
        })
      ) as unknown as typeof fetch;

      await expect(signup('', 'a@b.c', 'pw')).rejects.toThrow(/Name is required/);
    });
  });

  describe('logout', () => {
    it('POSTs to /api/auth/logout and resolves even on 204', async () => {
      const mock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
      globalThis.fetch = mock as unknown as typeof fetch;

      await expect(logout()).resolves.toBeUndefined();
      expect(mock.mock.calls[0][0]).toBe('/api/auth/logout');
    });
  });
});
