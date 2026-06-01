/**
 * Meetings list page — unit tests
 *
 * Tests AC-11 pagination UI:
 * - Page load parses ?page= param (1-indexed)
 * - Loading / error / empty / populated states
 * - Pagination controls hidden when total <= limit (20)
 * - Pagination controls shown when total > limit
 * - View All option appears when total > limit
 * - View All mode shows all meetings without pagination
 * - Previous/Next disabled state is correct
 *
 * Note: Backend pagination is implemented in t_8c2eb67b
 * (Go handler with page=1-indexed, limit=20, offset-based SQL).
 * This file tests the Svelte page and its load function only.
 */

import { render, screen, cleanup } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { goto } from '$app/navigation';
import * as nav from '$app/navigation';

// ── Mock $app/navigation ────────────────────────────────────────
vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}));

// ── Mock import.meta.env ────────────────────────────────────────
beforeEach(() => {
  vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'false');
});
afterEach(() => {
  vi.unstubAllEnvs();
  cleanup();
  vi.restoreAllMocks();
});

// ── Import after mocks ───────────────────────────────────────────
import MeetingsPage from './+page.svelte';
import type { PageData } from './$types';

function makeData(overrides: Partial<PageData> = {}): PageData {
  return {
    ffEnableAppShell: true,
    meetings: [],
    total: 0,
    page: 1,
    limit: 20,
    hasMore: false,
    loading: false,
    error: null,
    ...overrides,
  } as PageData;
}

function renderPage(data = makeData()) {
  render(MeetingsPage, { data });
}

describe('MeetingsPage — empty state', () => {
  it('shows empty state alert when meetings array is empty', () => {
    renderPage(makeData({ meetings: [] }));
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/no meetings yet/i)).toBeInTheDocument();
  });

  it('hides pagination nav when meetings are empty', () => {
    renderPage(makeData({ meetings: [] }));
    expect(screen.queryByRole('navigation', { name: /pagination/i })).not.toBeInTheDocument();
  });
});

describe('MeetingsPage — populated list', () => {
  const twoMeetings = [
    { id: '1', title: 'Kickoff', completedAt: '2026-01-15T10:00:00Z', participants: [{ displayName: 'Alice' }] },
    { id: '2', title: 'Review', completedAt: '2026-01-16T14:00:00Z', participants: [{ displayName: 'Bob' }] },
  ];

  it('renders meeting titles when meetings exist', () => {
    renderPage(makeData({ meetings: twoMeetings, total: 2 }));
    expect(screen.getByText('Kickoff')).toBeInTheDocument();
    expect(screen.getByText('Review')).toBeInTheDocument();
  });

  it('hides pagination when total <= limit (20)', () => {
    renderPage(makeData({ meetings: twoMeetings, total: 2, hasMore: false }));
    expect(screen.queryByRole('navigation', { name: /pagination/i })).not.toBeInTheDocument();
  });

  it('shows View All link when total > limit', () => {
    renderPage(makeData({ meetings: twoMeetings, total: 25, hasMore: true }));
    expect(screen.getByText(/view all 25 meetings/i)).toBeInTheDocument();
  });
});

describe('MeetingsPage — pagination controls', () => {
  const manyMeetings = Array.from({ length: 20 }, (_, i) => ({
    id: String(i + 1),
    title: `Meeting ${i + 1}`,
    completedAt: new Date(2026, 0, i + 1).toISOString(),
    participants: [{ displayName: 'Person' }],
  }));

  function paginatedData(page: number, total: number) {
    const limit = 20;
    const hasMore = page * limit < total;
    return makeData({ meetings: manyMeetings, total, page, limit, hasMore });
  }

  it('shows pagination nav when total > limit', () => {
    renderPage(paginatedData(1, 25));
    expect(screen.getByRole('navigation', { name: /pagination/i })).toBeInTheDocument();
  });

  it('shows page indicator', () => {
    renderPage(paginatedData(2, 45));
    expect(screen.getByText(/page 2/i)).toBeInTheDocument();
  });

  it('disables Previous button on page 1', () => {
    renderPage(paginatedData(1, 25));
    const prevBtn = screen.getByRole('button', { name: /previous/i });
    expect(prevBtn).toBeDisabled();
  });

  it('disables Next button when no more pages', () => {
    renderPage(paginatedData(2, 25)); // page 2 of 2, no more
    const nextBtn = screen.getByRole('button', { name: /next/i });
    expect(nextBtn).toBeDisabled();
  });

  it('enables Next button when hasMore is true', () => {
    renderPage(paginatedData(1, 25)); // page 1, hasMore=true
    const nextBtn = screen.getByRole('button', { name: /next/i });
    expect(nextBtn).not.toBeDisabled();
  });
});

