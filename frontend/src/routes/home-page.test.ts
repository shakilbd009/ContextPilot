/**
 * Root +page.svelte — unit tests
 * Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md (F1 — CWE-79 XSS repair)
 *
 * Tests:
 * - F1 (XSS) regression: a malicious title from /api/upcoming/dashboard-count
 *   is rendered as literal text inside .next-meeting-hint — no <img>/<script>
 *   element is inserted into the DOM, and the onerror/onload handler does
 *   not fire (because Svelte's `{...}` text interpolation escapes HTML).
 * - Loading state renders "Loading…"
 * - Error state renders "Unable to load"
 * - count === 0 renders the empty hint with the "Schedule one" link
 * - count === 1 with a benign next meeting renders the title + relative day
 * - count > 1 renders the plural "X meetings scheduled"
 *
 * Note: The +page.svelte uses an `$effect` that fetches
 * /api/upcoming/dashboard-count on mount (when VITE_FF_ENABLE_UPCOMING_MEETINGS=true).
 * We mock `globalThis.fetch` to control the response per test.
 *
 * Note 2: This test file lives at `src/routes/home-page.test.ts` (not
 * `+page.test.ts`) because vitest v4.1.7 reserves `+`-prefixed filenames
 * (the warning is benign but the file gets filtered from the include glob
 * in some setups). The test still imports the SvelteKit virtual `./$types`.
 */

import { render, cleanup, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

import Page from './+page.svelte';
import type { PageData } from './$types';

// ── Test helper ───────────────────────────────────────────────────

function makeData(overrides: Partial<PageData> = {}): PageData {
  return {
    ffEnableAppShell: true,
    ...overrides,
  } as PageData;
}

function mockFetchOk(payload: unknown) {
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => payload,
  } as unknown as Response);
}

function mockFetchError() {
  return vi.fn().mockRejectedValue(new Error('network down'));
}

// ── Setup ─────────────────────────────────────────────────────────

beforeEach(() => {
  vi.stubEnv('VITE_FF_ENABLE_UPCOMING_MEETINGS', 'true');
  // Default: a benign empty state. Tests that need a specific payload override fetch.
  globalThis.fetch = mockFetchOk({ count: 0, nextMeeting: null });
});

afterEach(() => {
  vi.unstubAllEnvs();
  cleanup();
  vi.restoreAllMocks();
});

// ── F1 regression: stored XSS in dashboard hint ───────────────────

describe('Root +page.svelte — F1 XSS regression (CWE-79)', () => {
  it('renders a malicious title as literal text, NOT as an element', async () => {
    const maliciousTitle = '<img src=x onerror=window.__pocFired=true;alert("XSS-DASH-POC")>';
    const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
    globalThis.fetch = mockFetchOk({
      count: 1,
      nextMeeting: { title: maliciousTitle, scheduledStart: tomorrow },
    });

    render(Page, { props: { data: makeData() } });

    // Wait for the fetch + reactive update to settle.
    const hint = await waitFor(() => {
      const el = document.querySelector('.next-meeting-hint');
      expect(el).not.toBeNull();
      return el as HTMLElement;
    });

    // The exact payload should appear as a text node — never parsed as HTML.
    expect(hint.textContent).toContain(maliciousTitle);

    // Critical: no <img> element should have been inserted. Svelte's `{...}`
    // text interpolation escapes HTML, so the raw "<img ...>" string lives
    // in a text node, not as a real DOM element.
    expect(hint.querySelector('img')).toBeNull();
    expect(hint.querySelector('script')).toBeNull();
    expect(hint.querySelector('iframe')).toBeNull();
    expect(hint.querySelector('object')).toBeNull();
    expect(hint.querySelector('embed')).toBeNull();

    // The onerror handler attribute should not exist anywhere in the hint subtree.
    expect(hint.querySelector('[onerror]')).toBeNull();
    expect(hint.querySelector('[onload]')).toBeNull();
  });

  it('renders script-tag payload as literal text without executing', async () => {
    const scriptPayload = "<script>window.__pocScriptFired=true;alert('XSS-SCRIPT')</script>";
    const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
    globalThis.fetch = mockFetchOk({
      count: 1,
      nextMeeting: { title: scriptPayload, scheduledStart: tomorrow },
    });

    render(Page, { props: { data: makeData() } });

    const hint = await waitFor(() => {
      const el = document.querySelector('.next-meeting-hint');
      expect(el).not.toBeNull();
      return el as HTMLElement;
    });

    expect(hint.textContent).toContain(scriptPayload);
    expect(hint.querySelector('script')).toBeNull();
  });

  it('escapes payload attributes that try to break out of attribute context', async () => {
    const breakout = '"><img src=x onerror=alert(1)>';
    const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
    globalThis.fetch = mockFetchOk({
      count: 1,
      nextMeeting: { title: breakout, scheduledStart: tomorrow },
    });

    render(Page, { props: { data: makeData() } });

    const hint = await waitFor(() => {
      const el = document.querySelector('.next-meeting-hint');
      expect(el).not.toBeNull();
      return el as HTMLElement;
    });

    expect(hint.textContent).toContain(breakout);
    expect(hint.querySelector('img')).toBeNull();
  });
});

