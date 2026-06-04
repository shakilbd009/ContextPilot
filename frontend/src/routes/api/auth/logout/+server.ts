// POST /api/auth/logout — SvelteKit proxy that forwards the logout
// call to the Go backend and copies its Set-Cookie clear headers
// back to the browser. Used by the frontend auth store to clear
// the session before redirecting to /login.

import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

const SERVER_URL = () => env.SERVER_URL ?? 'http://localhost:3000';

export const POST: RequestHandler = async () => {
  let res: Response;
  try {
    res = await fetch(`${SERVER_URL()}/api/v1/auth/logout`, {
      method: 'POST',
    });
  } catch {
    return new Response(null, { status: 502 });
  }

  const out = new Response(res.body, {
    status: res.status,
    statusText: res.statusText,
  });

  const skip = new Set(['content-length', 'content-encoding', 'transfer-encoding']);
  for (const [k, v] of res.headers.entries()) {
    if (skip.has(k.toLowerCase())) continue;
    out.headers.set(k, v);
  }
  // Forward the expiring Set-Cookie headers so the browser removes
  // the session_id and X-User-ID cookies.
  for (const c of res.headers.getSetCookie()) {
    out.headers.append('Set-Cookie', c);
  }
  return out;
};

export const GET: RequestHandler = async () =>
  new Response('POST only', { status: 405 });
