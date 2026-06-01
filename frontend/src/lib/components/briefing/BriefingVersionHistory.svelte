<script lang="ts">
  import { Badge } from '$lib/components/ui';
  import type { BriefingVersionSummary } from '$lib/api/briefing';

  interface Props {
    versions: BriefingVersionSummary[];
    activeVersionNumber: number | null;
    onSelectVersion: (versionNumber: number) => void;
  }

  let { versions, activeVersionNumber, onSelectVersion }: Props = $props();

  let expanded = $state<Set<number>>(new Set());

  function toggleExpanded(versionNumber: number) {
    if (expanded.has(versionNumber)) {
      expanded.delete(versionNumber);
    } else {
      expanded.add(versionNumber);
    }
    expanded = new Set(expanded);
  }

  const triggerLabels: Record<string, string> = {
    auto: 'Auto-generated',
    manual_regenerate: 'Manual regenerate',
  };

  const statusVariant: Record<string, 'success' | 'neutral' | 'danger'> = {
    active: 'success',
    superseded: 'neutral',
  };

  const statusLabel: Record<string, string> = {
    active: 'Active',
    superseded: 'Superseded',
  };

  const resultVariant: Record<string, 'success' | 'warning' | 'danger' | 'neutral'> = {
    ready: 'success',
    ready_with_caveats: 'warning',
    no_prior_memory: 'neutral',
    failed: 'danger',
  };

  const resultLabel: Record<string, string> = {
    ready: 'Ready',
    ready_with_caveats: 'Ready with caveats',
    no_prior_memory: 'No prior memory',
    failed: 'Failed',
  };
</script>

<div class="version-history">
  <h3 class="version-history__heading">
    Version history
    <span class="version-history__count">{versions.length}</span>
  </h3>

  {#if versions.length === 0}
    <p class="version-history__empty">No versions yet.</p>
  {:else}
    <ol class="version-list" role="list">
      {#each versions as version (version.id)}
        <li class="version-item" class:version-item--active={version.is_active}>
          <div class="version-item__main">
            <button
              type="button"
              class="version-toggle"
              onclick={() => toggleExpanded(version.version_number)}
              aria-expanded={expanded.has(version.version_number)}
            >
              <svg
                class="version-toggle__chevron"
                class:version-toggle__chevron--open={expanded.has(version.version_number)}
                width="12"
                height="12"
                viewBox="0 0 12 12"
                fill="none"
                aria-hidden="true"
              >
                <path d="M3 4.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <span class="version-toggle__number">v{version.version_number}</span>
              {#if version.is_active}
                <Badge variant="success">Active</Badge>
              {/if}
              <Badge variant={resultVariant[version.result] ?? 'neutral'}>{resultLabel[version.result] ?? version.result}</Badge>
            </button>

            <div class="version-item__meta">
              <span class="version-item__trigger">
                {triggerLabels[version.trigger_type] ?? version.trigger_type}
              </span>
              <span class="version-item__dot" aria-hidden="true">·</span>
              <span class="version-item__date">
                {new Date(version.created_at).toLocaleString()}
              </span>
              <span class="version-item__dot" aria-hidden="true">·</span>
              <span class="version-item__sources">
                {version.source_count} {version.source_count === 1 ? 'source' : 'sources'}
              </span>
            </div>
          </div>

          {#if expanded.has(version.version_number)}
            <div class="version-item__details">
              <dl class="version-details">
                <div class="version-details__row">
                  <dt>Version</dt>
                  <dd>v{version.version_number}</dd>
                </div>
                <div class="version-details__row">
                  <dt>Status</dt>
                  <dd>{statusLabel[version.status] ?? version.status}</dd>
                </div>
                <div class="version-details__row">
                  <dt>Result</dt>
                  <dd>{resultLabel[version.result] ?? version.result}</dd>
                </div>
                <div class="version-details__row">
                  <dt>Trigger</dt>
                  <dd>{triggerLabels[version.trigger_type] ?? version.trigger_type}</dd>
                </div>
                <div class="version-details__row">
                  <dt>Sources</dt>
                  <dd>{version.source_count}</dd>
                </div>
                <div class="version-details__row">
                  <dt>Created</dt>
                  <dd>{new Date(version.created_at).toLocaleString()}</dd>
                </div>
              </dl>

              {#if !version.is_active}
                <button
                  type="button"
                  class="version-item__select-btn"
                  onclick={() => onSelectVersion(version.version_number)}
                >
                  View this version
                </button>
              {/if}
            </div>
          {/if}
        </li>
      {/each}
    </ol>
  {/if}
</div>

<style>
  .version-history {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .version-history__heading {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .version-history__count {
    background: var(--color-background-subtle);
    color: var(--color-text-muted);
    font-size: var(--text-xs);
    font-weight: 400;
    padding: 0.0625rem var(--space-2);
    border-radius: var(--radius-full);
  }

  .version-history__empty {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin: 0;
  }

  .version-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .version-item {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    overflow: hidden;
    transition: border-color var(--transition-fast);
  }

  .version-item:hover {
    border-color: var(--color-text-muted);
  }

  .version-item--active {
    border-color: var(--color-success);
    border-style: solid;
  }

  .version-item__main {
    padding: var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .version-toggle {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: var(--font-sans);
    width: 100%;
    text-align: left;
    border-radius: var(--radius-sm);
  }

  .version-toggle:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .version-toggle__chevron {
    flex-shrink: 0;
    transition: transform var(--transition-fast);
  }

  .version-toggle__chevron:not(.version-toggle__chevron--open) {
    transform: rotate(-90deg);
  }

  .version-toggle__chevron--open {
    transform: rotate(0deg);
  }

  .version-toggle__number {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
    font-family: var(--font-mono);
  }

  .version-item__meta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding-left: calc(12px + var(--space-2));
  }

  .version-item__trigger,
  .version-item__date,
  .version-item__sources {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .version-item__dot {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }

  .version-item__details {
    padding: var(--space-3);
    border-top: 1px solid var(--color-border);
    background: var(--color-background-subtle);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .version-details {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--space-1) var(--space-4);
    margin: 0;
  }

  .version-details dt {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    font-weight: 500;
  }

  .version-details dd {
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    margin: 0;
  }

  .version-item__select-btn {
    align-self: flex-start;
    background: none;
    border: 1px solid var(--color-border);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-sm);
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--color-text-secondary);
    cursor: pointer;
    font-family: var(--font-sans);
    transition: border-color var(--transition-fast), color var(--transition-fast);
  }

  .version-item__select-btn:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  .version-item__select-btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }
</style>