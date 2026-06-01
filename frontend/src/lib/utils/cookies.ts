/**
 * Cookie parsing utilities for SvelteKit server routes.
 */

export function getCookie(request: Request, name: string): string {
  const cookieHeader = request.headers.get('cookie') ?? '';
  const match = cookieHeader.split(';').find((c) => {
    const [k] = c.trim().split('=');
    return k === name;
  });
  return match ? decodeURIComponent(match.split('=')[1]) : '';
}