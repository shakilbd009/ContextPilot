// POST /api/auth/signup — SvelteKit proxy to the Go auth signup endpoint.
//
// See /api/auth/login/+server.ts for the rationale: the
// SvelteKit → Go hop exists so the Set-Cookie response from the
// Go server reaches the browser on the same origin. The browser
// would never accept a Set-Cookie from a cross-origin response
// for HttpOnly+Secure+SameSite=Strict cookies (no CORS-allow-
// credential grant covers this in a useful way for SvelteKit dev).

import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const SERVER_URL = () => env.SERVER_URL ?? 'http://localhost:3000';

export const POST: RequestHandler = async ({ request }) => {
  const body = await request.text();

  let parsed: { email?: string; password?: string; name?: string };
  try {
    parsed = JSON.parse(body);
  } catch {
    return jsonResponse({ message: 'Invalid JSON body' }, 400);
  }
  if (!parsed.email || !parsed.password || !parsed.name) {
    return jsonResponse(
      { message: 'Email, password, and name are required' },
      400
    );
  }

  let res: Response;
  try {
    res = await fetch(`${SERVER_URL()}/api/v1/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    });
  } catch {
    return jsonResponse({ message: 'Auth backend unavailable' }, 502);
  }

  const out = new Response(res.body, {
    status: res.status,
    statusText: res.statusText,
  });

  const skip = new Set(['content-length', 'content-encoding', 'transfer-encoding', 'set-cookie']);
  for (const [k, v] of res.headers.entries()) {
    if (skip.has(k.toLowerCase())) continue;
    out.headers.set(k, v);
  }
  // Same critical Set-Cookie forwarding as /login.
  for (const c of res.headers.getSetCookie()) {
    out.headers.append('Set-Cookie', c);
  }
  return out;
};

export const GET: RequestHandler = async () =>
  jsonResponse({ message: 'POST only' }, 405);

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
