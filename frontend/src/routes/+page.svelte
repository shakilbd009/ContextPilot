<script lang="ts">
  import type { PageData } from './$types';
  import { Card, Badge, Button, Alert } from '$lib/components/ui';
  import { isAuthenticated } from '$lib/stores/auth';

  let { data }: { data: PageData } = $props();

  // AC-6: feature flag gate — show landing when disabled
  const appShellEnabled = $derived(data.ffEnableAppShell);

  // ── Upcoming meetings dashboard (FR-21) ──────────────────────────
  const ffEnableUpcomingMeetings = import.meta.env.VITE_FF_ENABLE_UPCOMING_MEETINGS === 'true';

  // F1 (CWE-79) repair: the previous implementation imperatively set
  // `cardEl.innerHTML = \`<p>${next.title} ...</p>\``, which interpolated
  // the user-controlled meeting title unescaped — a stored-XSS sink
  // (PoC: title `<img src=x onerror=alert(1)>` fired the alert).
  // The new implementation uses Svelte reactive state + `{...}` text
  // interpolation in the template below, which auto-escapes HTML so
  // malicious payloads render as literal text, never as DOM elements.
  type NextMeeting = { title: string; scheduledStart?: string; scheduled_start?: string };
  let upcomingCount: number | null = $state(null);
  let nextMeeting: NextMeeting | null = $state(null);
  let loadError = $state(false);

  $effect(() => {
    if (!ffEnableUpcomingMeetings) return;
    fetch('/api/upcoming/dashboard-count')
      .then(r => (r.ok ? r.json() : null))
      .then((data: { count?: number; nextMeeting?: NextMeeting | null } | null) => {
        if (!data) return;
        upcomingCount = data.count ?? 0;
        nextMeeting = data.nextMeeting ?? null;
      })
      .catch(() => {
        loadError = true;
      });
  });

  function formatRelativeDay(iso: string | undefined): string {
    if (!iso) return '';
    const d = new Date(iso);
    const today = new Date();
    const tomorrow = new Date();
    tomorrow.setDate(today.getDate() + 1);
    if (d.toDateString() === today.toDateString()) return 'Today';
    if (d.toDateString() === tomorrow.toDateString()) return 'Tomorrow';
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }

  // Derived strings consumed by `{#if}` branches below. Svelte's `{...}`
  // text interpolation will textContent-set these — no innerHTML anywhere.
  const countDisplay = $derived.by(() => {
    if (loadError || upcomingCount === null) return '—';
    return String(upcomingCount);
  });

  const hintText = $derived.by(() => {
    if (loadError) return 'Unable to load';
    if (upcomingCount === null) return 'Loading…';
    if (upcomingCount === 0) return 'No meetings scheduled';
    if (upcomingCount === 1 && nextMeeting) {
      const when = formatRelativeDay(nextMeeting.scheduledStart ?? nextMeeting.scheduled_start);
      return `Next: ${nextMeeting.title} — ${when}`;
    }
    return 'Meetings scheduled';
  });
</script>

<svelte:head>
  <title>{appShellEnabled ? 'Dashboard' : 'ContextPilot'} — Meeting Intelligence</title>
</svelte:head>

