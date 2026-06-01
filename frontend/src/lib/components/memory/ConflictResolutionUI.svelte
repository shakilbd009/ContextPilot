<script lang="ts">
  import { Button, Badge, Alert } from '$lib/components/ui';
  import { escapeHtml } from '$lib/utils/html';
  import type { MemoryConflict } from '$lib/api/memory';

  interface Props {
    conflicts: MemoryConflict[];
    onResolve: (conflictId: string, resolutionNote: string) => Promise<void>;
  }

  let { conflicts, onResolve }: Props = $props();

  let resolutions = $state<Record<string, string>>({});
  let submitting = $state<Set<string>>(new Set());

  async function handleResolve(conflictId: string) {
    const note = resolutions[conflictId]?.trim() ?? '';
    submitting.add(conflictId);
    submitting = new Set(submitting);
    try {
      await onResolve(conflictId, note);
      delete resolutions[conflictId];
      resolutions = { ...resolutions };
    } finally {
      submitting.delete(conflictId);
      submitting = new Set(submitting);
    }
  }

  function qualityVariant(q: string): 'success' | 'warning' | 'danger' | 'neutral' {
    switch (q) {
      case 'strong_evidence': return 'success';
      case 'weak_evidence': return 'warning';
      case 'insufficient_evidence': return 'neutral';
      case 'conflicting_evidence': return 'danger';
      default: return 'neutral';
    }
  }
</script>