// ── Loading / error / empty / n>1 / benign n=1 ────────────────────

describe('Root +page.svelte — upcoming meetings card states', () => {
  it('renders loading state while fetch is pending', () => {
    // Make fetch never resolve so we observe the initial loading branch.
    globalThis.fetch = vi.fn(() => new Promise(() => { /* never resolves */ })) as unknown as typeof fetch;

    render(Page, { props: { data: makeData() } });

    const card = document.getElementById('upcoming-meetings-card');
    expect(card).not.toBeNull();
    expect(card!.textContent).toContain('Loading…');
  });

  it('renders "Unable to load upcoming meetings." on fetch failure', async () => {
    globalThis.fetch = mockFetchError();

    render(Page, { props: { data: makeData() } });

    await waitFor(() => {
      const card = document.getElementById('upcoming-meetings-card');
      expect(card?.textContent).toContain('Unable to load upcoming meetings');
    });

    // The stat card should also flip to the "Unable to load" hint.
    const hintEl = document.getElementById('upcoming-hint');
    expect(hintEl?.textContent).toBe('Unable to load');
    const countEl = document.getElementById('upcoming-count');
    expect(countEl?.textContent).toBe('—');
  });

  it('renders the empty hint with the schedule link when count === 0', async () => {
    globalThis.fetch = mockFetchOk({ count: 0, nextMeeting: null });

    render(Page, { props: { data: makeData() } });

    // Wait for the empty-state branch to render by waiting for the schedule link.
    const link = await waitFor(() => {
      const el = document.querySelector('#upcoming-meetings-card a[href="/upcoming/new"]');
      expect(el).not.toBeNull();
      return el as HTMLAnchorElement;
    });

    expect(link.textContent).toContain('Schedule one');

    // The full empty hint should now be present.
    const card = document.getElementById('upcoming-meetings-card')!;
    expect(card.textContent).toContain('No upcoming meetings');
  });

  it('renders the next-meeting hint with a benign title and relative day', async () => {
    const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
    globalThis.fetch = mockFetchOk({
      count: 1,
      nextMeeting: { title: 'Q3 Planning', scheduledStart: tomorrow },
    });

    render(Page, { props: { data: makeData() } });

    const hint = await waitFor(() => {
      const el = document.querySelector('.next-meeting-hint');
      expect(el).not.toBeNull();
      return el as HTMLElement;
    });

    expect(hint.textContent).toContain('Q3 Planning');
    expect(hint.querySelector('.muted')).not.toBeNull();
  });

  it('renders the plural "X meetings scheduled" hint when count > 1', async () => {
    globalThis.fetch = mockFetchOk({
      count: 3,
      nextMeeting: { title: 'Earliest', scheduledStart: new Date(Date.now() + 86400_000).toISOString() },
    });

    render(Page, { props: { data: makeData() } });

    await waitFor(() => {
      const el = document.querySelector('.next-meeting-hint');
      expect(el?.textContent).toContain('3 meetings scheduled');
    });
  });

  it('does not fetch when VITE_FF_ENABLE_UPCOMING_MEETINGS is false', () => {
    vi.stubEnv('VITE_FF_ENABLE_UPCOMING_MEETINGS', 'false');
    const fetchSpy = vi.fn();
    globalThis.fetch = fetchSpy as unknown as typeof fetch;

    render(Page, { props: { data: makeData() } });

    expect(fetchSpy).not.toHaveBeenCalled();
  });
});
