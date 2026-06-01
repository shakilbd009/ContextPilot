<script lang="ts">
  import { Badge, Button } from '$lib/components/ui';
  import type { BriefingSource, QualityStatus } from '$lib/api/briefing';

  interface Props {
    sources: BriefingSource[];
    onExclude: (sourceId: string) => void;
    excludedSourceIds?: Set<string>;
  }

  let { sources, onExclude, excludedSourceIds = new Set() }: Props = $props();

  let expanded = $state<Set<string>>(new Set());

  function toggleExpanded(sourceId: string) {
    if (expanded.has(sourceId)) {
      expanded.delete(sourceId);
    } else {
      expanded.add(sourceId);
    }
    expanded = new Set(expanded);
  }

  function qualityVariant(q: QualityStatus): 'success' | 'warning' | 'neutral' | 'danger' {
    switch (q) {
      case 'strong_evidence': return 'success';
      case 'weak_evidence': return 'warning';
      case 'insufficient_evidence': return 'neutral';
      case 'conflicting_evidence': return 'danger';
      default: return 'neutral';
    }
  }

  function qualityLabel(q: QualityStatus): string {
    switch (q) {
      case 'strong_evidence': return 'Strong';
      case 'weak_evidence': return 'Weak';
      case 'insufficient_evidence': return 'Insufficient';
      case 'conflicting_evidence': return 'Conflicting';
      default: return q;
    }
  }
</script>

<div class="source-list">
  <h4 class="source-list__heading">Source meetings</h4>

  {#if sources.length === 0}
    <p class="source-list__empty">No source meetings selected.</p>
  {:else}
    <ol class="source-list__items" role="list">
      {#each sources as source (source.source_meeting_id)}
        {@const isExcluded = excludedSourceIds.has(source.source_meeting_id)}
        {@const isExpanded = expanded.has(source.source_meeting_id)}
        <li class="source-item" class:source-item--excluded={isExcluded}>
          <div class="source-item__main">
            <button
              type="button"
              class="source-toggle"
              onclick={() => toggleExpanded(source.source_meeting_id)}
              aria-expanded={isExpanded}
            >
              <svg
                class="source-toggle__chevron"
                class:source-toggle__chevron--open={isExpanded}
                width="12"
                height="12"
                viewBox="0 0 12 12"
                fill="none"
                aria-hidden="true"
              >
                <path d="M3 4.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <span class="source-toggle__id">{source.source_meeting_id.slice(0, 8)}…</span>
              <Badge variant={qualityVariant(source.quality_status)}>{qualityLabel(source.quality_status)}</Badge>
            </button>

            <div class="source-item__actions">
              {#if !isExcluded}
                <Button
                  variant="ghost"
                  size="sm"
                  onclick={() => onExclude(source.source_meeting_id)}
                >
                  Exclude
                </Button>
              {:else}
                <span class="source-item__excluded-label">Excluded</span>
              {/if}
            </div>
          </div>

          {#if isExpanded && !isExcluded}
            <div class="source-item__details">
              <div class="source-item__reasons">
                <p class="source-item__reasons-label">Why this meeting was selected:</p>
                <ul class="reason-list" role="list">
                  {#each source.relatedness_reasons as reason}
                    <li class="reason-list__item">{reason}</li>
                  {/each}
                </ul>
              </div>
            </div>
          {/if}
        </li>
      {/each}
    </ol>
  {/if}
</div>

<style>
  .source-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .source-list__heading {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .source-list__empty {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin: 0;
    padding: var(--space-2) 0;
  }

  .source-list__items {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .source-item {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    overflow: hidden;
  }

  .source-item--excluded {
    opacity: 0.5;
  }

  .source-item__main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-2) var(--space-3);
    background: var(--color-background);
    gap: var(--space-2);
  }

  .source-toggle {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: var(--font-sans);
    border-radius: var(--radius-sm);
  }

  .source-toggle:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .source-toggle__chevron {
    flex-shrink: 0;
    transition: transform var(--transition-fast);
  }

  .source-toggle__chevron:not(.source-toggle__chevron--open) {
    transform: rotate(-90deg);
  }

  .source-toggle__chevron--open {
    transform: rotate(0deg);
  }

  .source-toggle__id {
    font-size: var(--text-xs);
    font-family: var(--font-mono);
    color: var(--color-text-secondary);
  }

  .source-item__actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .source-item__excluded-label {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    font-style: italic;
  }

  .source-item__details {
    padding: var(--space-3);
    border-top: 1px solid var(--color-border);
    background: var(--color-background-subtle);
  }

  .source-item__reasons-label {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--color-text-muted);
    margin: 0 0 var(--space-1) 0;
  }

  .reason-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .reason-list__item {
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    padding-left: var(--space-3);
    position: relative;
  }

  .reason-list__item::before {
    content: '–';
    position: absolute;
    left: 0;
    color: var(--color-text-muted);
  }
</style>