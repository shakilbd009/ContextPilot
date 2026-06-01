<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { Button, Card, Alert, Badge, Spinner } from '$lib/components/ui';
  import BriefingStatusView from '$lib/components/upcoming/BriefingStatusView.svelte';
  import type { UpcomingMeeting } from '$lib/api/upcoming';
  import {
    getUpcomingMeeting,
    cancelUpcomingMeeting,
  } from '$lib/api/upcoming';

  let { data }: { data: { meetingId: string } } = $props();

  const ffUpcoming = import.meta.env.VITE_FF_ENABLE_UPCOMING_MEETINGS === 'true';
  const ffPreCallBriefing = import.meta.env.VITE_FF_ENABLE_PRE_CALL_BRIEFING === 'true';
  const meetingId = $derived(data.meetingId ?? $page.params.id);

  let meeting = $state<UpcomingMeeting | null>(null);
  let loading = $state(true);
  let cancelling = $state(false);
  let serverError = $state('');
  let notFound = $state(false);
  let timeRemaining = $state<number | null>(null);

  // ── Edit-window countdown (FR-9) ─────────────────────────────
  // Milliseconds remaining in the edit window; null means window not applicable
  const editWindowRemaining = $derived(() => {
    if (!meeting) return null;
    const deadline = new Date(new Date(meeting.scheduledStart).getTime() + 15 * 60 * 1000);
    return Math.max(0, deadline.getTime() - Date.now());
  });

  const isEditable = $derived(() => editWindowRemaining() > 0);
  const isCancellable = $derived(() => editWindowRemaining() > 0);

  // Live countdown tick
  $effect(() => {
    if (!meeting || meeting.status === 'cancelled') {
      timeRemaining = null;
      return;
    }
    timeRemaining = editWindowRemaining();
    const id = setInterval(() => {
      timeRemaining = editWindowRemaining();
      if (timeRemaining === 0) clearInterval(id);
    }, 1000);
    return () => clearInterval(id);
  });

  function formatCountdown(ms: number): string {
    const totalSec = Math.floor(ms / 1000);
    const h = Math.floor(totalSec / 3600);
    const m = Math.floor((totalSec % 3600) / 60);
    const s = totalSec % 60;
    if (h > 0) return `${h}h ${m}m remaining`;
    if (m > 0) return `${m}m ${s}s remaining`;
    return `${s}s remaining`;
  }

  async function loadMeeting() {
    loading = true;
    serverError = '';
    const res = await getUpcomingMeeting(meetingId, fetch);
    loading = false;
    if (res.ok) {
      meeting = res.data;
    } else if (res.status === 404 || res.data.error === 'not_found') {
      notFound = true;
    } else {
      serverError = res.data.message ?? 'Failed to load meeting.';
    }
  }

  async function handleCancel() {
    if (!meeting || !confirm('Cancel this meeting? This cannot be undone.')) return;
    cancelling = true;
    serverError = '';
    const res = await cancelUpcomingMeeting(meetingId, fetch);
    cancelling = false;
    if (res.ok) {
      meeting = res.data;
    } else {
      serverError = res.data.message ?? 'Failed to cancel meeting.';
    }
  }

  function formatDate(iso: string): string {
    try {
      return new Intl.DateTimeFormat('en-US', {
        month: 'long',
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

  // Load on mount
  $effect(() => {
    if (ffUpcoming) loadMeeting();
  });
</script>

<svelte:head>
  <title>{meeting?.title ?? 'Meeting Detail'} — ContextPilot</title>
</svelte:head>

{#if !ffUpcoming}
  <div class="page-container">
    <Alert variant="warning">
      Upcoming meetings are not enabled. Set <code>VITE_FF_ENABLE_UPCOMING_MEETINGS=true</code> to enable them.
    </Alert>
  </div>
{:else if loading}
  <div class="page-container">
    <div class="loading-state" aria-label="Loading meeting" aria-busy="true">
      <Spinner />
      <span>Loading meeting&hellip;</span>
    </div>
  </div>
{:else if notFound}
  <div class="page-container">
    <Alert variant="error">Meeting not found.</Alert>
    <div class="back-link">
      <Button variant="ghost" onclick={() => goto('/upcoming')}>← Back to upcoming meetings</Button>
    </div>
  </div>
{:else if meeting}
  <div class="page-container">
    <header class="page-header">
      <div class="page-header__left">
        <Button variant="ghost" size="sm" onclick={() => goto('/upcoming')}>← Upcoming</Button>
        <div class="meeting-status">
          <Badge
            variant={meeting.status === 'cancelled' ? 'danger' : 'success'}
          >
            {meeting.status}
          </Badge>
          {#if meeting.status === 'scheduled' && timeRemaining !== null && timeRemaining > 0}
            <span class="countdown-badge" aria-label="Edit window time remaining">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
                <circle cx="6" cy="6" r="5" stroke="currentColor" stroke-width="1.5"/>
                <path d="M6 3.5v2.5l1.5 1.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
              {formatCountdown(timeRemaining)}
            </span>
          {:else if meeting.status === 'scheduled' && !isEditable()}
            <span class="readonly-indicator" title="Edit window has passed">Read-only</span>
          {/if}
        </div>
      </div>
      <div class="page-header__actions">
        {#if meeting.status === 'scheduled'}
          {#if isEditable()}
            <Button variant="secondary" onclick={() => goto(`/upcoming/${meetingId}/edit`)}>
              Edit
            </Button>
          {/if}
          {#if isCancellable()}
            <Button variant="danger" loading={cancelling} onclick={handleCancel}>
              Cancel meeting
            </Button>
          {/if}
        {:else}
          <span class="cancelled-note">This meeting was cancelled</span>
        {/if}
      </div>
    </header>

    {#if serverError}
      <Alert variant="error">{serverError}</Alert>
    {/if}

    <div class="meeting-detail">
      <div class="meeting-detail__main">
        <div class="meeting-hero">
          <h1 class="meeting-hero__title">{meeting.title}</h1>
          <div class="meeting-hero__meta">
            <span class="meeting-hero__datetime">
              {formatDateTime(meeting.scheduledStart)}
            </span>
            {#if meeting.clientOrOrganization}
              <span class="meeting-hero__separator" aria-hidden="true">·</span>
              <span class="meeting-hero__org">{meeting.clientOrOrganization}</span>
            {/if}
          </div>
        </div>

        {#if meeting.description}
          <Card>
            <h2 class="section-title">Description / Agenda</h2>
            <p class="description-text">{meeting.description}</p>
          </Card>
        {/if}

        {#if meeting.participants && meeting.participants.length > 0}
          <Card>
            <h2 class="section-title">Participants</h2>
            <ul class="participants-list" aria-label="Participants">
              {#each meeting.participants as p}
                <li class="participant-item">
                  <div class="participant-item__avatar" aria-hidden="true">
                    {p.displayName.charAt(0).toUpperCase()}
                  </div>
                  <div class="participant-item__info">
                    <span class="participant-item__name">{p.displayName}</span>
                    {#if p.email}
                      <span class="participant-item__email">{p.email}</span>
                    {/if}
                    {#if p.organization}
                      <span class="participant-item__org">{p.organization}</span>
                    {/if}
                  </div>
                </li>
              {/each}
            </ul>
          </Card>
        {/if}
      </div>

      <aside class="meeting-detail__sidebar">
        {#if ffPreCallBriefing && meeting}
          <BriefingStatusView
            briefingStatus={meeting.briefingStatus}
            isBriefingStale={meeting.isBriefingStale}
            onNavigateToBriefing={() => goto(`/meetings/${meetingId}/briefing`)}
          />
        {/if}

        <Card>
          <h2 class="section-title">Details</h2>
          <dl class="details-list">
            <div class="details-list__item">
              <dt>Status</dt>
              <dd><Badge variant={meeting.status === 'cancelled' ? 'danger' : 'success'}>{meeting.status}</Badge></dd>
            </div>
            <div class="details-list__item">
              <dt>Scheduled</dt>
              <dd>{formatDateTime(meeting.scheduledStart)}</dd>
            </div>
            {#if meeting.clientOrOrganization}
              <div class="details-list__item">
                <dt>Client / Organization</dt>
                <dd>{meeting.clientOrOrganization}</dd>
              </div>
            {/if}
            <div class="details-list__item">
              <dt>Created</dt>
              <dd>{formatDate(meeting.createdAt)}</dd>
            </div>
          </dl>
        </Card>
      </aside>
    </div>
  </div>
{/if}

<style>
  .page-container {
    max-width: 1000px;
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
    align-items: center;
    gap: var(--space-3);
  }

  .meeting-status {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .readonly-indicator {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    font-style: italic;
  }

  .countdown-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: var(--text-xs);
    color: var(--color-warning);
    font-weight: 500;
  }

  .page-header__actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .cancelled-note {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    font-style: italic;
  }

  .back-link {
    margin-top: var(--space-2);
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

  /* ── Layout ── */
  .meeting-detail {
    display: grid;
    grid-template-columns: 1fr 280px;
    gap: var(--space-6);
    align-items: start;
  }

  @media (max-width: 768px) {
    .meeting-detail {
      grid-template-columns: 1fr;
    }
  }

  .meeting-detail__main {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .meeting-hero {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .meeting-hero__title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .meeting-hero__meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    flex-wrap: wrap;
  }

  .meeting-hero__separator {
    color: var(--color-border);
  }

  .section-title {
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0 0 var(--space-3);
  }

  .description-text {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.6;
    white-space: pre-wrap;
  }

  /* ── Participants ── */
  .participants-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .participant-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .participant-item__avatar {
    width: 2rem;
    height: 2rem;
    border-radius: 50%;
    background: var(--color-primary);
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--text-sm);
    font-weight: 600;
    flex-shrink: 0;
  }

  .participant-item__info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .participant-item__name {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  .participant-item__email {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .participant-item__org {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  /* ── Details ── */
  .details-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    margin: 0;
  }

  .details-list__item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .details-list__item dt {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .details-list__item dd {
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    margin: 0;
  }
</style>