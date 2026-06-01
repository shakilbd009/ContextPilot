<script lang="ts">
  import { Button } from '$lib/components/ui';
  import { escapeHtml } from '$lib/utils/html';
  import type { BriefingConflict } from '$lib/api/briefing';

  interface Props {
    conflicts: BriefingConflict[];
    onResolve: (conflictId: string, resolutionNote: string) => Promise<void>;
    onResolveConflict?: (conflictId: string, resolutionNote: string) => Promise<void>;
  }

  let { conflicts, onResolve, onResolveConflict }: Props = $props();

  // Prefer onResolveConflict if provided, otherwise fall back to onResolve
  let handleResolve = $derived(onResolveConflict ?? onResolve);

  let resolutions = $state<Record<string, string>>({});
  let submitting = $state<Set<string>>(new Set());

  async function _handleResolve(conflictId: string) {
    const note = resolutions[conflictId]?.trim() ?? '';
    submitting.add(conflictId);
    submitting = new Set(submitting);
    try {
      await handleResolve(conflictId, note);
      delete resolutions[conflictId];
      resolutions = { ...resolutions };
    } finally {
      submitting.delete(conflictId);
      submitting = new Set(submitting);
    }
  }
</script>

{#if conflicts.length === 0}
  <p class="no-conflicts">No conflicting evidence. All clear.</p>
{:else}
  <div class="conflict-queue" role="list" aria-label="Conflict review queue">
    <h4 class="conflict-queue__heading">
      {conflicts.length} {conflicts.length === 1 ? 'conflict' : 'conflicts'} to review
    </h4>

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
              <span class="conflict-card__category-badge">{conflict.category}</span>
              Conflict detected
            </span>
          </div>
        </div>

        <div class="conflict-card__evidence">
          <p class="conflict-card__snippet-label">Conflicting evidence:</p>
          <blockquote class="conflict-card__snippet">
            <!-- Evidence snippets are plain text — escape before render per FR-4a -->
            {escapeHtml(conflict.evidence_snippet)}
          </blockquote>
        </div>

        <!-- Resolution -->
        <div class="conflict-resolution">
          <label for="resolution-{conflict.id}" class="conflict-resolution__label">
            Resolution note
          </label>
          <textarea
            id="resolution-{conflict.id}"
            name="resolution-{conflict.id}"
            class="conflict-resolution__textarea"
            placeholder="Describe how this conflict was resolved or note that evidence is insufficient..."
            rows="3"
            bind:value={resolutions[conflict.id]}
            disabled={isSubmitting}
          ></textarea>
          <Button
            variant="primary"
            size="sm"
            loading={isSubmitting}
            disabled={!note.trim()}
            onclick={() => _handleResolve(conflict.id)}
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

  .conflict-card__category-badge {
    background: rgb(217 119 6 / 0.1);
    color: var(--color-warning);
    padding: 0.0625rem var(--space-2);
    border-radius: var(--radius-full);
    font-size: var(--text-xs);
    font-weight: 600;
  }

  .conflict-card__evidence {
    padding: var(--space-4);
    background: var(--color-background);
  }

  .conflict-card__snippet-label {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--color-text-muted);
    margin: 0 0 var(--space-1) 0;
  }

  .conflict-card__snippet {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    font-style: italic;
    margin: 0;
    line-height: 1.5;
    border-left: 2px solid var(--color-border);
    padding-left: var(--space-3);
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