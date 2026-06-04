/**
 * Upcoming meetings list/calendar page — unit tests
 * Ref: evals/unit/brd-05-manual-upcoming-meeting-creation.md
 *
 * Tests:
 * - Feature flag disabled → warning alert shown
 * - Loading / error / empty states
 * - List view renders meetings with correct metadata
 * - Calendar view groups meetings by date
 * - View toggle (list ↔ calendar)
 * - Schedule meeting button navigates to /upcoming/new
 * - List item links to /upcoming/[id]
 * - Meeting count displayed
 *
 * Note: Data loaded server-side via +page.server.ts load function.
 * Feature flag: VITE_FF_ENABLE_UPCOMING_MEETINGS
 */

import { render, screen, cleanup } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { goto } from '$app/navigation';

// ── Mock $app/navigation ────────────────────────────────────────
vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}));

// ── Mock $app/state — create URL lazily inside factory ───────────
// This avoids TDZ issues with module-level const variables.
let viewMode = 'list';
vi.mock('$app/state', () => ({
  get page() {
    return {
      url: {
        searchParams: {
          get(key: string) {
            if (key === 'view') return viewMode;
            return null;
          },
        },
      },
    };
  },
}));

// ── Helper to set view mode for tests ───────────────────────────
function setViewMode(mode: string) {
  viewMode = mode;
}

// ── Mock import.meta.env ────────────────────────────────────────
beforeEach(() => {
  setViewMode('list');
  vi.stubEnv('VITE_FF_ENABLE_UPCOMING_MEETINGS', 'true');
  vi.clearAllMocks();
});
afterEach(() => {
  vi.unstubAllEnvs();
  cleanup();
  vi.restoreAllMocks();
});

// ── Import after mocks ───────────────────────────────────────────
import UpcomingListPage from './+page.svelte';
import type { PageData } from './$types';

// ── Helper factories ─────────────────────────────────────────────

function makeMeeting(overrides: Partial<{
  id: string;
  title: string;
  scheduledStart: string;
  status: 'scheduled' | 'cancelled';
  clientOrOrganization: string;
  participants: Array<{ displayName: string }>;
}> = {}) {
  return {
    id: '1',
    title: 'Q3 Planning',
    scheduledStart: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
    status: 'scheduled' as const,
    clientOrOrganization: '',
    participants: [],
    ...overrides,
  };
}

function makeData(overrides: Partial<PageData> = {}): PageData {
  return {
    ffEnableAppShell: false,
    meetings: [],
    total: 0,
    loading: false,
    error: null,
    ...overrides,
  } as PageData;
}

// ── Tests ─────────────────────────────────────────────────────────

describe('UpcomingListPage — feature flag disabled', () => {
  it('shows warning alert when flag is false', () => {
    vi.stubEnv('VITE_FF_ENABLE_UPCOMING_MEETINGS', 'false');
    cleanup();
    render(UpcomingListPage, { data: makeData() });
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/VITE_FF_ENABLE_UPCOMING_MEETINGS=true/i)).toBeInTheDocument();
  });
});

describe('UpcomingListPage — loading state', () => {
  it('shows spinner when loading=true', () => {
    render(UpcomingListPage, { data: makeData({ loading: true }) });
    expect(screen.getByLabelText(/loading meetings/i)).toBeInTheDocument();
  });
});

describe('UpcomingListPage — error state', () => {
  it('shows error alert when error is present', () => {
    render(UpcomingListPage, { data: makeData({ error: 'Failed to load meetings (500)' }) });
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/failed to load meetings/i)).toBeInTheDocument();
  });
});

describe('UpcomingListPage — empty state', () => {
  it('shows empty state when meetings array is empty and not loading', () => {
    render(UpcomingListPage, { data: makeData({ meetings: [], total: 0 }) });
    expect(screen.getByText(/no upcoming meetings/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /schedule a meeting/i })).toBeInTheDocument();
  });

  it('has no list or calendar view when meetings are empty', () => {
    render(UpcomingListPage, { data: makeData({ meetings: [], total: 0 }) });
    expect(screen.queryByLabelText(/upcoming meetings list/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/upcoming meetings calendar/i)).not.toBeInTheDocument();
  });
});

