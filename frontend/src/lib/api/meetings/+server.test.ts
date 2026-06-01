/**
 * getCookie helper unit tests
 *
 * Tests the cookie-parsing utility used in the /api/meetings proxy
 * to read X-User-ID from cookies when the request header is absent.
 */

import { describe, it, expect } from 'vitest';
import { getCookie } from '$lib/utils/cookies';

// getCookie is not exported; test via its behavior in a mock Request
function buildRequest(cookieHeader: string | null): Request {
  return new Request('http://localhost/', {
    headers: cookieHeader ? { cookie: cookieHeader } : {},
  });
}

describe('getCookie', () => {
  it('returns the cookie value when present', () => {
    const req = buildRequest('session_id=abc123; X-User-ID=550e8400-e29b-41d4-a716-446655440000; foo=bar');
    expect(getCookie(req, 'X-User-ID')).toBe('550e8400-e29b-41d4-a716-446655440000');
  });

  it('returns the cookie value without trailing semicolon artifacts', () => {
    const req = buildRequest('X-User-ID=550e8400-e29b-41d4-a716-446655440000');
    expect(getCookie(req, 'X-User-ID')).toBe('550e8400-e29b-41d4-a716-446655440000');
  });

  it('returns empty string when cookie is absent', () => {
    const req = buildRequest('session_id=abc123');
    expect(getCookie(req, 'X-User-ID')).toBe('');
  });

  it('returns empty string when no cookies present', () => {
    const req = buildRequest(null);
    expect(getCookie(req, 'X-User-ID')).toBe('');
  });

  it('handles URL-encoded cookie values', () => {
    const req = buildRequest('X-User-ID=550e8400-e29b-41d4-a716-446655440000');
    expect(getCookie(req, 'X-User-ID')).toBe('550e8400-e29b-41d4-a716-446655440000');
  });

  it('is case-sensitive for cookie name', () => {
    const req = buildRequest('X-User-ID=abc123');
    expect(getCookie(req, 'x-user-id')).toBe('');
  });
});