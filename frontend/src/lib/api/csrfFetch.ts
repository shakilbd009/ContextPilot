/**
 * CSRF-aware fetch helper for browser callers.
 *
 * Reads the `csrf_token` cookie (set by hooks.server.ts on first GET)
 * and echoes its value in the `X-CSRF-Token` header on every
 * state-changing request. The double-submit pattern is enforced by
 * the SvelteKit hooks layer.
 */

const CSRF_COOKIE_NAME = 'csrf_token';
const CSRF_HEADER_NAME = 'X-CSRF-Token';
const STATE_CHANGING_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

/**
 * Read a single cookie by name from document.cookie. Returns '' when
 * the cookie is absent. Intended for browser-only use.
 */
function readCookie(name: string): string {
	if (typeof document === 'undefined') return '';
	const cookies = document.cookie ? document.cookie.split(';') : [];
	for (const c of cookies) {
		const idx = c.indexOf('=');
		const k = idx >= 0 ? c.slice(0, idx).trim() : c.trim();
		if (k === name) {
			const v = idx >= 0 ? c.slice(idx + 1) : '';
			try {
				return decodeURIComponent(v);
			} catch {
				return v;
			}
		}
	}
	return '';
}

/**
 * Issue a CSRF-aware fetch. For state-changing methods the CSRF token
 * cookie value is echoed in the `X-CSRF-Token` header. The first GET
 * to any /api/* path triggers the hooks layer to set the cookie; on
 * subsequent state-changing requests the cookie is present and we
 * echo it.
 */
export function csrfFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
	const method = (init?.method ?? 'GET').toUpperCase();
	const headers = new Headers(init?.headers ?? {});
	if (STATE_CHANGING_METHODS.has(method)) {
		const token = readCookie(CSRF_COOKIE_NAME);
		if (token) {
			headers.set(CSRF_HEADER_NAME, token);
		}
	}
	return fetch(input, { ...init, headers });
}
