<script lang="ts">
  import { Badge } from '$lib/components/ui';
  import type { BriefingReadinessSignal } from '$lib/api/memory';

  interface Props {
    signal: BriefingReadinessSignal;
    coreCategoriesReady: boolean;
    blockingConflictCount: number;
    insufficientCategoryCount: number;
  }

  let {
    signal,
    coreCategoriesReady,
    blockingConflictCount,
    insufficientCategoryCount,
  }: Props = $props();

  const config: Record< BriefingReadinessSignal, { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral'; description: string } > = {
    ready: {
      label: 'Ready',
      variant: 'success',
      description: 'All core categories have evidence and no conflicts block this meeting.',
    },
    ready_with_weak: {
      label: 'Ready with weak',
      variant: 'warning',
      description: 'All core categories are present but some have weak evidence.',
    },
    ready_with_insufficient: {
      label: 'Ready with gaps',
      variant: 'warning',
      description: 'Core categories have insufficient evidence for some items.',
    },
    needs_review: {
      label: 'Needs review',
      variant: 'warning',
      description: 'Unresolved conflicts or missing core categories require attention.',
    },
    processing: {
      label: 'Processing',
      variant: 'neutral',
      description: 'Memory is being processed.',
    },
    reprocessing: {
      label: 'Reprocessing',
      variant: 'neutral',
      description: 'Memory is being reprocessed.',
    },
    failed: {
      label: 'Failed',
      variant: 'danger',
      description: 'Memory processing failed. Retry is available.',
    },
    retry_exhausted: {
      label: 'Retry exhausted',
      variant: 'danger',
      description: 'Automatic retries failed. Manual intervention may be needed.',
    },
    stale: {
      label: 'Stale',
      variant: 'warning',
      description: 'Source meeting changed. Reprocessing queued.',
    },
  };

  const { label, variant, description } = $derived(config[signal] ?? config.processing);
</script>

<div class="briefing-readiness" role="status" aria-live="polite">
  <div class="briefing-readiness__header">
    <div class="briefing-readiness__title-group">
      {#if variant === 'success'}
        <svg class="briefing-readiness__icon briefing-readiness__icon--success" width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
          <path d="M5 8l2 2 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      {:else if variant === 'warning'}
        <svg class="briefing-readiness__icon briefing-readiness__icon--warning" width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <path d="M8 2L14.5 13.5H1.5L8 2z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
          <path d="M8 6.5v3.5M8 11.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      {:else if variant === 'danger'}
        <svg class="briefing-readiness__icon briefing-readiness__icon--danger" width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
          <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      {:else}
        <svg class="briefing-readiness__icon briefing-readiness__icon--neutral" width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
          <path d="M8 5v3.5M8 10v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      {/if}
      <span class="briefing-readiness__label">Briefing readiness</span>
    </div>
    <Badge {variant}>{label}</Badge>
  </div>

  <p class="briefing-readiness__description">{description}</p>

  <ul class="briefing-readiness__signals" role="list">
    {#if blockingConflictCount > 0}
      <li class="signal signal--danger">
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <path d="M7 2L12.5 11.5H1.5L7 2z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
          <path d="M7 5.5v3M7 9.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        {blockingConflictCount} unresolved {blockingConflictCount === 1 ? 'conflict' : 'conflicts'} blocking briefing
      </li>
    {/if}

    {#if !coreCategoriesReady}
      <li class="signal signal--warning">
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
          <path d="M7 4v3.5M7 9v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        Core categories have insufficient evidence
      </li>
    {/if}

    {#if insufficientCategoryCount > 0 && signal === 'ready_with_insufficient'}
      <li class="signal signal--warning">
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
          <path d="M7 4v3.5M7 9v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        {insufficientCategoryCount} {insufficientCategoryCount === 1 ? 'category' : 'categories'} with insufficient evidence
      </li>
    {/if}

    {#if signal === 'ready' || signal === 'ready_with_weak'}
      <li class="signal signal--success">
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
          <path d="M4.5 7l2 2 3-3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        Eligible for pre-call briefing
      </li>
    {/if}
  </ul>
</div>

<style>
  .briefing-readiness {
    padding: var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-background);
  }

  .briefing-readiness__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    margin-bottom: var(--space-2);
  }

  .briefing-readiness__title-group {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .briefing-readiness__icon {
    flex-shrink: 0;
  }

  .briefing-readiness__icon--success {
    color: var(--color-success);
  }

  .briefing-readiness__icon--warning {
    color: var(--color-warning);
  }

  .briefing-readiness__icon--danger {
    color: var(--color-danger);
  }

  .briefing-readiness__icon--neutral {
    color: var(--color-text-muted);
  }

  .briefing-readiness__label {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .briefing-readiness__description {
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    margin: 0 0 var(--space-3) 0;
    line-height: 1.5;
  }

  .briefing-readiness__signals {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .signal {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-xs);
    line-height: 1.4;
  }

  .signal--success {
    color: var(--color-success);
  }

  .signal--warning {
    color: var(--color-warning);
  }

  .signal--danger {
    color: var(--color-danger);
  }
</style>