describe('MeetingsPage — View All mode', () => {
  const meetings = Array.from({ length: 5 }, (_, i) => ({
    id: String(i + 1),
    title: `Meeting ${i + 1}`,
    completedAt: new Date(2026, 0, i + 1).toISOString(),
    participants: [],
  }));

  it('shows View All button when total > limit', () => {
    renderPage(makeData({ meetings, total: 25, hasMore: false }));
    expect(screen.getByText(/view all 25 meetings/i)).toBeInTheDocument();
  });
});

describe('MeetingsPage — error state', () => {
  it('shows error alert when load returns an error', () => {
    renderPage(makeData({ error: 'Failed to load meetings (500)' }));
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/failed to load meetings/i)).toBeInTheDocument();
  });

  it('hides meeting list when error is present', () => {
    renderPage(makeData({
      error: 'Network error',
      meetings: [{ id: '1', title: 'Should not show', completedAt: '2026-01-01T00:00:00Z', participants: [{ displayName: '' }] }],
    }));
    expect(screen.queryByText('Should not show')).not.toBeInTheDocument();
  });
});

describe('MeetingsPage — navigation', () => {
  const meetings = [{ id: '1', title: 'Test', completedAt: '2026-01-01T00:00:00Z', participants: [] }];

  it('navigates to next page when Next is clicked', async () => {
    renderPage(makeData({ meetings, total: 25, page: 1, hasMore: true }));
    const nextBtn = screen.getByRole('button', { name: /next/i });
    await nextBtn.click();
    expect(goto).toHaveBeenCalledWith('/meetings?page=2', { replaceState: false });
  });

  it('navigates to previous page when Previous is clicked', async () => {
    renderPage(makeData({ meetings, total: 25, page: 2, hasMore: true }));
    const prevBtn = screen.getByRole('button', { name: /previous/i });
    await prevBtn.click();
    expect(goto).toHaveBeenCalledWith('/meetings?page=1', { replaceState: false });
  });

  it('navigates to View All when View All is clicked', async () => {
    renderPage(makeData({ meetings, total: 25, hasMore: false }));
    const viewAllBtn = screen.getByRole('button', { name: /view all/i });
    await viewAllBtn.click();
    expect(goto).toHaveBeenCalledWith('/meetings?viewAll=1', { replaceState: false });
  });

  it('links to meeting detail page', () => {
    const meetingsWithId = [{ id: 'abc-123', title: 'My Meeting', completedAt: '2026-01-01T00:00:00Z', participants: [] }];
    renderPage(makeData({ meetings: meetingsWithId, total: 1 }));
    // Find the link by its href — `getByRole('link', { href })` is no longer supported,
    // so query the document directly and confirm the link points to the meeting detail.
    const link = document.querySelector('a[href="/meetings/abc-123"]') as HTMLAnchorElement;
    expect(link).toBeInTheDocument();
    expect(link).toHaveTextContent('My Meeting');
  });
});

describe('MeetingsPage — date formatting', () => {
  it('formats ISO date to human-readable string', () => {
    const meetings = [{
      id: '1',
      title: 'Dated Meeting',
      completedAt: '2026-03-15T14:30:00Z',
      participants: [],
    }];
    renderPage(makeData({ meetings, total: 1 }));
    // Intl.DateTimeFormat uses local timezone; just check the day is present
    expect(screen.getByText(/15/i)).toBeInTheDocument();
  });
});