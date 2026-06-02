/**
 * csrfFetch helper tests.
 *
 * Verifies the browser-side CSRF double-submit token flow: state-
 * changing requests get an `X-CSRF-Token` header carrying the value of
 * the `csrf_token` cookie; safe methods do not.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { csrfFetch } from './csrfFetch';

describe('csrfFetch', () => {
	let originalCookie: string;
	let fetchSpy: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		originalCookie = document.cookie;
		// Stub global.fetch and capture the init arg.
		fetchSpy = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }));
		(globalThis as any).fetch = fetchSpy;
	});

	afterEach(() => {
		// Restore document.cookie by clearing it.
		document.cookie = originalCookie.replace(/csrf_token=[^;]*/, '');
	});

	function setCsrfCookie(value: string) {
		document.cookie = `csrf_token=${value}; path=/`;
	}

	it('echoes the csrf_token cookie in the X-CSRF-Token header on POST', async () => {
		setCsrfCookie('test-token-aaa');
		await csrfFetch('/api/upcoming', { method: 'POST', body: '{}' });
		expect(fetchSpy).toHaveBeenCalledTimes(1);
		const init = fetchSpy.mock.calls[0][1] as RequestInit;
		const headers = init.headers as Headers;
		expect(headers.get('X-CSRF-Token')).toBe('test-token-aaa');
	});

	it('echoes the token on PUT/PATCH/DELETE', async () => {
		setCsrfCookie('test-token-bbb');
		for (const method of ['PUT', 'PATCH', 'DELETE'] as const) {
			fetchSpy.mockClear();
			await csrfFetch('/api/foo', { method });
			const headers = fetchSpy.mock.calls[0][1].headers as Headers;
			expect(headers.get('X-CSRF-Token')).toBe('test-token-bbb');
		}
	});

	it('does not add X-CSRF-Token to GET requests', async () => {
		setCsrfCookie('test-token-ccc');
		await csrfFetch('/api/foo');
		const init = fetchSpy.mock.calls[0][1] as RequestInit | undefined;
		if (init?.headers) {
			const headers = init.headers as Headers;
			expect(headers.get('X-CSRF-Token')).toBeNull();
		}
	});

	it('POSTs without the header when no cookie is set (token will be issued on next GET)', async () => {
		document.cookie = 'csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/';
		await csrfFetch('/api/upcoming', { method: 'POST' });
		const init = fetchSpy.mock.calls[0][1] as RequestInit;
		const headers = init.headers as Headers;
		expect(headers.get('X-CSRF-Token')).toBeNull();
	});

	it('preserves other headers the caller passed in', async () => {
		setCsrfCookie('test-token-ddd');
		await csrfFetch('/api/upcoming', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', 'X-User-ID': 'user-42' },
			body: '{}',
		});
		const headers = fetchSpy.mock.calls[0][1].headers as Headers;
		expect(headers.get('X-CSRF-Token')).toBe('test-token-ddd');
		expect(headers.get('Content-Type')).toBe('application/json');
		expect(headers.get('X-User-ID')).toBe('user-42');
	});
});
