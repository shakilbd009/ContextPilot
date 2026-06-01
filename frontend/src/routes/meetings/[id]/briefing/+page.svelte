<script lang="ts">
  import { page } from '$app/stores';
  import { Button, Card, Alert, Spinner } from '$lib/components/ui';
  import PreCallBriefing from '$lib/components/briefing/PreCallBriefing.svelte';
  import BriefingVersionHistory from '$lib/components/briefing/BriefingVersionHistory.svelte';
  import {
    getActiveBriefing,
    getBriefingVersions,
    regenerateBriefing,
    excludeBriefingSource,
    restoreBriefingSource,
    type BriefingActive,
    type BriefingVersionSummary,
  } from '$lib/api/briefing';

  // Feature flag — detect server/browser mismatch using dual namespace.
  // Browser: VITE_FF_ENABLE_PRE_CALL_BRIEFING (embedded at build time)
  // Server:  FF_ENABLE_PRE_CALL_BRIEFING (passed via +page.ts data)
  // flagMismatch fires when one is true and the other is false.
  let { data } = $props();

  const briefingEnabled = import.meta.env.VITE_FF_ENABLE_PRE_CALL_BRIEFING === 'true';
  const flagMismatch = $derived(data.serverPreCallBriefingEnabled !== briefingEnabled);

  const meetingId = $derived($page.params.id ?? '');

  let briefing = $state<BriefingActive | null>(null);
  let versions = $state<BriefingVersionSummary[]>([]);
  let loadingBriefing = $state(false);
  let loadingVersions = $state(false);
  let regenerating = $state(false);
  let regenerateError = $state<string | null>(null);
  let activeSection = $state<'briefing' | 'history'>('briefing');

  async function loadBriefing() {
    loadingBriefing = true;
    const res = await getActiveBriefing(meetingId, fetch);
    loadingBriefing = false;
    if (res.ok) {
      briefing = res.data;
    }
  }

  async function loadVersions() {
    loadingVersions = true;
    const res = await getBriefingVersions(meetingId, fetch);
    loadingVersions = false;
    if (res.ok) {
      versions = res.data;
    }
  }

  async function handleRegenerate() {
    regenerating = true;
    regenerateError = null;
    const res = await regenerateBriefing(meetingId, fetch);
    regenerating = false;
    if (!res.ok) {
      regenerateError = res.message;
    } else {
      await loadBriefing();
    }
  }

  async function handleExcludeSource(sourceId: string) {
    await excludeBriefingSource(meetingId, sourceId, fetch);
    await loadBriefing();
  }

  async function handleRestoreSource(sourceId: string) {
    await restoreBriefingSource(meetingId, sourceId, fetch);
    await loadBriefing();
  }

  async function handleResolveConflict(conflictId: string, resolutionNote: string) {
    // POST to /api/upcoming/{meetingId}/briefing/conflicts/{conflictId}/resolve
    const res = await fetch(`/api/upcoming/${meetingId}/briefing/conflicts/${conflictId}/resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ resolution_note: resolutionNote }),
    });
    if (res.ok) {
      await loadBriefing();
    }
  }

  async function handleSelectVersion(versionNumber: number) {
    alert(`View version ${versionNumber} — full version diff is out of scope for BRD-04 MVP`);
    void versionNumber;
  }

  // Initial load
  $effect(() => {
    if (meetingId) {
      loadBriefing();
      loadVersions();
    }
  });
</script>

<svelte:head>
  <title>Pre-call Briefing — ContextPilot</title>
</svelte:head>

<div class="page-container">
  <header class="page-header">
    <div class="page-header__left">
      <h1 class="page-title">Pre-call Briefing</h1>
      <Button variant="secondary" size="sm" onclick={() => window.location.href = `/meetings/${meetingId}`}>
        Back to meeting
      </Button>
    </div>
  </header>

  {#if !briefingEnabled}
    <Alert variant="info">
      Pre-call briefing is not yet enabled. Enable <code>VITE_FF_ENABLE_PRE_CALL_BRIEFING</code> to unlock this feature.
    </Alert>
  {:else if loadingBriefing}
    <div class="loading-state">
      <Spinner />
      <span>Loading briefing...</span>
    </div>
  {:else if briefing}
    <!-- Tab navigation -->
    <div class="briefing-tabs" role="tablist" aria-label="Briefing sections">
      <button
        role="tab"
        aria-selected={activeSection === 'briefing'}
        class="briefing-tab"
        class:briefing-tab--active={activeSection === 'briefing'}
        onclick={() => { activeSection = 'briefing'; loadBriefing(); }}
      >
        Briefing
      </button>
      <button
        role="tab"
        aria-selected={activeSection === 'history'}
        class="briefing-tab"
        class:briefing-tab--active={activeSection === 'history'}
        onclick={() => { activeSection = 'history'; loadVersions(); }}
      >
        Version history
        {#if versions.length > 0}
          <span class="briefing-tab__badge">{versions.length}</span>
        {/if}
      </button>
    </div>

    {#if activeSection === 'briefing'}
      {#if regenerateError}
        <Alert variant="error" dismissible>
          Failed to regenerate: {regenerateError}
        </Alert>
      {/if}

      <PreCallBriefing
        {briefing}
        onRegenerate={handleRegenerate}
        onExcludeSource={handleExcludeSource}
        onRestoreSource={handleRestoreSource}
        onResolveConflict={handleResolveConflict}
        {regenerating}
        {flagMismatch}
      />
    {:else}
      {#if loadingVersions}
        <div class="loading-state">
          <Spinner />
          <span>Loading version history...</span>
        </div>
      {:else}
        <Card>
          <BriefingVersionHistory
            {versions}
            activeVersionNumber={briefing?.version.version_number ?? null}
            onSelectVersion={handleSelectVersion}
          />
        </Card>
      {/if}
    {/if}
  {:else}
    <Card>
      <p class="placeholder-text">
        No briefing is available yet. Make sure this meeting has related prior meetings, then refresh.
      </p>
    </Card>
  {/if}
</div>

<style>
  .page-container {
    max-width: 900px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    flex-wrap: wrap;
    margin-bottom: var(--space-2);
  }

  .page-header__left {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-wrap: wrap;
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .briefing-tabs {
    display: flex;
    gap: var(--space-1);
    border-bottom: 1px solid var(--color-border);
  }

  .briefing-tab {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3);
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-secondary);
    cursor: pointer;
    font-family: var(--font-sans);
    transition: color var(--transition-fast), border-color var(--transition-fast);
    margin-bottom: -1px;
  }

  .briefing-tab:hover {
    color: var(--color-text-primary);
  }

  .briefing-tab--active {
    color: var(--color-primary);
    border-bottom-color: var(--color-primary);
  }

  .briefing-tab:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-sm);
  }

  .briefing-tab__badge {
    background: var(--color-background-subtle);
    color: var(--color-text-muted);
    font-size: var(--text-xs);
    font-weight: 400;
    padding: 0.0625rem var(--space-1);
    border-radius: var(--radius-full);
    min-width: 1.25rem;
    text-align: center;
  }

  .loading-state {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-8);
    justify-content: center;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }

  .placeholder-text {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    margin: 0;
    line-height: 1.6;
  }
</style>