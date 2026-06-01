<script lang="ts">
  import { page } from '$app/stores';
  import { Button, Badge, Card, Alert, Spinner } from '$lib/components/ui';
  import MemoryView from '$lib/components/memory/MemoryView.svelte';
  import BriefingReadinessIndicator from '$lib/components/memory/BriefingReadinessIndicator.svelte';
  import PriorMemoryMatches from '$lib/components/memory/PriorMemoryMatches.svelte';
  import ConflictResolutionUI from '$lib/components/memory/ConflictResolutionUI.svelte';
  import MemoryVersionHistory from '$lib/components/memory/MemoryVersionHistory.svelte';
  import {
    getActiveMemory,
    getMemoryState,
    getMemoryVersions,
    getMemoryConflicts,
    reprocessMemory,
    resolveConflict,
    removePriorMemoryMatch,
    type MemoryState,
    type MemoryContent,
    type MemoryVersionSummary,
    type MemoryConflict as MemoryConflictType,
    type PriorMemoryInput,
    type BriefingReadinessSignal,
  } from '$lib/api/memory';

  // Feature flag gate — both must be enabled to show the memory section
  const memoryEnabled = import.meta.env.VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING === 'true';
  const briefingEnabled = import.meta.env.VITE_FF_ENABLE_PRE_CALL_BRIEFING === 'true';

  interface Props {
    data: { meetingId: string; meeting?: { id: string; title: string } | null };
  }

  let { data }: Props = $props();

  const meetingId = $derived($page.params.id ?? data.meetingId);

  // Memory UI state
  let memoryState = $state<MemoryState | null>(null);
  let briefingSignal = $state<BriefingReadinessSignal>('processing');
  let memoryContent = $state<MemoryContent | null>(null);
  let priorInputs = $state<PriorMemoryInput[]>([]);
  let versions = $state<MemoryVersionSummary[]>([]);
  let conflicts = $state<MemoryConflictType[]>([]);
  let activeVersionNumber = $state<number | null>(null);

  let loadingMemory = $state(false);
  let loadingVersions = $state(false);
  let loadingConflicts = $state(false);
  let reprocessing = $state(false);
  let reprocessError = $state<string | null>(null);
  let activeSection = $state<'memory' | 'conflicts' | 'history'>('memory');

  async function loadMemory() {
    loadingMemory = true;
    const res = await getActiveMemory(meetingId, fetch);
    loadingMemory = false;

    if (res.ok) {
      memoryState = res.data.state as MemoryState;
      briefingSignal = res.data.briefing_readiness_signal;
      memoryContent = res.data.memory.content as MemoryContent;
      activeVersionNumber = res.data.memory.version_number;
    }
  }

  async function loadVersions() {
    loadingVersions = true;
    const res = await getMemoryVersions(meetingId, fetch);
    loadingVersions = false;
    if (res.ok) {
      versions = res.data;
    }
  }

  async function loadConflicts() {
    loadingConflicts = true;
    const res = await getMemoryConflicts(meetingId, fetch);
    loadingConflicts = false;
    if (res.ok) {
      conflicts = res.data;
    }
  }

  async function handleReprocess() {
    reprocessing = true;
    reprocessError = null;
    const res = await reprocessMemory(meetingId, fetch);
    reprocessing = false;
    if (!res.ok) {
      reprocessError = res.message;
    } else {
      await loadMemory();
    }
  }

  async function handleResolveConflict(conflictId: string, resolutionNote: string) {
    const res = await resolveConflict(meetingId, conflictId, resolutionNote, fetch);
    if (res.ok) {
      await loadMemory();
      await loadConflicts();
    }
  }

  async function handleExcludePrior(priorMemoryVersionId: string) {
    await removePriorMemoryMatch(meetingId, priorMemoryVersionId, fetch);
    await loadMemory();
  }

  const reprocessableStates: MemoryState[] = ['failed', 'retry_exhausted', 'stale', 'completed_with_insufficient_evidence'];
  const canReprocess = $derived(memoryState && reprocessableStates.includes(memoryState));
</script>

<svelte:head>
  <title>Meeting Detail — ContextPilot</title>
</svelte:head>

