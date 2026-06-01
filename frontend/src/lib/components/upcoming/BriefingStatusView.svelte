<script lang="ts">
  import { Card, Badge, Button } from '$lib/components/ui';
  import type { BriefingStatusValue } from '$lib/api/upcoming';

  interface Props {
    briefingStatus?: BriefingStatusValue;
    isBriefingStale?: boolean;
    onNavigateToBriefing?: () => void;
  }

  let {
    briefingStatus,
    isBriefingStale = false,
    onNavigateToBriefing,
  }: Props = $props();

  // Nullish — BRD-04 not enabled or briefing not applicable
  const isAbsent = $derived(briefingStatus == null || briefingStatus === 'absent');

  // Status configuration matching backend determineBriefingStatus semantics
  const statusConfig: Record<
    BriefingStatusValue,
    { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral'; description: string }
  > = {
    queued: {
      label: 'Queued',
      variant: 'neutral',
      description: 'Briefing generation has been queued and will start shortly.',
    },
    generating: {
      label: 'Generating',
      variant: 'neutral',
      description: 'Briefing is currently being generated.',
    },
    ready: {
      label: 'Ready',
      variant: 'success',
      description: 'Briefing is ready for your pre-call preparation.',
    },
    failed: {
      label: 'Failed',
      variant: 'danger',
      description: 'Briefing generation failed. Please try again.',
    },
    stale: {
      label: 'Stale',
      variant: 'warning',
      description: 'Briefing is outdated due to new meeting activity.',
    },
    disabled: {
      label: 'Disabled',
      variant: 'neutral',
      description: 'Pre-call briefing is not enabled. Contact your administrator.',
    },
    unavailable: {
      label: 'Unavailable',
      variant: 'neutral',
      description: 'Briefing is temporarily unavailable. Check back later.',
    },
    unknown: {
      label: 'Unknown',
      variant: 'neutral',
      description: 'Briefing status could not be determined.',
    },
    absent: {
      label: 'Not started',
      variant: 'neutral',
      description: 'No briefing has been generated yet.',
    },
  };

  const config = $derived(
    briefingStatus != null && briefingStatus in statusConfig
      ? statusConfig[briefingStatus]
      : statusConfig.absent
  );
</script>

<Card>
  <div
    class="briefing-status"
    aria-labelledby="briefing-status-label"
  >
    <div class="briefing-status__header">
      <span
        class="briefing-status__label"
        id="briefing-status-label"
      >Pre-Call Briefing</span>
      {#if !isAbsent}
        <Badge variant={config.variant}>{config.label}</Badge>
      {/if}
    </div>

    {#if isAbsent}
      <!-- No briefing yet — show entry point -->
      <p class="briefing-status__description">
        No briefing has been generated yet. Create a briefing to get AI-powered pre-call preparation.
      </p>
      {#if onNavigateToBriefing}
        <div class="briefing-status__action">
          <Button variant="secondary" size="sm" onclick={onNavigateToBriefing}>
            Generate briefing
          </Button>
        </div>
      {/if}
    {:else}
      <p class="briefing-status__description">{config.description}</p>

      {#if briefingStatus === 'ready' && onNavigateToBriefing}
        <div class="briefing-status__action">
          <Button variant="secondary" size="sm" onclick={onNavigateToBriefing}>
            View briefing
          </Button>
        </div>
      {:else if briefingStatus === 'stale' && onNavigateToBriefing}
        <div class="briefing-status__action">
          <Button variant="secondary" size="sm" onclick={onNavigateToBriefing}>
            Refresh briefing
          </Button>
        </div>
      {:else if briefingStatus === 'failed' && onNavigateToBriefing}
        <div class="briefing-status__action">
          <Button variant="secondary" size="sm" onclick={onNavigateToBriefing}>
            Retry
          </Button>
        </div>
      {/if}

      {#if isBriefingStale}
        <p class="briefing-status__stale-note" role="alert">
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
            <path d="M6 1.5L10.5 9.5H1.5L6 1.5Z" stroke="currentColor" stroke-width="1.25" stroke-linejoin="round"/>
            <path d="M6 5v2.5" stroke="currentColor" stroke-width="1.25" stroke-linecap="round"/>
            <circle cx="6" cy="9" r="0.5" fill="currentColor"/>
          </svg>
          Source materials have been updated since this briefing was generated.
        </p>
      {/if}
    {/if}
  </div>
</Card>

<style>
  .briefing-status {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .briefing-status__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
  }

  .briefing-status__label {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--color-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .briefing-status__description {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.5;
  }

  .briefing-status__action {
    margin-top: var(--space-1);
  }

  .briefing-status__stale-note {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: var(--text-xs);
    color: var(--color-warning);
    margin: 0;
    font-style: italic;
  }
</style>
