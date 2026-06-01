<script lang="ts">
  import type { EvidenceItem } from '$lib/api/memory';
  import { escapeHtml } from '$lib/utils/html';

  interface Props {
    evidence: EvidenceItem[];
    collapsed?: boolean;
  }

  let { evidence, collapsed = true }: Props = $props();

  // svelte-ignore state_referenced_locally — open is intentionally initialized once from collapsed, then user-toggled
  let open = $state(!collapsed);

  const sourceTypeLabel: Record<string, string> = {
    transcript: 'Transcript',
    notes: 'Notes',
    prior_memory_reference: 'Prior Memory',
  };

  const sourceTypeColor: Record<string, string> = {
    transcript: 'neutral',
    notes: 'success',
    prior_memory_reference: 'warning',
  } as const;
</script>

{#if evidence.length > 0}
  <div class="evidence">
    <button
      type="button"
      class="evidence__toggle"
      onclick={() => (open = !open)}
      aria-expanded={open}
    >
      <svg
        class="evidence__chevron"
        class:evidence__chevron--open={open}
        width="12"
        height="12"
        viewBox="0 0 12 12"
        fill="none"
        aria-hidden="true"
      >
        <path d="M3 4.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
      {evidence.length} evidence {evidence.length === 1 ? 'source' : 'sources'}
    </button>

    {#if open}
      <ul class="evidence__list" role="list">
        {#each evidence as item}
          <li class="evidence__item">
            <div class="evidence__meta">
              <span class="evidence__source-type evidence__source-type--{sourceTypeColor[item.source_type] ?? 'neutral'}">
                {sourceTypeLabel[item.source_type] ?? item.source_type}
              </span>
              {#if item.source_location}
                <span class="evidence__offset">
                  chars {item.source_location.start}–{item.source_location.end}
                </span>
              {/if}
            </div>
            <blockquote class="evidence__snippet">
              {escapeHtml(item.snippet)}
            </blockquote>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
{/if}

<style>
  .evidence {
    margin-top: var(--space-2);
  }

  .evidence__toggle {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    background: none;
    border: none;
    padding: 0;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    cursor: pointer;
    font-family: var(--font-sans);
    border-radius: var(--radius-sm);
    transition: color var(--transition-fast);
  }

  .evidence__toggle:hover {
    color: var(--color-text-secondary);
  }

  .evidence__toggle:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .evidence__chevron {
    transition: transform var(--transition-fast);
    flex-shrink: 0;
  }

  .evidence__chevron--open {
    transform: rotate(0deg);
  }

  .evidence__chevron:not(.evidence__chevron--open) {
    transform: rotate(-90deg);
  }

  .evidence__list {
    list-style: none;
    margin: var(--space-2) 0 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .evidence__item {
    border-left: 2px solid var(--color-border);
    padding-left: var(--space-3);
  }

  .evidence__meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-1);
  }

  .evidence__source-type {
    font-size: var(--text-xs);
    font-weight: 600;
    padding: 0.0625rem var(--space-2);
    border-radius: var(--radius-full);
  }

  .evidence__source-type--neutral {
    background: var(--color-background-subtle);
    color: var(--color-text-secondary);
  }

  .evidence__source-type--success {
    background: rgb(5 150 105 / 0.1);
    color: var(--color-success);
  }

  .evidence__source-type--warning {
    background: rgb(217 119 6 / 0.1);
    color: var(--color-warning);
  }

  .evidence__offset {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    font-family: var(--font-mono);
  }

  .evidence__snippet {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    font-style: italic;
    line-height: 1.5;
  }
</style>
