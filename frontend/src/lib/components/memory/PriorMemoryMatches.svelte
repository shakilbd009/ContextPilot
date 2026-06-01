<script lang="ts">
  import { Button, Badge, Card } from '$lib/components/ui';
  import { escapeHtml } from '$lib/utils/html';
  import type { PriorMemoryInput } from '$lib/api/memory';

  interface Props {
    matches: PriorMemoryInput[];
    onExclude: (priorMemoryVersionId: string) => Promise<void>;
    onInclude?: (priorMemoryVersionId: string) => Promise<void>;
  }

  let { matches, onExclude, onInclude }: Props = $props();

  let excluding = $state<Set<string>>(new Set());
  let confirmExcludeId = $state<string | null>(null);

  async function handleExclude(priorMemoryVersionId: string) {
    excluding.add(priorMemoryVersionId);
    excluding = new Set(excluding);
    try {
      await onExclude(priorMemoryVersionId);
    } finally {
      excluding.delete(priorMemoryVersionId);
      excluding = new Set(excluding);
      confirmExcludeId = null;
    }
  }

  const confidenceVariant: Record<string, 'success' | 'warning'> = {
    safe_match: 'success',
    uncertain: 'warning',
  };

  const confidenceLabel: Record<string, string> = {
    safe_match: 'Safe match',
    uncertain: 'Needs review',
  };
</script>

{#if matches.length > 0}
  <div class="prior-matches">
    <h3 class="prior-matches__title">Prior memories used in processing</h3>

    <ul class="prior-matches__list" role="list">
      {#each matches as match (match.id)}
        {@const isExcluding = excluding.has(match.prior_memory_version_id)}
        <li class="prior-match">
          <div class="prior-match__info">
            {#if match.prior_meeting_title}
              <p class="prior-match__title">{escapeHtml(match.prior_meeting_title)}</p>
            {/if}
            {#if match.prior_meeting_completed_at}
              <p class="prior-match__date">
                {new Date(match.prior_meeting_completed_at).toLocaleDateString()}
              </p>
            {/if}
            <div class="prior-match__badges">
              {#if match.match_confidence}
                <Badge variant={confidenceVariant[match.match_confidence] ?? 'neutral'}>
                  {confidenceLabel[match.match_confidence] ?? match.match_confidence}
                </Badge>
              {/if}
              {#if match.excluded_by_user}
                <Badge variant="danger">Excluded</Badge>
              {/if}
              {#if match.included_by_user}
                <Badge variant="warning">Manually included</Badge>
              {/if}
            </div>
          </div>

          <div class="prior-match__actions">
            {#if match.excluded_by_user && onInclude}
              <Button
                variant="ghost"
                size="sm"
                onclick={() => onInclude?.(match.prior_memory_version_id)}
                disabled={isExcluding}
              >
                Reinstate
              </Button>
            {:else if !match.excluded_by_user}
              {#if confirmExcludeId === match.prior_memory_version_id}
                <Button
                  variant="danger"
                  size="sm"
                  loading={isExcluding}
                  onclick={() => handleExclude(match.prior_memory_version_id)}
                >
                  Confirm exclude
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onclick={() => (confirmExcludeId = null)}
                >
                  Cancel
                </Button>
              {:else}
                <Button
                  variant="ghost"
                  size="sm"
                  onclick={() => (confirmExcludeId = match.prior_memory_version_id)}
                  disabled={isExcluding}
                >
                  Remove
                </Button>
              {/if}
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  </div>
{/if}

<style>
  .prior-matches {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .prior-matches__title {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .prior-matches__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .prior-match {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-background-subtle);
    transition: border-color var(--transition-fast);
  }

  .prior-match:hover {
    border-color: var(--color-text-muted);
  }

  .prior-match__info {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    flex: 1;
    min-width: 0;
  }

  .prior-match__title {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .prior-match__date {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    margin: 0;
  }

  .prior-match__badges {
    display: flex;
    gap: var(--space-1);
    flex-wrap: wrap;
    margin-top: var(--space-1);
  }

  .prior-match__actions {
    display: flex;
    gap: var(--space-1);
    flex-shrink: 0;
  }
</style>