/**
 * Unit tests for upcoming/+page.server.ts
 * Verifies the server-side load function for the /upcoming list page.
 *
 * Coverage:
 *  - Feature flag disabled             → empty data, no store access
 *  - Feature flag enabled, empty store → seeded with demo meeting (Q3 Planning Review)
 *  - Feature flag enabled, populated   → returns meetings sorted by scheduledStart asc
 *  - Status filter (status=scheduled)  → only 'scheduled' meetings returned
 *  - Status filter (status=all)        → all meetings returned regardless of status
 *  - Status filter (status=cancelled)  → only 'cancelled' meetings returned
 *  - Sort order                        → meetings sorted by scheduledStart asc
 *  - Try/catch on store error          → returns error branch (empty + error string)
 *
 * Background: previously the load function called fetch('/api/upcoming?...') which
 * in dev mode got routed to the Vite proxy → Go backend → 4xx/5xx, causing the
 * /upcoming page to render the error branch instead of the h1. The fix reads the
 * in-memory store directly. These tests pin the new behavior.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// ── Reset global store + module cache before each test ───────────
beforeEach(() => {
  // @ts-expect-error — test setup: drop the global store between tests
  delete globalThis.__upcomingStore;
  vi.resetModules();
  // Stub the Vite env var the load function reads
  vi.stubEnv('VITE_FF_ENABLE_UPCOMING_MEETINGS', 'true');
});

afterEach(() => {
  vi.unstubAllEnvs();
  // @ts-expect-error — test cleanup
  delete globalThis.__upcomingStore;
});

function buildUrl(qs: Record<string, string> = {}): URL {
  const u = new URL('http://localhost/upcoming');
  for (const [k, v] of Object.entries(qs)) u.searchParams.set(k, v);
  return u;
}

describe('upcoming +page.server.ts load function', () => {
  it('returns empty data when feature flag is disabled', async () => {
    vi.stubEnv('VITE_FF_ENABLE_UPCOMING_MEETINGS', 'false');
    // Import via the actual route path so the file resolves correctly.
    // Module is at src/routes/upcoming/+page.server.ts; test is at src/tests/unit/.
    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl() } as any);
    expect(data).toEqual({ meetings: [], total: 0, loading: false, error: null });
    // Store must NOT be created when the flag is off
    expect(globalThis.__upcomingStore).toBeUndefined();
  });

  it('seeds the demo meeting (Q3 Planning Review) when the store is empty', async () => {
    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl() } as any);
    expect(data.error).toBeNull();
    expect(data.meetings).toHaveLength(1);
    expect(data.total).toBe(1);
    expect(data.meetings[0].title).toBe('Q3 Planning Review');
    expect(data.meetings[0].id).toBe('11111111-1111-1111-1111-111111111111');
    expect(data.meetings[0].status).toBe('scheduled');
  });

  it('does not re-seed when the store already contains meetings', async () => {
    // Pre-populate the store with a sentinel meeting
    const store = new Map<string, any>();
    store.set('sentinel', { id: 'sentinel', title: 'Pre-existing', scheduledStart: '2030-01-01T00:00:00.000Z', status: 'scheduled', participants: [] });
    globalThis.__upcomingStore = store as any;

    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl() } as any);
    expect(data.meetings).toHaveLength(1);
    expect(data.meetings[0].id).toBe('sentinel');
    expect(data.meetings[0].title).toBe('Pre-existing');
  });

  it('returns meetings sorted by scheduledStart ascending', async () => {
    const store = new Map<string, any>();
    store.set('a', { id: 'a', title: 'Later', scheduledStart: '2030-06-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    store.set('b', { id: 'b', title: 'Earliest', scheduledStart: '2030-01-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    store.set('c', { id: 'c', title: 'Middle', scheduledStart: '2030-03-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    globalThis.__upcomingStore = store as any;

    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl() } as any);
    expect(data.meetings.map((m: any) => m.id)).toEqual(['b', 'c', 'a']);
  });

  it('filters by status=scheduled by default', async () => {
    const store = new Map<string, any>();
    store.set('s1', { id: 's1', title: 'Sched A', scheduledStart: '2030-01-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    store.set('s2', { id: 's2', title: 'Sched B', scheduledStart: '2030-02-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    store.set('c1', { id: 'c1', title: 'Cancelled', scheduledStart: '2030-03-01T10:00:00.000Z', status: 'cancelled', participants: [] });
    globalThis.__upcomingStore = store as any;

    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl() } as any);
    expect(data.meetings).toHaveLength(2);
    expect(data.meetings.map((m: any) => m.id).sort()).toEqual(['s1', 's2']);
  });

  it('filters by status=all to include cancelled meetings', async () => {
    const store = new Map<string, any>();
    store.set('s1', { id: 's1', title: 'Sched', scheduledStart: '2030-01-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    store.set('c1', { id: 'c1', title: 'Cancelled', scheduledStart: '2030-02-01T10:00:00.000Z', status: 'cancelled', participants: [] });
    globalThis.__upcomingStore = store as any;

    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl({ status: 'all' }) } as any);
    expect(data.meetings).toHaveLength(2);
  });

  it('filters by status=cancelled to include only cancelled meetings', async () => {
    const store = new Map<string, any>();
    store.set('s1', { id: 's1', title: 'Sched', scheduledStart: '2030-01-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    store.set('c1', { id: 'c1', title: 'Cancelled', scheduledStart: '2030-02-01T10:00:00.000Z', status: 'cancelled', participants: [] });
    globalThis.__upcomingStore = store as any;

    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl({ status: 'cancelled' }) } as any);
    expect(data.meetings).toHaveLength(1);
    expect(data.meetings[0].id).toBe('c1');
  });

  it('returns total equal to the filtered meetings length', async () => {
    const store = new Map<string, any>();
    store.set('s1', { id: 's1', title: 'Sched', scheduledStart: '2030-01-01T10:00:00.000Z', status: 'scheduled', participants: [] });
    store.set('c1', { id: 'c1', title: 'Cancelled', scheduledStart: '2030-02-01T10:00:00.000Z', status: 'cancelled', participants: [] });
    globalThis.__upcomingStore = store as any;

    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl() } as any);
    expect(data.total).toBe(1);
  });

  it('returns the error branch shape when an unexpected error occurs', async () => {
    // Force Array.from(store.values()) to throw by making __upcomingStore a non-Map
    globalThis.__upcomingStore = { size: 0, values: () => { throw new Error('boom'); } } as any;

    const mod = await import('../../routes/upcoming/+page.server');
    const data = await mod.load({ url: buildUrl() } as any);
    expect(data.error).toBe('Failed to load upcoming meetings.');
    expect(data.meetings).toEqual([]);
    expect(data.total).toBe(0);
  });
});
