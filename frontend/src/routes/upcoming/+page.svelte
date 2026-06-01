<script lang="ts">
  import type { PageData } from './$types';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { Button, Card, Alert, Badge, Spinner } from '$lib/components/ui';
  import type { UpcomingMeeting } from '$lib/api/upcoming';

  let { data }: { data: PageData } = $props();

  const ffEnableUpcomingMeetings = import.meta.env.VITE_FF_ENABLE_UPCOMING_MEETINGS === 'true';

  // ── View mode (persisted in URL search param) ─────────────────
  const viewMode = $derived(page.url.searchParams.get('view') ?? 'list');

  // ── Server-side data ───────────────────────────────────────────
  const meetings = $derived(data.meetings ?? ([] as UpcomingMeeting[]));
  const loading = $derived(data.loading ?? false);
  const error = $derived(data.error ?? null);
  const total = $derived(data.total ?? 0);

  // ── Formatting helpers ─────────────────────────────────────────
  function formatDate(iso: string): string {
    try {
      return new Intl.DateTimeFormat('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      }).format(new Date(iso));
    } catch {
      return iso;
    }
  }

  function formatTime(iso: string): string {
    try {
      return new Intl.DateTimeFormat('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        hour12: true,
      }).format(new Date(iso));
    } catch {
      return '';
    }
  }

  function formatDateTime(iso: string): string {
    return `${formatDate(iso)} at ${formatTime(iso)}`;
  }

  function isToday(iso: string): boolean {
    const d = new Date(iso);
    const today = new Date();
    return d.toDateString() === today.toDateString();
  }

  function isTomorrow(iso: string): boolean {
    const d = new Date(iso);
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    return d.toDateString() === tomorrow.toDateString();
  }

  function relativeDay(iso: string): string {
    if (isToday(iso)) return 'Today';
    if (isTomorrow(iso)) return 'Tomorrow';
    return formatDate(iso);
  }

  // Group meetings by date for calendar view
  const groupedByDate = $derived(() => {
    const groups: Record<string, UpcomingMeeting[]> = {};
    for (const m of meetings) {
      const key = formatDate(m.scheduledStart);
      if (!groups[key]) groups[key] = [];
      groups[key].push(m);
    }
    return groups;
  });
</script>

<svelte:head>
  <title>Upcoming Meetings — ContextPilot</title>
</svelte:head>