describe('UpcomingListPage — list view', () => {
  const meetings = [
    makeMeeting({ id: 'm1', title: 'Kickoff', scheduledStart: new Date(Date.now() + 86400_000).toISOString() }),
    makeMeeting({ id: 'm2', title: 'Review', scheduledStart: new Date(Date.now() + 172800_000).toISOString() }),
  ];

  it('renders meeting titles in list view', () => {
    render(UpcomingListPage, { data: makeData({ meetings, total: 2 }) });
    expect(screen.getByText('Kickoff')).toBeInTheDocument();
    expect(screen.getByText('Review')).toBeInTheDocument();
  });

  it('renders meeting count in header', () => {
    render(UpcomingListPage, { data: makeData({ meetings, total: 2 }) });
    expect(screen.getByText('2 meetings')).toBeInTheDocument();
  });

  it('links to meeting detail page', () => {
    render(UpcomingListPage, { data: makeData({ meetings, total: 1 }) });
    // The link contains the title as text but also has date/time/meta content
    // Use a regex that matches the accessible name (includes title but may have more)
    const link = screen.getByRole('link', { name: /kickoff/i });
    expect(link).toHaveAttribute('href', '/upcoming/m1');
  });

  it('shows client/org when present', () => {
    const withOrg = makeMeeting({ id: 'm1', title: 'Acme Sync', clientOrOrganization: 'Acme Corp' });
    render(UpcomingListPage, { data: makeData({ meetings: [withOrg], total: 1 }) });
    expect(screen.getByText('Acme Corp')).toBeInTheDocument();
  });

  it('shows participant count when participants exist', () => {
    const withParticipants = makeMeeting({
      id: 'm1',
      participants: [{ displayName: 'Alice' }, { displayName: 'Bob' }],
    });
    render(UpcomingListPage, { data: makeData({ meetings: [withParticipants], total: 1 }) });
    expect(screen.getByText('2 participants')).toBeInTheDocument();
  });

  it('hides participant count when no participants', () => {
    const noParticipants = makeMeeting({ id: 'm1', participants: [] });
    render(UpcomingListPage, { data: makeData({ meetings: [noParticipants], total: 1 }) });
    // "X participants" count should not appear when array is empty
    // Note: the hint "Add participants to improve briefing accuracy" always renders
    // so we query for the specific count pattern (digit + "participant")
    expect(screen.queryByText(/^\d+ participants?$/)).not.toBeInTheDocument();
  });
});

describe('UpcomingListPage — calendar view', () => {
  it('renders calendar view when view=calendar', () => {
    setViewMode('calendar');
    cleanup();
    const tomorrow = new Date(Date.now() + 86400_000);
    const meetings = [makeMeeting({ id: 'm1', title: 'Sync', scheduledStart: tomorrow.toISOString() })];
    render(UpcomingListPage, { data: makeData({ meetings, total: 1 }) });
    expect(screen.getByLabelText(/upcoming meetings calendar/i)).toBeInTheDocument();
  });

  it('renders meetings in calendar view', () => {
    setViewMode('calendar');
    cleanup();
    const day1 = new Date(Date.now() + 86400_000).toISOString();
    const day2 = new Date(Date.now() + 172800_000).toISOString();
    const meetings = [
      makeMeeting({ id: 'm1', title: 'Meeting 1', scheduledStart: day1 }),
      makeMeeting({ id: 'm2', title: 'Meeting 2', scheduledStart: day2 }),
    ];
    render(UpcomingListPage, { data: makeData({ meetings, total: 2 }) });
    expect(screen.getByText('Meeting 1')).toBeInTheDocument();
    expect(screen.getByText('Meeting 2')).toBeInTheDocument();
  });
});

describe('UpcomingListPage — view toggle', () => {
  it('shows list and calendar toggle buttons', () => {
    render(UpcomingListPage, { data: makeData({ meetings: [] }) });
    expect(screen.getByRole('button', { name: /list/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /calendar/i })).toBeInTheDocument();
  });

  it('links to ?view=calendar for hydration-safe navigation', () => {
    render(UpcomingListPage, { data: makeData({ meetings: [] }) });
    const calBtn = screen.getByRole('button', { name: /calendar/i });
    expect(calBtn).toHaveAttribute('href', '/upcoming?view=calendar');
  });

  it('links to ?view=list for hydration-safe navigation', () => {
    setViewMode('calendar');
    cleanup();
    render(UpcomingListPage, { data: makeData({ meetings: [] }) });
    const listBtn = screen.getByRole('button', { name: /list/i });
    expect(listBtn).toHaveAttribute('href', '/upcoming?view=list');
  });
});

describe('UpcomingListPage — schedule meeting button', () => {
  it('navigates to /upcoming/new when clicked', async () => {
    render(UpcomingListPage, { data: makeData({ meetings: [] }) });
    const btn = screen.getByRole('button', { name: /schedule meeting/i });
    await btn.click();
    expect(goto).toHaveBeenCalledWith('/upcoming/new');
  });
});

describe('UpcomingListPage — page title', () => {
  it('sets correct page title', () => {
    render(UpcomingListPage, { data: makeData({ meetings: [] }) });
    expect(document.title).toContain('Upcoming Meetings');
  });
});