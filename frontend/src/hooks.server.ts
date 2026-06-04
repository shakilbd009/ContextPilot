import { env } from '$env/dynamic/private';
import type { Handle } from '@sveltejs/kit';
import { getAllowedOrigins, isAllowedOrigin, isStateChanging, requestOrigin } from '$lib/server/csrf';

export const PUBLIC_ROUTES = ['/', '/login', '/signup', '/404', '/api/auth/login', '/api/auth/signup'];

export function isPublicRoute(routeId: string | null | undefined): boolean {
  return PUBLIC_ROUTES.includes(routeId ?? '');
}

// Origins allowed to issue state-changing requests (POST/PUT/PATCH/DELETE).
// The Origin header is preferred; Referer is used as fallback when Origin
// is absent. Comparison is exact string match on the origin (scheme://host).
function csrfCheck(request: Request, origin: string | null): boolean {
  // Prefer Origin header; fall back to Referer-derived origin.
  const fromOrigin = origin ?? requestOrigin(request);

  if (!fromOrigin) return true; // let Go backend handle missing origin
  return isAllowedOrigin(fromOrigin, getAllowedOrigins(env.CSRF_ALLOWED_ORIGINS));
}

export const handle: Handle = async ({ event, resolve }) => {
  const path = event.url.pathname;
  const isApiRoute = path.startsWith('/api/');
  if (isApiRoute) {
    console.error(`[hooks] API request: ${event.request.method} ${path}, session_id=${event.cookies.get('session_id')}, ffAppShell=${env.FF_ENABLE_APP_SHELL}`);
  }

  // CSRF check (CWE-352) — apply Origin/Referer allowlist to state-changing
  // methods before any other processing. Only enforced when the SvelteKit dev
  // server is NOT in passthrough mode (passthrough omits Origin/Referer).
  const stateChanging = isStateChanging(event.request.method);
  if (isApiRoute && stateChanging) {
    const origin = event.request.headers.get('origin');
    if (!csrfCheck(event.request, origin)) {
      return new Response('CSRF: origin not allowed', { status: 403 });
    }
  }

  const ffEnableAppShell = env.FF_ENABLE_APP_SHELL === 'true';
  const isProtected = !isPublicRoute(event.route.id);
  const sessionCookie = event.cookies.get('session_id');
  const isAuthenticated = !!sessionCookie;

  if (isProtected && ffEnableAppShell && !isAuthenticated) {
    if (isApiRoute) {
      return new Response(null, { status: 401 });
    }
    const redirectTo = encodeURIComponent(event.url.pathname);
    return Response.redirect(
      new URL(`/login?redirectTo=${redirectTo}`, event.url.origin).toString(),
      302
    );
  }

  event.locals.ffEnableAppShell = ffEnableAppShell;
  event.locals.isAuthenticated = isAuthenticated;

  const response = await resolve(event);
  return response;
};
