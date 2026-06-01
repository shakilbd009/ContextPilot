/**
 * Unit tests for meetings/[id]/+page.server.ts
 * Verifies the server-side load function for meeting detail pages.
 *
 * Coverage:
 *  - early-exit (no cookie)         → { meeting: null }, fetch NOT called
 *  - early-exit (empty cookie)     → { meeting: null }, fetch NOT called
 *  - 404 from backend               → { meeting: null }, fetch called once
 *  - fetch rejects                  → { meeting: null }, fetch called once
 *
 * Happy-path (200 + JSON body) is verified via E2E against the live Go backend.
 * jsdom does not replicate Response.json() as a real async body stream, so we
 * test the null/error branches in unit tests and rely on E2E for the success path.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { Cookies } from '@sveltejs/kit';

const SERVER_URL = process.env.SERVER_URL ?? 'http://localhost:3000';

// Build a minimal Cookies-compatible mock for testing
function buildCookies(cookieHeader: string | null): Cookies {
  return {
    get: (name: string) => {
      if (!cookieHeader) return undefined;
      const match = cookieHeader.split(';').find((c) => {
        const [k] = c.trim().split('=');
        return k === name;
      });
      return match ? decodeURIComponent(match.split('=')[1]) : undefined;
    },
    getAll: () => [],
    set: () => {},
    delete: () => {},
    serialize: () => '',
  } as Cookies;
}

describe('PageServerLoad', () => {
  beforeEach(() => {
    vi.resetModules();
    // Restore global fetch to a clean state between tests
    globalThis.fetch = vi.fn();
  });

  afterEach(() => {
    // Don't leave a mocked fetch on the global object
    vi.restoreAllMocks();
  });

  it('returns meeting: null and skips fetch when X-User-ID cookie is absent', async () => {
    const fetchMock = vi.fn();
    globalThis.fetch = fetchMock;

    const { load } = await import('../../routes/meetings/[id]/+page.server');
    const result = await load({
      params: { id: '123' },
      cookies: buildCookies(null),
      fetch: fetchMock,
    } as any);

    expect(result).toEqual({ meeting: null });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('returns meeting: null and skips fetch when X-User-ID cookie is empty string', async () => {
    const fetchMock = vi.fn();
    globalThis.fetch = fetchMock;

    const { load } = await import('../../routes/meetings/[id]/+page.server');
    const result = await load({
      params: { id: '123' },
      cookies: buildCookies(''),
      fetch: fetchMock,
    } as any);

    expect(result).toEqual({ meeting: null });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('returns meeting: null when backend returns non-OK status', async () => {
    const notFoundResponse = new Response(null, { status: 404 });
    const fetchMock = vi.fn().mockResolvedValue(notFoundResponse);
    globalThis.fetch = fetchMock;

    const { load } = await import('../../routes/meetings/[id]/+page.server');
    const result = await load({
      params: { id: 'not-found' },
      cookies: buildCookies('X-User-ID=user-uuid-123'),
      fetch: fetchMock, // also pass as explicit arg
    } as any);

    expect(result).toEqual({ meeting: null });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('returns meeting: null when fetch rejects', async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error('Network failure'));
    globalThis.fetch = fetchMock;

    const { load } = await import('../../routes/meetings/[id]/+page.server');
    const result = await load({
      params: { id: '123' },
      cookies: buildCookies('X-User-ID=user-uuid-123'),
      fetch: fetchMock, // also pass as explicit arg
    } as any);

    expect(result).toEqual({ meeting: null });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});