{#if conflicts.length === 0}
  <p class="no-conflicts">No pending conflicts. All clear.</p>
{:else}
  <div class="conflict-queue" role="list" aria-label="Conflict review queue">
    <h3 class="conflict-queue__heading">
      {conflicts.length} {conflicts.length === 1 ? 'conflict' : 'conflicts'} to review
    </h3>

    {#each conflicts as conflict (conflict.id)}
      {@const note = resolutions[conflict.id] ?? ''}
      {@const isSubmitting = submitting.has(conflict.id)}

      <div class="conflict-card" role="listitem">
        <div class="conflict-card__header">
          <div class="conflict-card__header-left">
            <svg class="conflict-card__icon" width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <path d="M7 1.75a.875.875 0 0 1 .875.875c0 .159-.042.314-.117.456l-3.5 6.72A.875.875 0 0 1 3.5 10H1.75v.875h12.5V10h-1.75a.875.875 0 0 1-.758-1.174l-3.5-6.72A.875.875 0 0 1 7 1.75ZM7 3.5l3.08 5.917H3.92L7 3.5ZM2.187 11.083h9.626v.875h-9.626v-.875Z" fill="currentColor"/>
            </svg>
            <span class="conflict-card__category">
              <Badge variant="warning">{conflict.conflicting_items.current[0]?.category ?? 'Conflict'}</Badge>
              Conflict between this meeting and prior memory
            </span>
          </div>
        </div>

        <div class="conflict-card__sides">
          <!-- Current evidence -->
          <div class="conflict-side">
            <p class="conflict-side__label">Current meeting</p>
            {#each conflict.conflicting_items.current as item}
              <div class="conflict-item">
                <div class="conflict-item__header">
                  <Badge variant="warning">{item.category}</Badge>
                  <Badge variant={qualityVariant(item.quality_status)}>{item.quality_status}</Badge>
                </div>
                <p class="conflict-item__text">{escapeHtml(item.item_text)}</p>
                {#if item.evidence.length > 0}
                  <details class="conflict-evidence">
                    <summary class="conflict-evidence__toggle">
                      {item.evidence.length} {item.evidence.length === 1 ? 'source' : 'sources'}
                    </summary>
                    <ul class="evidence-list" role="list">
                      {#each item.evidence as ev}
                        <li class="evidence-list__item">
                          <div class="evidence-list__header">
                            <span class="evidence-list__type">{ev.source_type}</span>
                          </div>
                          <blockquote class="evidence-list__snippet">{escapeHtml(ev.snippet)}</blockquote>
                        </li>
                      {/each}
                    </ul>
                  </details>
                {/if}
              </div>
            {/each}
          </div>

          <!-- Prior evidence -->
          <div class="conflict-side conflict-side--prior">
            <p class="conflict-side__label">Prior memory</p>
            {#each conflict.conflicting_items.prior as item}
              <div class="conflict-item">
                <div class="conflict-item__header">
                  <Badge variant="warning">{item.category}</Badge>
                  <Badge variant={qualityVariant(item.quality_status)}>{item.quality_status}</Badge>
                </div>
                <p class="conflict-item__text">{escapeHtml(item.item_text)}</p>
                {#if item.evidence.length > 0}
                  <details class="conflict-evidence">
                    <summary class="conflict-evidence__toggle">
                      {item.evidence.length} {item.evidence.length === 1 ? 'source' : 'sources'}
                    </summary>
                    <ul class="evidence-list" role="list">
                      {#each item.evidence as ev}
                        <li class="evidence-list__item">
                          <div class="evidence-list__header">
                            <span class="evidence-list__type">{ev.source_type}</span>
                          </div>
                          <blockquote class="evidence-list__snippet">{escapeHtml(ev.snippet)}</blockquote>
                        </li>
                      {/each}
                    </ul>
                  </details>
                {/if}
              </div>
            {/each}
          </div>
        </div>

        <!-- Resolution -->
        <div class="conflict-resolution">
          <label for="resolution-{conflict.id}" class="conflict-resolution__label">
            Resolution note
          </label>
          <textarea
            id="resolution-{conflict.id}"
            class="conflict-resolution__textarea"
            placeholder="Describe how this conflict was resolved (evidence-backed note, or state that evidence is insufficient to resolve)..."
            rows="3"
            bind:value={resolutions[conflict.id]}
            disabled={isSubmitting}
          ></textarea>
          <Button
            variant="primary"
            size="sm"
            loading={isSubmitting}
            disabled={!note.trim()}
            onclick={() => handleResolve(conflict.id)}
          >
            Mark reviewed
          </Button>
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .no-conflicts {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin: 0;
  }

  .conflict-queue {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .conflict-queue__heading {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .conflict-card {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    overflow: hidden;
  }

  .conflict-card__header {
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--color-border);
    background: var(--color-background-subtle);
  }

  .conflict-card__header-left {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .conflict-card__icon {
    color: var(--color-warning);
    flex-shrink: 0;
  }

  .conflict-card__category {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
  }

  .conflict-card__sides {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1px;
    background: var(--color-border);
  }

  @media (max-width: 640px) {
    .conflict-card__sides {
      grid-template-columns: 1fr;
    }
  }

  .conflict-side {
    background: var(--color-background);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .conflict-side--prior {
    background: var(--color-background-subtle);
    border-left: 3px solid var(--color-warning);
  }

  @media (min-width: 641px) {
    .conflict-side--prior {
      border-left: none;
      border-right: 3px solid var(--color-warning);
    }
  }

  .conflict-side__label {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-muted);
    margin: 0 0 var(--space-1) 0;
  }

  .conflict-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .conflict-item__header {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex-wrap: wrap;
  }

  .conflict-item__text {
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    margin: 0;
    line-height: 1.5;
  }

  .conflict-evidence {
    font-size: var(--text-xs);
  }

  .conflict-evidence__toggle {
    color: var(--color-text-muted);
    cursor: pointer;
    font-size: var(--text-xs);
    list-style: none;
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .conflict-evidence__toggle::-webkit-details-marker {
    display: none;
  }

  .conflict-evidence__toggle::before {
    content: '▶';
    font-size: 0.5em;
    transition: transform var(--transition-fast);
  }

  details[open] .conflict-evidence__toggle::before {
    transform: rotate(90deg);
  }

  .evidence-list {
    list-style: none;
    margin: var(--space-2) 0 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .evidence-list__item {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    border-left: 2px solid var(--color-border);
    padding-left: var(--space-2);
  }

  .evidence-list__header {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .evidence-list__type {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: capitalize;
  }

  .evidence-list__snippet {
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    font-style: italic;
    margin: var(--space-1) 0 0 0;
  }

  .conflict-resolution {
    background: var(--color-background);
    border-top: 1px solid var(--color-border);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .conflict-resolution__label {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  .conflict-resolution__textarea {
    width: 100%;
    padding: var(--space-2) var(--space-3);
    font-family: var(--font-sans);
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    resize: vertical;
    transition: border-color var(--transition-fast);
  }

  .conflict-resolution__textarea:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .conflict-resolution__textarea:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>