<div class="page-container">
  {#if !appShellEnabled}
    <!-- Landing / disabled state (app shell not enabled) -->
    <section class="landing" aria-labelledby="landing-heading">
      <div class="landing__hero">
        <h1 id="landing-heading" class="landing__title">Meeting context, when you need it most</h1>
        <p class="landing__subtitle">
          ContextPilot surfaces relevant background from your previous conversations
          so you can walk into every meeting prepared.
        </p>
        <div class="landing__actions">
          <Button variant="primary" onclick={() => window.location.href = '/signup'}>
            Get started
          </Button>
          <Button variant="secondary" onclick={() => window.location.href = '/login'}>
            Sign in
          </Button>
        </div>
      </div>

      <div class="landing__features">
        <Card>
          <h2 class="landing__feature-title">Pre-call briefings</h2>
          <p class="landing__feature-desc">Get a summary of past discussions with a specific contact before your next call.</p>
        </Card>
        <Card>
          <h2 class="landing__feature-title">Decision tracking</h2>
          <p class="landing__feature-desc">Never lose track of commitments made in meetings. ContextPilot remembers.</p>
        </Card>
        <Card>
          <h2 class="landing__feature-title">Action items</h2>
          <p class="landing__feature-desc">Keep todos front and center — who said what, and what's due.</p>
        </Card>
      </div>

      {#if appShellEnabled === false}
        <Alert variant="info">
          App shell is in development. Set <code>FF_ENABLE_APP_SHELL=true</code> and <code>VITE_FF_ENABLE_APP_SHELL=true</code> to enable full access.
        </Alert>
      {/if}
    </section>

  {:else}
    <!-- Dashboard (authenticated, app shell enabled) -->
    <section class="dashboard" aria-labelledby="dashboard-heading">
      <header class="dashboard__header">
        <h1 id="dashboard-heading" class="dashboard__title">Dashboard</h1>
        <Button variant="primary" onclick={() => window.location.href = '/meetings/new'}>
          New meeting
        </Button>
      </header>

      <!-- Upcoming meetings count — FR-21 dashboard preparation entry -->
      <div class="dashboard__stats">
        <Card>
          <p class="stat__label">Upcoming meetings</p>
          <p class="stat__value" id="upcoming-count" aria-live="polite">{countDisplay}</p>
          <p class="stat__hint" id="upcoming-hint">{hintText}</p>
        </Card>
        <Card>
          <p class="stat__label">Briefings due</p>
          <p class="stat__value">0</p>
        </Card>
        <Card>
          <p class="stat__label">Action items</p>
          <p class="stat__value">0</p>
        </Card>
      </div>

      <div class="dashboard__section">
        <div class="dashboard__section-header">
          <h2 class="dashboard__section-title">Upcoming meetings</h2>
          {#if ffEnableUpcomingMeetings}
            <a href="/upcoming" class="dashboard__see-all">View all</a>
          {:else}
            <span class="dashboard__see-all dashboard__see-all--disabled">View all</span>
          {/if}
        </div>
        {#if ffEnableUpcomingMeetings}
          <!--
            F1 (CWE-79) repair: was `cardEl.innerHTML = ...` interpolating
            `next.title` unescaped. Now rendered through Svelte `{#if}` branches
            with `{...}` text interpolation, which auto-escapes HTML.
          -->
          <div id="upcoming-meetings-card">
            {#if loadError}
              <p class="empty-hint">Unable to load upcoming meetings.</p>
            {:else if upcomingCount === null}
              <p class="empty-hint">Loading…</p>
            {:else if upcomingCount === 0}
              <p class="empty-hint">No upcoming meetings. <a href="/upcoming/new">Schedule one</a> to get started.</p>
            {:else if upcomingCount === 1 && nextMeeting}
              <p class="next-meeting-hint">{nextMeeting.title} <span class="muted">— {formatRelativeDay(nextMeeting.scheduledStart ?? nextMeeting.scheduled_start)}</span></p>
            {:else}
              <p class="next-meeting-hint">{upcomingCount} meeting{upcomingCount !== 1 ? 's' : ''} scheduled</p>
            {/if}
          </div>
        {:else}
          <Alert variant="info">
            Upcoming meetings are not enabled.
          </Alert>
        {/if}
      </div>
    </section>
  {/if}
</div>

<style>
  .page-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-6);
  }

  /* ── Landing ── */
  .landing {
    display: flex;
    flex-direction: column;
    gap: var(--space-8);
    padding-top: var(--space-8);
  }

  .landing__hero {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    max-width: 640px;
  }

  .landing__title {
    font-size: var(--text-3xl);
    font-weight: 700;
    color: var(--color-text-primary);
    line-height: 1.2;
    margin: 0;
  }

  .landing__subtitle {
    font-size: var(--text-lg);
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.6;
  }

  .landing__actions {
    display: flex;
    gap: var(--space-3);
    flex-wrap: wrap;
    margin-top: var(--space-2);
  }

  .landing__features {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: var(--space-4);
  }

  .landing__feature-title {
    font-size: var(--text-xl);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0 0 var(--space-2);
  }

  .landing__feature-desc {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.6;
  }

  /* ── Dashboard ── */
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .dashboard__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    flex-wrap: wrap;
  }

  .dashboard__title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .dashboard__stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: var(--space-4);
  }

  .stat__label {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    margin: 0 0 var(--space-1);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 500;
  }

  .stat__value {
    font-size: var(--text-3xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
    line-height: 1;
  }

  .stat__hint {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    margin: var(--space-1) 0 0;
  }

  .dashboard__section {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .dashboard__section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .dashboard__section-title {
    font-size: var(--text-xl);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .dashboard__see-all {
    font-size: var(--text-sm);
    color: var(--color-primary);
    text-decoration: none;
    font-weight: 500;
  }

  .dashboard__see-all:hover {
    text-decoration: underline;
  }

  .dashboard__see-all:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-sm);
  }

  .dashboard__see-all--disabled {
    color: var(--color-text-muted);
    cursor: default;
    pointer-events: none;
  }

  .empty-hint {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin: 0;
  }

  .next-meeting-hint {
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    margin: 0;
  }

  .muted {
    color: var(--color-text-muted);
  }
</style>