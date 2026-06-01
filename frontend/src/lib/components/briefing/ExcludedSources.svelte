<script lang="ts">
  import { Button } from '$lib/components/ui';
  import type { ExcludedSource } from '$lib/api/briefing';

  interface Props {
    excludedSources: ExcludedSource[];
    onRestore: (sourceId: string) => void;
  }

  let { excludedSources, onRestore }: Props = $props();

  let expanded = $state(false);
</script>

{#if excludedSources.length > 0}
  <div class="excluded-sources">
    <button
      type="button"
      class="excluded-sources__toggle"
      onclick={() => { expanded = !expanded; }}
      aria-expanded={expanded}
    >
      <svg
        class="excluded-sources__chevron"
        class:excluded-sources__chevron--open={expanded}
        width="12"
        height="12"
        viewBox="0 0 12 12"
        fill="none"
        aria-hidden="true"
      >
        <path d="M3 4.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      <span class="excluded-sources__label">
        {excludedSources.length} {excludedSources.length === 1 ? 'source' : 'sources'} excluded
      </span>
    </button>

    {#if expanded}
      <ul class="excluded-list" role="list">
        {#each excludedSources as source (source.source_meeting_id)}
          <li class="excluded-item">
            <div class="excluded-item__info">
              <span class="excluded-item__title">
                {source.prior_meeting_title ?? source.source_meeting_id.slice(0, 8)}
              </span>
              <span class="excluded-item__id">{source.source_meeting_id.slice(0, 8)}…</span>
            </div>
            <div class="excluded-item__reasons">
              {#each source.relatedness_reasons as reason}
                <span class="excluded-item__reason">{reason}</span>
              {/each}
            </div>
            <Button
              variant="ghost"
              size="sm"
              onclick={() => onRestore(source.source_meeting_id)}
            >
              Restore
            </Button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
{/if}

<style>
  .excluded-sources {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    overflow: hidden;
  }

  .excluded-sources__toggle {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    background: var(--color-background-subtle);
    border: none;
    padding: var(--space-2) var(--space-3);
    cursor: pointer;
    font-family: var(--font-sans);
    text-align: left;
  }

  .excluded-sources__toggle:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: -2px;
  }

  .excluded-sources__chevron {
    flex-shrink: 0;
    color: var(--color-text-muted);
    transition: transform var(--transition-fast);
  }

  .excluded-sources__chevron:not(.excluded-sources__chevron--open) {
    transform: rotate(-90deg);
  }

  .excluded-sources__chevron--open {
    transform: rotate(0deg);
  }

  .excluded-sources__label {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .excluded-list {
    list-style: none;
    margin: 0;
    padding: 0;
    border-top: 1px solid var(--color-border);
  }

  .excluded-item {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
    padding: var(--space-3);
    border-bottom: 1px solid var(--color-border);
    background: var(--color-background);
  }

  .excluded-item:last-child {
    border-bottom: none;
  }

  .excluded-item__info {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    flex: 1;
    min-width: 0;
  }

  .excluded-item__title {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .excluded-item__id {
    font-size: var(--text-xs);
    font-family: var(--font-mono);
    color: var(--color-text-muted);
  }

  .excluded-item__reasons {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    flex: 1;
  }

  .excluded-item__reason {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }
</style>