// POST /api/auth/login — SvelteKit proxy to the Go auth endpoint.
//
// The browser cannot talk to the Go backend directly because the
// session_id cookie that the backend issues is HttpOnly + Secure +
// SameSite=Strict. The SvelteKit dev server and the Go backend are
// different origins from the browser's perspective, so a direct
// cross-origin fetch would either (a) fail CORS for the Set-Cookie
// response, or (b) silently drop Set-Cookie if the browser permits
// it. By routing the login through a same-origin SvelteKit handler
// we guarantee:
//
//   1. The browser sends the POST to its own origin (no CORS
//      dance, no preflight).
//   2. The SvelteKit handler issues a server-to-server fetch to the
//      Go backend (no cookie needed; the backend is in the same
//      trust boundary).
//   3. The Set-Cookie header from the Go response is copied verbatim
//      onto the SvelteKit response that the browser sees — same
//      origin, so the browser accepts it.
//
// This is the production path for F4 from t_793ea842. The client
// code (login/+page.svelte) never reads or writes cookies directly;
// it just calls fetch('/api/auth/login', ...) and lets the
// HttpOnly cookie do its job.

import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const SERVER_URL = () => env.SERVER_URL ?? 'http://localhost:3000';

export const POST: RequestHandler = async ({ request }) => {
  // Buffer the body once. We may need to read it twice (validation
  // + forward).
  const body = await request.text();

  let parsed: { email?: string; password?: string };
  try {
    parsed = JSON.parse(body);
  } catch {
    return jsonResponse({ message: 'Invalid JSON body' }, 400);
  }
  if (!parsed.email || !parsed.password) {
    return jsonResponse({ message: 'Email and password are required' }, 400);
  }

  let res: Response;
  try {
    res = await fetch(`${SERVER_URL()}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    });
  } catch (err) {
    return jsonResponse(
      { message: 'Auth backend unavailable' },
      502
    );
  }

  // Build the SvelteKit response by copying the Go response body and
  // most headers — but explicitly forward Set-Cookie so the
  // HttpOnly session cookie actually reaches the browser.
  const out = new Response(res.body, {
    status: res.status,
    statusText: res.statusText,
  });

  // Copy headers, EXCEPT those that would re-encode or re-apply
  // transport-level security we already control (Content-Length is
  // recomputed by the runtime; Content-Encoding is for the
  // server-to-server hop and must not leak to the browser).
  const skip = new Set(['content-length', 'content-encoding', 'transfer-encoding']);
  for (const [k, v] of res.headers.entries()) {
    if (skip.has(k.toLowerCase())) continue;
    out.headers.set(k, v);
  }
  // Critical: forward every Set-Cookie header. SvelteKit's Response
  // supports headers.append so multiple Set-Cookie headers
  // (session_id + X-User-ID) survive the round-trip.
  const setCookies = res.headers.getSetCookie();
  for (const c of setCookies) {
    out.headers.append('Set-Cookie', c);
  }
  return out;
};

// Method not allowed for non-POST.
export const GET: RequestHandler = async () =>
  jsonResponse({ message: 'POST only' }, 405);

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
