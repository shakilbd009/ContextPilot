<script lang="ts">
  import { Badge } from '$lib/components/ui';
  import type { MemoryVersionSummary } from '$lib/api/memory';

  interface Props {
    versions: MemoryVersionSummary[];
    activeVersionNumber: number | null;
    onSelectVersion: (versionNumber: number) => void;
  }

  let { versions, activeVersionNumber, onSelectVersion }: Props = $props();

  let expanded = $state<Set<number>>(new Set());
  let selectedVersionNumber = $state<number | null>(null);

  function toggleExpanded(versionNumber: number) {
    if (expanded.has(versionNumber)) {
      expanded.delete(versionNumber);
    } else {
      expanded.add(versionNumber);
    }
    expanded = new Set(expanded);
  }

  const triggerLabels: Record<string, string> = {
    import: 'Imported meeting',
    reprocess: 'Manual reprocess',
    stale_reprocess: 'Stale reprocess',
    manual_retry: 'Manual retry',
    conflict_resolution: 'Conflict resolution',
  };

  const statusVariant: Record<string, 'success' | 'neutral'> = {
    active: 'success',
    superseded: 'neutral',
    conflict_review: 'neutral',
  };

  const statusLabel: Record<string, string> = {
    active: 'Active',
    superseded: 'Superseded',
    conflict_review: 'Conflict review',
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
              {#if version.status === 'superseded'}
                <Badge variant="neutral">Superseded</Badge>
              {/if}
              {#if version.status === 'conflict_review'}
                <Badge variant="warning">Conflict review</Badge>
              {/if}
            </button>

            <div class="version-item__meta">
              <span class="version-item__trigger">
                {triggerLabels[version.trigger_type] ?? version.trigger_type}
              </span>
              <span class="version-item__dot" aria-hidden="true">·</span>
              <span class="version-item__date">
                {new Date(version.created_at).toLocaleString()}
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
                  <dt>Trigger</dt>
                  <dd>{triggerLabels[version.trigger_type] ?? version.trigger_type}</dd>
                </div>
                <div class="version-details__row">
                  <dt>Created</dt>
                  <dd>{new Date(version.created_at).toLocaleString()}</dd>
                </div>
                {#if version.created_by}
                  <div class="version-details__row">
                    <dt>By</dt>
                    <dd>{version.created_by}</dd>
                  </div>
                {/if}
              </dl>

              {#if !version.is_active}
                <button
                  type="button"
                  class="version-item__compare-btn"
                  onclick={() => { selectedVersionNumber = version.version_number; }}
                >
                  {selectedVersionNumber === version.version_number ? 'Hide comparison' : 'Compare to active'}
                </button>
              {/if}
            </div>
          {/if}
        </li>
      {/each}
    </ol>

    {#if selectedVersionNumber !== null}
      {@const selectedVersion = versions.find(v => v.version_number === selectedVersionNumber)}
      {@const activeVersion = versions.find(v => v.is_active)}
      {#if selectedVersion && activeVersion}
        <div class="version-compare">
          <div class="version-compare__header">
            <h4 class="version-compare__title">
              Comparing v{selectedVersionNumber} → v{activeVersion.version_number} (active)
            </h4>
            <button
              type="button"
              class="version-compare__close"
              onclick={() => { selectedVersionNumber = null; }}
              aria-label="Close comparison"
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <path d="M3 3l8 8M11 3l-8 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </button>
          </div>
          <div class="version-compare__notice">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5"/>
              <path d="M7 4v3.5M7 9v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            </svg>
            Full content comparison requires fetching version content from the server.
            This view shows metadata only.
          </div>
          <dl class="version-compare__meta">
            <div class="version-compare__meta-row">
              <dt>Selected version trigger</dt>
              <dd>{triggerLabels[selectedVersion.trigger_type] ?? selectedVersion.trigger_type}</dd>
            </div>
            <div class="version-compare__meta-row">
              <dt>Active version trigger</dt>
              <dd>{triggerLabels[activeVersion.trigger_type] ?? activeVersion.trigger_type}</dd>
            </div>
            <div class="version-compare__meta-row">
              <dt>Selected created</dt>
              <dd>{new Date(selectedVersion.created_at).toLocaleString()}</dd>
            </div>
            <div class="version-compare__meta-row">
              <dt>Active created</dt>
              <dd>{new Date(activeVersion.created_at).toLocaleString()}</dd>
            </div>
          </dl>
        </div>
      {/if}
    {/if}
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

  .version-toggle__chevron--open {
    transform: rotate(0deg);
  }

  .version-toggle__chevron:not(.version-toggle__chevron--open) {
    transform: rotate(-90deg);
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

  .version-item__trigger {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .version-item__date {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
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

  .version-item__compare-btn {
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

  .version-item__compare-btn:hover {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  .version-item__compare-btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .version-compare {
    margin-top: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    overflow: hidden;
    background: var(--color-background);
  }

  .version-compare__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    background: var(--color-background-subtle);
    border-bottom: 1px solid var(--color-border);
  }

  .version-compare__title {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .version-compare__close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--color-text-muted);
    border-radius: var(--radius-sm);
    transition: color var(--transition-fast), background var(--transition-fast);
    flex-shrink: 0;
  }

  .version-compare__close:hover {
    color: var(--color-text-primary);
    background: var(--color-background);
  }

  .version-compare__close:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .version-compare__notice {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    border-bottom: 1px solid var(--color-border);
    line-height: 1.5;
  }

  .version-compare__notice svg {
    flex-shrink: 0;
    margin-top: 1px;
  }

  .version-compare__meta {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0;
  }

  .version-compare__meta-row {
    display: contents;
  }

  .version-compare__meta-row dt,
  .version-compare__meta-row dd {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-xs);
    border-bottom: 1px solid var(--color-border);
  }

  .version-compare__meta-row:nth-child(odd) dt { background: var(--color-background-subtle); }
  .version-compare__meta-row:nth-child(odd) dd { background: var(--color-background-subtle); }
  .version-compare__meta-row:nth-child(even) dt { background: var(--color-background); }
  .version-compare__meta-row:nth-child(even) dd { background: var(--color-background); }

  .version-compare__meta-row dt {
    font-weight: 500;
    color: var(--color-text-muted);
  }

  .version-compare__meta-row dd {
    color: var(--color-text-secondary);
  }

  @media (max-width: 480px) {
    .version-compare__meta {
      grid-template-columns: 1fr;
    }
  }
</style>