{#if !ffEnableUpcomingMeetings}
  <div class="page-container">
    <Alert variant="warning">
      Upcoming meetings are not enabled. Set <code>VITE_FF_ENABLE_UPCOMING_MEETINGS=true</code> to enable them.
    </Alert>
  </div>
{:else if error}
  <div class="page-container">
    <Alert variant="error">{error}</Alert>
  </div>
{:else}
  <div class="page-container">
    <header class="page-header">
      <div class="page-header__left">
        <h1 class="page-title">Upcoming Meetings</h1>
        {#if total > 0}
          <span class="meeting-count" aria-live="polite">{total} meeting{total !== 1 ? 's' : ''}</span>
        {/if}
      </div>
      <div class="page-header__actions">
        <!-- View mode toggle -->
        <div class="view-toggle" role="group" aria-label="Switch view">
          <Button
            variant={viewMode === 'list' ? 'secondary' : 'ghost'}
            size="sm"
            onclick={() => goto('/upcoming?view=list')}
            aria-pressed={viewMode === 'list'}
            aria-current={viewMode === 'list' ? 'true' : undefined}
          >
            List
          </Button>
          <Button
            variant={viewMode === 'calendar' ? 'secondary' : 'ghost'}
            size="sm"
            onclick={() => goto('/upcoming?view=calendar')}
            aria-pressed={viewMode === 'calendar'}
            aria-current={viewMode === 'calendar' ? 'true' : undefined}
          >
            Calendar
          </Button>
        </div>
        <Button variant="primary" onclick={() => goto('/upcoming/new')}>
          Schedule meeting
        </Button>
      </div>
    </header>

    {#if loading}
      <div class="loading-state" aria-label="Loading meetings" aria-busy="true">
        <Spinner />
        <span>Loading upcoming meetings…</span>
      </div>
    {:else if meetings.length === 0}
      <!-- Empty state -->
      <div class="empty-state">
        <Card>
          <div class="empty-state__content">
            <svg class="empty-state__icon" width="48" height="48" viewBox="0 0 48 48" fill="none" aria-hidden="true">
              <rect x="6" y="10" width="36" height="32" rx="4" stroke="currentColor" stroke-width="2"/>
              <path d="M6 18h36M16 6v8M32 6v8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
            <h2 class="empty-state__title">No upcoming meetings</h2>
            <p class="empty-state__desc">Schedule your first meeting to start getting pre-call briefings.</p>
            <Button variant="primary" onclick={() => goto('/upcoming/new')}>Schedule a meeting</Button>
          </div>
        </Card>
      </div>
    {:else if viewMode === 'list'}
      <!-- List view -->
      <div class="meetings-list" aria-label="Upcoming meetings list">
        {#each meetings as meeting (meeting.id)}
          <div class="meeting-card">
            <a href="/upcoming/{meeting.id}" class="meeting-link">
              <div class="meeting-card__main">
                <div class="meeting-card__date-badge" class:meeting-card__date-badge--today={isToday(meeting.scheduledStart)}>
                  <span class="date-badge__day">{new Date(meeting.scheduledStart).toLocaleDateString('en-US', { weekday: 'short' })}</span>
                  <span class="date-badge__num">{new Date(meeting.scheduledStart).getDate()}</span>
                </div>
                <div class="meeting-card__content">
                  <h2 class="meeting-card__title">{meeting.title}</h2>
                  <div class="meeting-card__meta">
                    <span class="meeting-card__time">
                      {formatTime(meeting.scheduledStart)}
                    </span>
                    {#if meeting.clientOrOrganization}
                      <span class="meeting-card__separator" aria-hidden="true">·</span>
                      <span class="meeting-card__org">{meeting.clientOrOrganization}</span>
                    {/if}
                    {#if meeting.participants?.length}
                      <span class="meeting-card__separator" aria-hidden="true">·</span>
                      <span class="meeting-card__participants">
                        {meeting.participants.length} participant{meeting.participants.length !== 1 ? 's' : ''}
                      </span>
                    {/if}
                  </div>
                </div>
              </div>
              <span class="meeting-card__arrow" aria-hidden="true">→</span>
            </a>
          </div>
        {/each}
      </div>
    {:else}
      <!-- Calendar view -->
      <div class="calendar-view" aria-label="Upcoming meetings calendar">
        {#each Object.entries(groupedByDate()) as [date, dayMeetings]}
          <div class="calendar-day">
            <h2 class="calendar-day__header">{date}</h2>
            <div class="calendar-day__meetings">
              {#each dayMeetings as meeting}
                <a href="/upcoming/{meeting.id}" class="calendar-meeting">
                  <div class="calendar-meeting__time">{formatTime(meeting.scheduledStart)}</div>
                  <div class="calendar-meeting__title">{meeting.title}</div>
                  {#if meeting.clientOrOrganization}
                    <div class="calendar-meeting__org">{meeting.clientOrOrganization}</div>
                  {/if}
                </a>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .page-container {
    max-width: 900px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    flex-wrap: wrap;
  }

  .page-header__left {
    display: flex;
    align-items: baseline;
    gap: var(--space-3);
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .meeting-count {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }

  .page-header__actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .view-toggle {
    display: flex;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    overflow: hidden;
  }

  /* ── Loading ── */
  .loading-state {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-8);
    justify-content: center;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }

  /* ── Empty state ── */
  .empty-state__content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-8);
    text-align: center;
  }

  .empty-state__icon {
    color: var(--color-text-muted);
  }

  .empty-state__title {
    font-size: var(--text-lg);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .empty-state__desc {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin: 0;
    max-width: 320px;
  }

  /* ── List view ── */
  .meetings-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .meeting-card {
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-sm);
    transition: box-shadow var(--transition-fast), border-color var(--transition-fast);
  }

  .meeting-card:hover {
    box-shadow: var(--shadow-md);
    border-color: var(--color-text-muted);
  }

  .meeting-link {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    padding: var(--space-4) var(--space-5);
    text-decoration: none;
    color: inherit;
    border-radius: var(--radius-sm);
  }

  .meeting-card__main {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    min-width: 0;
  }

  .meeting-card__date-badge {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 3rem;
    height: 3rem;
    border-radius: var(--radius-sm);
    background: var(--color-background-subtle);
    border: 1px solid var(--color-border);
    flex-shrink: 0;
  }

  .meeting-card__date-badge--today {
    background: var(--color-primary);
    border-color: var(--color-primary);
    color: white;
  }

  .date-badge__day {
    font-size: 0.625rem;
    font-weight: 600;
    text-transform: uppercase;
    line-height: 1;
  }

  .date-badge__num {
    font-size: var(--text-lg);
    font-weight: 700;
    line-height: 1;
    margin-top: 1px;
  }

  .meeting-card__content {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    min-width: 0;
  }

  .meeting-card__title {
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .meeting-card__meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    flex-wrap: wrap;
  }

  .meeting-card__separator {
    color: var(--color-border);
  }

  .meeting-card__arrow {
    font-size: var(--text-lg);
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  /* ── Calendar view ── */
  .calendar-view {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .calendar-day__header {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-secondary);
    margin: 0 0 var(--space-3);
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--color-border);
  }

  .calendar-day__meetings {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .calendar-meeting {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-4);
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    text-decoration: none;
    color: inherit;
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  }

  .calendar-meeting:hover {
    border-color: var(--color-primary);
    box-shadow: var(--shadow-sm);
  }

  .calendar-meeting__time {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-primary);
    min-width: 5rem;
  }

  .calendar-meeting__title {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
    flex: 1;
  }

  .calendar-meeting__org {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
</style>