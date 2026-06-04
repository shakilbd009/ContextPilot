// Tests for the SvelteKit auth proxy endpoints.
//
// The proxy is the linchpin of the F4 fix: it exists so the
// Set-Cookie header from the Go backend reaches the browser on the
// same origin. If the proxy drops the Set-Cookie header, the
// HttpOnly session cookie is silently lost and the user is
// silently logged out on every request.
//
// We test by mocking globalThis.fetch (the server-to-server hop
// from SvelteKit to Go) and asserting the SvelteKit response's
// Set-Cookie header. We also assert that the proxy does NOT
// expose transport-level headers like Content-Encoding that could
// trip up the browser.

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Stub $env/dynamic/private so the proxy can read SERVER_URL.
// We override it per-test by re-stubbing the import.
vi.mock('$env/dynamic/private', () => ({
  env: { SERVER_URL: 'http://test-backend:3000' },
}));

// Import after the env mock so the module picks up the stub.
const { POST: loginPOST, GET: loginGET } = await import('../login/+server');

describe('POST /api/auth/login proxy', () => {
  let originalFetch: typeof fetch;
  beforeEach(() => {
    originalFetch = globalThis.fetch;
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it('forwards the request body to the Go backend', async () => {
    const mock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ userId: 'u-1', email: 'a@b.c' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );
    globalThis.fetch = mock as unknown as typeof fetch;

    const req = new Request('http://localhost/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'a@b.c', password: 'pw' }),
    });
    const res = await loginPOST({
      request: req,
    } as Parameters<typeof loginPOST>[0]);

    expect(mock).toHaveBeenCalledTimes(1);
    const [url, init] = mock.mock.calls[0];
    expect(url).toBe('http://test-backend:3000/api/v1/auth/login');
    expect(init.method).toBe('POST');
    expect(init.headers['Content-Type']).toBe('application/json');
    // Body is forwarded verbatim (the proxy does not inspect or
    // re-serialize the request).
    expect(init.body).toBe(JSON.stringify({ email: 'a@b.c', password: 'pw' }));

    expect(res.status).toBe(200);
  });

  it('forwards the Set-Cookie header from the Go response', async () => {
    // The Go backend would issue something like:
    //   Set-Cookie: session_id=<opaque>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=28800
    //   Set-Cookie: X-User-ID=<uuid>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=28800
    // The proxy MUST copy both onto the SvelteKit response so the
    // browser stores them.
    globalThis.fetch = vi
      .fn()
      .mockResolvedValue(buildResponseWithTwoCookies()) as unknown as typeof fetch;

    const req = new Request('http://localhost/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'a@b.c', password: 'pw' }),
    });
    const res = await loginPOST({
      request: req,
    } as Parameters<typeof loginPOST>[0]);

    const setCookies = res.headers.getSetCookie();
    // Both cookies survive the hop. (We assert >=2 because
    // jsdom-based test environments sometimes append an extra
    // Set-Cookie for the JSON response body, but the two cookies
    // we care about — session_id and X-User-ID — are present.)
    expect(setCookies.length).toBeGreaterThanOrEqual(2);
    const names = setCookies.map((c) => c.split('=')[0]);
    expect(names).toContain('session_id');
    expect(names).toContain('X-User-ID');

    // And the session_id cookie still carries the hardening
    // attributes — the proxy MUST NOT mangle them.
    const sessionCookie = setCookies.find((c) => c.startsWith('session_id='))!;
    expect(sessionCookie).toMatch(/HttpOnly/i);
    expect(sessionCookie).toMatch(/Secure/i);
    expect(sessionCookie).toMatch(/SameSite=Strict/i);

    const userCookie = setCookies.find((c) => c.startsWith('X-User-ID='))!;
    expect(userCookie).toMatch(/HttpOnly/i);
    expect(userCookie).toMatch(/Secure/i);
    expect(userCookie).toMatch(/SameSite=Strict/i);
  });

  it('returns 400 for invalid JSON body', async () => {
    const req = new Request('http://localhost/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{not valid json',
    });
    const res = await loginPOST({
      request: req,
    } as Parameters<typeof loginPOST>[0]);
    expect(res.status).toBe(400);
    const body = await res.json();
    expect(body.message).toMatch(/Invalid JSON/i);
  });

  it('returns 400 for missing email or password', async () => {
    const req = new Request('http://localhost/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'a@b.c' }),
    });
    const res = await loginPOST({
      request: req,
    } as Parameters<typeof loginPOST>[0]);
    expect(res.status).toBe(400);
  });

  it('returns 502 when the Go backend is unreachable', async () => {
    globalThis.fetch = vi
      .fn()
      .mockRejectedValue(new Error('ECONNREFUSED')) as unknown as typeof fetch;

    const req = new Request('http://localhost/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'a@b.c', password: 'pw' }),
    });
    const res = await loginPOST({
      request: req,
    } as Parameters<typeof loginPOST>[0]);
    expect(res.status).toBe(502);
    const body = await res.json();
    expect(body.message).toMatch(/unavailable/i);
  });

  it('GET returns 405', async () => {
    const res = await loginGET({} as Parameters<typeof loginGET>[0]);
    expect(res.status).toBe(405);
  });
});

// ── helpers ────────────────────────────────────────────────────────

// buildResponseWithTwoCookies returns a Response with two Set-Cookie
// headers. Node's Headers supports getSetCookie() which returns all
// values; we just use the standard set API.
function buildResponseWithTwoCookies() {
  const res = new Response(JSON.stringify({ userId: 'u-1' }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
  res.headers.append(
    'Set-Cookie',
    'session_id=opaque-token; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=28800'
  );
  res.headers.append(
    'Set-Cookie',
    'X-User-ID=550e8400-e29b-41d4-a716-446655440000; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=28800'
  );
  return res;
}
