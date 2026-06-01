<script lang="ts">
  import type { BriefingPreparationStatus } from '$lib/api/briefing';

  interface Props {
    status: BriefingPreparationStatus;
  }

  let { status }: Props = $props();

  const config: Record<BriefingPreparationStatus, { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral'; role: 'status' | 'alert'; ariaLive: 'polite' | 'assertive' }> = {
    generating: {
      label: 'Generating',
      variant: 'neutral',
      role: 'status',
      ariaLive: 'polite',
    },
    ready: {
      label: 'Ready',
      variant: 'success',
      role: 'status',
      ariaLive: 'polite',
    },
    ready_with_caveats: {
      label: 'Ready with caveats',
      variant: 'warning',
      role: 'status',
      ariaLive: 'polite',
    },
    no_prior_memory: {
      label: 'No prior memory',
      variant: 'neutral',
      role: 'status',
      ariaLive: 'polite',
    },
    stale: {
      label: 'Stale',
      variant: 'warning',
      role: 'status',
      ariaLive: 'polite',
    },
    failed: {
      label: 'Failed',
      variant: 'danger',
      role: 'alert',
      ariaLive: 'assertive',
    },
    regenerating: {
      label: 'Regenerating',
      variant: 'neutral',
      role: 'status',
      ariaLive: 'polite',
    },
  };

  const { label, variant, role, ariaLive } = $derived(config[status] ?? config.generating);

  const announcementText = $derived(
    status === 'generating' ? 'Briefing generation in progress' :
    status === 'stale' ? 'Briefing may be outdated; regeneration available' :
    status === 'failed' ? 'Briefing generation failed' :
    status === 'regenerating' ? 'Regeneration in progress' :
    ''
  );
</script>

<!-- ARIA live region for dynamic states per BRD-04 accessibility -->
<div
  class="stale-indicator stale-indicator--{variant}"
  role={role}
  aria-live={ariaLive}
  aria-atomic="true"
>
  {#if variant === 'warning'}
    <svg class="stale-indicator__icon" width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <path d="M7 2L12.5 11.5H1.5L7 2z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
      <path d="M7 5.5v3M7 9.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  {:else if variant === 'danger'}
    <svg class="stale-indicator__icon" width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
      <path d="M5.5 5.5l3 3M8.5 5.5l-3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  {:else if variant === 'neutral'}
    <svg class="stale-indicator__icon" width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
      <path d="M7 4.5v3M7 9v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  {:else}
    <svg class="stale-indicator__icon" width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
      <path d="M4.5 7l2 2 3-3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  {/if}
  <span class="stale-indicator__label">{label}</span>
</div>

<!-- Screen-reader announcement text (visually hidden) -->
{#if announcementText}
  <span class="sr-only" role="status" aria-live="polite">{announcementText}</span>
{/if}

<style>
  .stale-indicator {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: 0.125rem var(--space-2);
    border-radius: var(--radius-full);
    font-size: var(--text-xs);
    font-weight: 600;
    line-height: 1.5;
    white-space: nowrap;
  }

  .stale-indicator--success {
    background: rgb(5 150 105 / 0.1);
    color: var(--color-success);
  }

  .stale-indicator--warning {
    background: rgb(217 119 6 / 0.1);
    color: var(--color-warning);
  }

  .stale-indicator--danger {
    background: rgb(220 38 38 / 0.1);
    color: var(--color-danger);
  }

  .stale-indicator--neutral {
    background: var(--color-background-subtle);
    color: var(--color-text-secondary);
  }

  .stale-indicator__icon {
    flex-shrink: 0;
  }

  .stale-indicator__label {
    font-family: var(--font-sans);
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
</style>