/**
 * CSRF helper unit tests.
 *
 * Covers requestOrigin, isAllowedOrigin, getAllowedOrigins,
 * isStateChanging, generateCSRFToken, and safeEqual — the building
 * blocks the hooks layer relies on for CSRF defense-in-depth.
 */

import { describe, it, expect } from 'vitest';
import {
	generateCSRFToken,
	getAllowedOrigins,
	isAllowedOrigin,
	isStateChanging,
	requestOrigin,
	safeEqual,
} from './csrf';

describe('isStateChanging', () => {
	it('treats POST/PUT/PATCH/DELETE as state-changing', () => {
		expect(isStateChanging('POST')).toBe(true);
		expect(isStateChanging('PUT')).toBe(true);
		expect(isStateChanging('PATCH')).toBe(true);
		expect(isStateChanging('DELETE')).toBe(true);
	});
	it('treats GET/HEAD/OPTIONS as safe', () => {
		expect(isStateChanging('GET')).toBe(false);
		expect(isStateChanging('HEAD')).toBe(false);
		expect(isStateChanging('OPTIONS')).toBe(false);
	});
	it('is case-insensitive', () => {
		expect(isStateChanging('post')).toBe(true);
		expect(isStateChanging('  DELETE  ')).toBe(true);
	});
});

describe('requestOrigin', () => {
	function buildRequest(headers: Record<string, string>): Request {
		return new Request('http://localhost/', { headers });
	}

	it('prefers Origin header', () => {
		const r = buildRequest({
			origin: 'https://app.contextpilot.com',
			referer: 'https://evil.com/page',
		});
		expect(requestOrigin(r)).toBe('https://app.contextpilot.com');
	});

	it('falls back to Referer-derived origin when no Origin', () => {
		const r = buildRequest({ referer: 'https://app.contextpilot.com/some/page?q=1' });
		expect(requestOrigin(r)).toBe('https://app.contextpilot.com');
	});

	it('returns null when both headers are missing', () => {
		expect(requestOrigin(buildRequest({}))).toBeNull();
	});

	it('returns null when Referer is malformed', () => {
		expect(requestOrigin(buildRequest({ referer: '://not-a-url' }))).toBeNull();
	});

	it('treats empty/whitespace headers as missing', () => {
		expect(requestOrigin(buildRequest({ origin: '   ' }))).toBeNull();
		expect(requestOrigin(buildRequest({ referer: '   ' }))).toBeNull();
	});
});

describe('isAllowedOrigin', () => {
	const allowed = ['http://localhost:5173', 'http://localhost:8080'];

	it('matches exact origins', () => {
		expect(isAllowedOrigin('http://localhost:5173', allowed)).toBe(true);
		expect(isAllowedOrigin('http://localhost:8080', allowed)).toBe(true);
	});

	it('rejects substring/path/scheme confusion', () => {
		expect(isAllowedOrigin('http://localhost:5173.evil.com', allowed)).toBe(false);
		expect(isAllowedOrigin('http://localhost:5173/', allowed)).toBe(false);
		expect(isAllowedOrigin('http://localhost:5173/a', allowed)).toBe(false);
		expect(isAllowedOrigin('https://localhost:5173', allowed)).toBe(false);
		expect(isAllowedOrigin('localhost:5173', allowed)).toBe(false);
		expect(isAllowedOrigin('http://evil.com', allowed)).toBe(false);
	});

	it('rejects empty list (defense in depth)', () => {
		expect(isAllowedOrigin('http://anything', [])).toBe(false);
	});
});

describe('getAllowedOrigins', () => {
	it('parses comma-separated values, trims whitespace, drops empties', () => {
		expect(
			getAllowedOrigins('http://localhost:5173, http://localhost:8080 ,,https://app.contextpilot.com')
		).toEqual([
			'http://localhost:5173',
			'http://localhost:8080',
			'https://app.contextpilot.com',
		]);
	});

	it('falls back to local-dev defaults when env is undefined or empty', () => {
		expect(getAllowedOrigins(undefined)).toEqual([
			'http://localhost:5173',
			'http://localhost:8080',
		]);
		expect(getAllowedOrigins('')).toEqual(['http://localhost:5173', 'http://localhost:8080']);
		expect(getAllowedOrigins('   ')).toEqual(['http://localhost:5173', 'http://localhost:8080']);
	});
});

describe('generateCSRFToken', () => {
	it('produces a non-empty unique value', () => {
		const a = generateCSRFToken();
		const b = generateCSRFToken();
		expect(a.length).toBeGreaterThan(0);
		expect(a).not.toBe(b);
	});
});

describe('safeEqual', () => {
	it('returns true for identical strings', () => {
		expect(safeEqual('abc', 'abc')).toBe(true);
	});
	it('returns false for different strings', () => {
		expect(safeEqual('abc', 'abd')).toBe(false);
	});
	it('returns false for different lengths', () => {
		expect(safeEqual('abc', 'abcd')).toBe(false);
		expect(safeEqual('', 'a')).toBe(false);
	});
});
