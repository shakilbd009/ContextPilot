<script lang="ts">
  import type { PageData } from './$types';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { Button, Alert } from '$lib/components/ui';

  let { data }: { data: PageData } = $props();

  // Server-side auth guard ensures only authenticated users with ffEnableAppShell=true reach here
  // (hooks.server.ts handles the redirect)

  const ffEnableManualMeetingImport =
    import.meta.env.VITE_FF_ENABLE_MANUAL_MEETING_IMPORT === 'true';

  // ── Derived state ───────────────────────────────────────────────
  const meetings = $derived(data.meetings ?? []);
  const total = $derived(data.total ?? 0);
  const currentPage = $derived(data.page ?? 1);
  const limit = $derived(data.limit ?? 20);
  const hasMore = $derived(data.hasMore ?? false);
  const loading = $derived(data.loading ?? false);
  const error = $derived(data.error ?? null);

  const isViewAll = $derived(
    page.url.searchParams.has('viewAll')
  );

  const showPagination = $derived(
    total > limit && !isViewAll
  );

  const canGoPrev = $derived(currentPage > 1 && !isViewAll);
  const canGoNext = $derived(hasMore && !isViewAll);

  // ── Helpers ─────────────────────────────────────────────────────
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

  function navigateTo(p: number) {
    goto(`/meetings?page=${p}`, { replaceState: false });
  }

  function viewAll() {
    goto('/meetings?viewAll=1', { replaceState: false });
  }

  function goPrev() {
    if (canGoPrev) navigateTo(currentPage - 1);
  }

  function goNext() {
    if (canGoNext) navigateTo(currentPage + 1);
  }
</script>

<svelte:head>
  <title>Meetings — ContextPilot</title>
</svelte:head>

<div class="page-container">
  <header class="page-header">
    <h1 class="page-title">Meetings</h1>
    {#if ffEnableManualMeetingImport}
      <Button variant="primary" onclick={() => goto('/meetings/new')}>
        Import meeting
      </Button>
    {/if}
  </header>

  <!-- ── Error ──────────────────────────────────────────────────── -->
  {#if error}
    <Alert variant="error">
      {error}
    </Alert>

  <!-- ── Loading skeleton ────────────────────────────────────────── -->
  {:else if loading}
    <div class="meetings-list" aria-label="Meeting list" aria-busy="true">
      {#each Array(limit) as _, i}
        <div class="meeting-skeleton" aria-hidden="true">
          <div class="skeleton__title"></div>
          <div class="skeleton__meta"></div>
        </div>
      {/each}
    </div>

  <!-- ── Empty state ───────────────────────────────────────────── -->
  {:else if meetings.length === 0}
    <Alert variant="info">
      No meetings yet.{#if ffEnableManualMeetingImport}
        <a href="/meetings/new">Import one</a> to get started.
      {/if}
    </Alert>

  <!-- ── Meeting list ───────────────────────────────────────────── -->
  {:else}
    <div class="meetings-list" aria-label="Meeting list">
      {#each meetings as meeting (meeting.id)}
        <div class="meeting-card">
          <a href="/meetings/{meeting.id}" class="meeting-link">
            <div class="meeting-card__content">
              <h2 class="meeting-card__title">{meeting.title}</h2>
              <div class="meeting-card__meta">
                <span class="meeting-card__date">
                  {formatDate(meeting.completedAt)}
                </span>
                {#if meeting.participants?.length > 0}
                  <span class="meeting-card__participants">
                    {meeting.participants.length}
                    participant{meeting.participants.length !== 1 ? 's' : ''}
                  </span>
                {/if}
              </div>
            </div>
            <span class="meeting-card__arrow" aria-hidden="true">→</span>
          </a>
        </div>
      {/each}
    </div>

    <!-- ── Pagination controls ──────────────────────────────────── -->
    {#if showPagination}
      <nav class="pagination" aria-label="Meeting list pagination">
        <Button
          variant="secondary"
          size="sm"
          disabled={!canGoPrev}
          onclick={goPrev}
          aria-label="Previous page"
        >
          ← Previous
        </Button>

        <span class="pagination__info" aria-live="polite">
          Page {currentPage}
          {#if total > 0}
            <span class="pagination__count">
              ({meetings.length} of {total} meetings)
            </span>
          {/if}
        </span>

        <Button
          variant="secondary"
          size="sm"
          disabled={!canGoNext}
          onclick={goNext}
          aria-label="Next page"
        >
          Next →
        </Button>
      </nav>

      <!-- View All option when there are more than one page -->
      {#if total > limit}
        <div class="view-all">
          <Button variant="ghost" size="sm" onclick={viewAll}>
            View all {total} meetings
          </Button>
        </div>
      {/if}

    <!-- ── View All mode footer ──────────────────────────────────── -->
    {:else if isViewAll}
      <div class="view-all">
        <Alert variant="info">
          Showing all {total} meetings.
        </Alert>
        <Button variant="secondary" size="sm" onclick={() => navigateTo(1)}>
          Switch to paginated view
        </Button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .page-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-6);
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    margin-bottom: var(--space-6);
    flex-wrap: wrap;
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .meetings-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    margin-top: var(--space-6);
  }

  /* ── Meeting card ── */
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

  .meeting-link:hover .meeting-card__title {
    color: var(--color-primary);
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
    transition: color var(--transition-fast);
  }

  .meeting-card__meta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }

  .meeting-card__date {
    white-space: nowrap;
  }

  .meeting-card__participants {
    white-space: nowrap;
  }

  .meeting-card__arrow {
    font-size: var(--text-lg);
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  /* ── Loading skeleton ── */
  .meeting-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-5);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }

  .skeleton__title {
    height: 16px;
    width: 55%;
    background: var(--color-background-subtle);
    border-radius: var(--radius-xs);
    animation: pulse 1.5s ease-in-out infinite;
  }

  .skeleton__meta {
    height: 12px;
    width: 30%;
    background: var(--color-background-subtle);
    border-radius: var(--radius-xs);
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  /* ── Pagination ── */
  .pagination {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    margin-top: var(--space-8);
    flex-wrap: wrap;
  }

  .pagination__info {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    white-space: nowrap;
  }

  .pagination__count {
    color: var(--color-text-muted);
  }

  .view-all {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    margin-top: var(--space-4);
  }
</style>
