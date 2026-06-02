/**
 * CSRF (CWE-352) defense for SvelteKit /api/* routes.
 *
 * The defense has two layers:
 *   1. Origin/Referer allowlist — applied to any state-changing method
 *      (POST, PUT, PATCH, DELETE) in hooks.server.ts.
 *   2. Double-submit CSRF token — a per-session token issued as a
 *      JS-readable cookie on first safe-method (GET) request, and
 *      echoed in an `X-CSRF-Token` header on every state-changing
 *      request. The token is bound to the session and validated in
 *      hooks.server.ts.
 *
 * This is the SvelteKit-layer half of the CSRF baseline. The Go chi
 * router applies the same Origin/Referer allowlist as defense-in-depth
 * (see backend/internal/middleware/csrf.go).
 */

/**
 * The set of HTTP methods considered "state-changing" — same as the
 * Go middleware default.
 */
export const STATE_CHANGING_METHODS = ['POST', 'PUT', 'PATCH', 'DELETE'] as const;
export type StateChangingMethod = (typeof STATE_CHANGING_METHODS)[number];

/** Cookie name used to carry the CSRF token. Readable by JS (not HttpOnly). */
export const CSRF_COOKIE_NAME = 'csrf_token';

/** Header name the client must echo the cookie value in. */
export const CSRF_HEADER_NAME = 'x-csrf-token';

/**
 * Returns the set of allowed origins, parsed from CSRF_ALLOWED_ORIGINS
 * (comma-separated env var). Falls back to the local-dev default of
 * {http://localhost:5173, http://localhost:8080} when unset. Empty
 * entries and whitespace are skipped.
 */
export function getAllowedOrigins(envValue: string | undefined): string[] {
	const fallback = ['http://localhost:5173', 'http://localhost:8080'];
	const raw = envValue && envValue.trim() !== '' ? envValue : fallback.join(',');
	const out: string[] = [];
	for (const part of raw.split(',')) {
		const trimmed = part.trim();
		if (trimmed === '') continue;
		out.push(trimmed);
	}
	return out;
}

/**
 * True if `method` is one of POST/PUT/PATCH/DELETE (case-insensitive,
 * whitespace-trimmed).
 */
export function isStateChanging(method: string): method is StateChangingMethod {
	const upper = method.trim().toUpperCase();
	return STATE_CHANGING_METHODS.includes(upper as StateChangingMethod);
}

/**
 * Returns the effective origin of the request — prefers the Origin
 * header, falls back to the origin derived from the Referer header.
 * Returns null when neither header is present or the Referer is not a
 * valid URL. Exported for testing.
 */
export function requestOrigin(request: Request): string | null {
	const origin = request.headers.get('origin');
	if (origin && origin.trim() !== '') return origin.trim();
	const referer = request.headers.get('referer');
	if (referer && referer.trim() !== '') {
		try {
			const u = new URL(referer);
			return `${u.protocol}//${u.host}`;
		} catch {
			return null;
		}
	}
	return null;
}

/**
 * True if `origin` is an exact, full match against one of `allowed`.
 * Exact match (not substring, not prefix) prevents bypass via
 * "http://allowed.com.evil.com" or similar tricks.
 */
export function isAllowedOrigin(origin: string, allowed: string[]): boolean {
	return allowed.includes(origin);
}

/**
 * Generates a fresh CSRF token. Uses crypto.randomUUID() for an
 * unguessable, URL-safe value. The token has no expiry on its own;
 * rotation is driven by session lifecycle (cookie expiry / logout).
 */
export function generateCSRFToken(): string {
	// crypto.randomUUID is available in Node 19+ and in the SvelteKit
	// request-context environment via the global `crypto` symbol.
	return crypto.randomUUID();
}

/**
 * Constant-time string comparison to thwart timing oracles. Returns
 * false immediately on length mismatch.
 */
export function safeEqual(a: string, b: string): boolean {
	if (a.length !== b.length) return false;
	let mismatch = 0;
	for (let i = 0; i < a.length; i++) {
		mismatch |= a.charCodeAt(i) ^ b.charCodeAt(i);
	}
	return mismatch === 0;
}