<div class="page-container">
  <header class="page-header">
    <h1 class="page-title">{data.meeting?.title ?? 'Meeting Detail'}</h1>
    <div class="page-actions">
      {#if memoryEnabled && canReprocess}
        <Button
          variant="secondary"
          loading={reprocessing}
          onclick={handleReprocess}
        >
          {reprocessing ? 'Reprocessing...' : 'Reprocess memory'}
        </Button>
      {/if}
      {#if briefingEnabled}
        <Button variant="secondary" onclick={() => window.location.href = `/meetings/${meetingId}/briefing`}>
          View briefing
        </Button>
      {/if}
    </div>
  </header>

  {#if reprocessError}
    <Alert variant="error" dismissible>
      Failed to reprocess: {reprocessError}
    </Alert>
  {/if}

  {#if !memoryEnabled}
    <Alert variant="info">
      Memory processing is not enabled. Enable <code>VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING</code> to view memory.
    </Alert>
  {:else}
    <!-- Memory section tabs -->
    <div class="memory-tabs" role="tablist" aria-label="Memory sections">
      <button
        role="tab"
        aria-selected={activeSection === 'memory'}
        class="memory-tab"
        class:memory-tab--active={activeSection === 'memory'}
        onclick={() => { activeSection = 'memory'; loadMemory(); }}
      >
        Memory
      </button>
      <button
        role="tab"
        aria-selected={activeSection === 'conflicts'}
        class="memory-tab"
        class:memory-tab--active={activeSection === 'conflicts'}
        onclick={() => { activeSection = 'conflicts'; loadConflicts(); }}
      >
        Conflicts
        {#if conflicts.length > 0}
          <span class="memory-tab__badge">{conflicts.length}</span>
        {/if}
      </button>
      <button
        role="tab"
        aria-selected={activeSection === 'history'}
        class="memory-tab"
        class:memory-tab--active={activeSection === 'history'}
        onclick={() => { activeSection = 'history'; loadVersions(); }}
      >
        Version history
      </button>
    </div>

    {#if activeSection === 'memory'}
      {#if loadingMemory}
        <div class="loading-state">
          <Spinner />
          <span>Loading memory...</span>
        </div>
      {:else if memoryContent}
        <!-- State badge -->
        {#if memoryState}
          <div class="memory-state-badge">
            <span class="memory-state-badge__label">Status</span>
            <Badge
              variant={
                memoryState === 'completed' ? 'success' :
                memoryState === 'failed' ? 'danger' :
                memoryState === 'stale' ? 'warning' :
                'neutral'
              }
            >
              {memoryState.replace(/_/g, ' ')}
            </Badge>
          </div>
        {/if}

        <!-- Briefing readiness -->
        <BriefingReadinessIndicator
          signal={briefingSignal}
          coreCategoriesReady={memoryContent.briefing_readiness.core_categories_ready}
          blockingConflictCount={memoryContent.briefing_readiness.blocking_conflicts.length}
          insufficientCategoryCount={memoryContent.briefing_readiness.insufficient_categories.length}
        />

        <!-- Prior memory matches -->
        {#if priorInputs.length > 0}
          <Card>
            <PriorMemoryMatches
              matches={priorInputs}
              onExclude={handleExcludePrior}
            />
          </Card>
        {/if}

        <!-- 7-category memory view -->
        <MemoryView content={memoryContent} meetingId={meetingId} />
      {:else}
        <Card>
          <p class="placeholder-text">
            No memory available yet. Process a meeting to generate memory.
          </p>
        </Card>
      {/if}
    {:else if activeSection === 'conflicts'}
      {#if loadingConflicts}
        <div class="loading-state">
          <Spinner />
          <span>Loading conflicts...</span>
        </div>
      {:else}
        <ConflictResolutionUI
          conflicts={conflicts}
          onResolve={handleResolveConflict}
        />
      {/if}
    {:else if activeSection === 'history'}
      {#if loadingVersions}
        <div class="loading-state">
          <Spinner />
          <span>Loading version history...</span>
        </div>
      {:else}
        <MemoryVersionHistory
          versions={versions}
          {activeVersionNumber}
          onSelectVersion={(vn) => { alert(`View version ${vn} — full diff UI out of scope for BRD-03 UI`); }}
        />
      {/if}
    {/if}
  {/if}
</div>

<style>
  .page-container {
    max-width: 1200px;
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
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .page-actions {
    display: flex;
    gap: var(--space-2);
  }

  .memory-tabs {
    display: flex;
    gap: var(--space-1);
    border-bottom: 1px solid var(--color-border);
    margin-bottom: var(--space-2);
  }

  .memory-tab {
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

  .memory-tab:hover {
    color: var(--color-text-primary);
  }

  .memory-tab--active {
    color: var(--color-primary);
    border-bottom-color: var(--color-primary);
  }

  .memory-tab:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-sm);
  }

  .memory-tab__badge {
    background: var(--color-danger);
    color: white;
    font-size: var(--text-xs);
    font-weight: 600;
    padding: 0.0625rem var(--space-1);
    border-radius: var(--radius-full);
    min-width: 1.25rem;
    text-align: center;
  }

  .memory-state-badge {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .memory-state-badge__label {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--color-text-muted);
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
  }
</style>