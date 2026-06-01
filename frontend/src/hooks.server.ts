import { env } from '$env/dynamic/private';
import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
  const path = event.url.pathname;
  const isApiRoute = path.startsWith('/api/');
  if (isApiRoute) {
    console.error(`[hooks] API request: ${event.request.method} ${path}, session_id=${event.cookies.get('session_id')}, ffAppShell=${env.FF_ENABLE_APP_SHELL}`);
  }

  const ffEnableAppShell = env.FF_ENABLE_APP_SHELL === 'true';
  const publicRoutes = ['/', '/login', '/signup', '/404'];
  const isProtected = !publicRoutes.includes(event.route.id ?? '');
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
