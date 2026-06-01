<script lang="ts">
  import { Card, Button, Alert, Spinner, Badge } from '$lib/components/ui';
  import StaleIndicator from './StaleIndicator.svelte';
  import BriefingSourceList from './BriefingSourceList.svelte';
  import ExcludedSources from './ExcludedSources.svelte';
  import BriefingConflictReview from './BriefingConflictReview.svelte';
  import { escapeHtml } from '$lib/utils/html';
  import type {
    BriefingActive,
    BriefingContent,
    BriefingPreparationStatus,
    QualityStatus,
  } from '$lib/api/briefing';

  interface Props {
    briefing: BriefingActive;
    onRegenerate: () => void;
    onExcludeSource: (sourceId: string) => void;
    onRestoreSource: (sourceId: string) => void;
    onResolveConflict: (conflictId: string, resolutionNote: string) => Promise<void>;
    regenerating?: boolean;
    flagMismatch?: boolean;
  }

  let {
    briefing,
    onRegenerate,
    onExcludeSource,
    onRestoreSource,
    onResolveConflict,
    regenerating = false,
    flagMismatch = false,
  }: Props = $props();

  let detailedOpen = $state(false);

  // Determine if regenerate button should be shown
  const regenerateableStates: BriefingPreparationStatus[] = ['failed', 'stale', 'ready_with_caveats'];
  const canRegenerate = $derived(
    regenerateableStates.includes(briefing.version.preparation_status as BriefingPreparationStatus)
  );

  // ARIA live region announcement based on status
  const status = $derived(briefing.version.preparation_status as BriefingPreparationStatus);

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
      case 'strong_evidence': return 'Strong evidence';
      case 'weak_evidence': return 'Weak evidence';
      case 'insufficient_evidence': return 'Insufficient evidence';
      case 'conflicting_evidence': return 'Conflicting evidence';
      default: return q;
    }
  }

  // Concise summary section labels
  const summaryFields: Array<{ key: keyof BriefingContent['concise_summary']; label: string }> = [
    { key: 'objective', label: 'Objective' },
    { key: 'recommended_focus', label: 'Recommended focus' },
    { key: 'top_prior_context', label: 'Prior context' },
    { key: 'open_actions', label: 'Open actions' },
    { key: 'risks_questions', label: 'Risks & questions' },
  ];
</script>

<div class="briefing">
  <!-- Header with status and actions -->
  <div class="briefing__header">
    <div class="briefing__header-left">
      <StaleIndicator status={status} />
      <span class="briefing__version">v{briefing.version.version_number}</span>
      {#if briefing.is_stale}
        <span class="briefing__stale-notice">
          {briefing.stale_source_count} {briefing.stale_source_count === 1 ? 'source has' : 'sources have'} been updated
        </span>
      {/if}
    </div>
    <div class="briefing__header-right">
      {#if canRegenerate}
        <Button
          variant="secondary"
          size="sm"
          loading={regenerating}
          onclick={onRegenerate}
        >
          {regenerating ? 'Regenerating...' : 'Regenerate'}
        </Button>
      {/if}
    </div>
  </div>

  {#if flagMismatch}
    <Alert variant="warning" role="alert">
      Feature flag mismatch detected. Briefing functionality may be limited. Contact your administrator.
    </Alert>
  {/if}

  <!-- No-prior-memory shell -->
  {#if briefing.content.no_prior_memory_shell?.generated}
    <Card>
      <div class="no-prior-memory">
        <div class="no-prior-memory__badge">No prior memory found</div>
        <p class="no-prior-memory__description">
          No related prior meetings were found for this upcoming meeting.
          Below is a preparation shell based on the upcoming meeting's own information.
        </p>
        {#if briefing.content.no_prior_memory_shell.objective}
          <div class="no-prior-memory__section">
            <span class="no-prior-memory__section-label">Objective</span>
            <p class="no-prior-memory__section-text">
              {escapeHtml(briefing.content.no_prior_memory_shell.objective)}
            </p>
          </div>
        {/if}
        {#if briefing.content.no_prior_memory_shell.recommended_focus}
          <div class="no-prior-memory__section">
            <span class="no-prior-memory__section-label">Recommended focus</span>
            <p class="no-prior-memory__section-text">
              {escapeHtml(briefing.content.no_prior_memory_shell.recommended_focus)}
            </p>
          </div>
        {/if}
        {#if briefing.content.no_prior_memory_shell.suggested_prep_questions.length > 0}
          <div class="no-prior-memory__section">
            <span class="no-prior-memory__section-label">Suggested prep questions</span>
            <ul class="no-prior-memory__questions">
              {#each briefing.content.no_prior_memory_shell.suggested_prep_questions as q}
                <li>{escapeHtml(q)}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    </Card>
  {:else}
    <!-- Concise summary — always visible -->
    <Card>
      <div class="concise-summary">
        <h3 class="concise-summary__heading">Briefing summary</h3>

        {#if briefing.content.concise_summary.preparation_status}
          <div class="concise-summary__status-line">
            <span class="concise-summary__status-label">Status:</span>
            <span class="concise-summary__status-value">
              {escapeHtml(briefing.content.concise_summary.preparation_status.replace(/_/g, ' '))}
            </span>
          </div>
        {/if}

        <div class="concise-summary__sections">
          {#each summaryFields as field}
            {@const value = briefing.content.concise_summary[field.key]}
            {#if value}
              <div class="concise-summary__field">
                <dt class="concise-summary__field-label">{field.label}</dt>
                <dd class="concise-summary__field-value">{escapeHtml(value)}</dd>
              </div>
            {/if}
          {/each}
        </div>
      </div>
    </Card>

    <!-- Detailed sections — collapsible -->
    <details class="detailed-sections" bind:open={detailedOpen}>
      <summary class="detailed-sections__toggle">
        <svg
          class="detailed-sections__chevron"
          class:detailed-sections__chevron--open={detailedOpen}
          width="12"
          height="12"
          viewBox="0 0 12 12"
          fill="none"
          aria-hidden="true"
        >
          <path d="M3 4.5l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        View full briefing details
      </summary>

      <div class="detailed-sections__content">
        <!-- Previous relevant context -->
        {#if briefing.content.detailed_sections.previous_relevant_context.statement}
          <div class="detail-section">
            <h4 class="detail-section__heading">
              Previous relevant context
              <Badge variant={qualityVariant(briefing.content.detailed_sections.previous_relevant_context.quality_status)}>
                {qualityLabel(briefing.content.detailed_sections.previous_relevant_context.quality_status)}
              </Badge>
            </h4>
            <p class="detail-section__statement">
              {escapeHtml(briefing.content.detailed_sections.previous_relevant_context.statement)}
            </p>
          </div>
        {/if}

        <!-- Important prior decisions -->
        {#if briefing.content.detailed_sections.important_prior_decisions.items.length > 0}
          <div class="detail-section">
            <h4 class="detail-section__heading">
              Important prior decisions
              <Badge variant={qualityVariant(briefing.content.detailed_sections.important_prior_decisions.quality_status)}>
                {qualityLabel(briefing.content.detailed_sections.important_prior_decisions.quality_status)}
              </Badge>
            </h4>
            <ol class="detail-section__list">
              {#each briefing.content.detailed_sections.important_prior_decisions.items as item}
                <li>{escapeHtml(item)}</li>
              {/each}
            </ol>
          </div>
        {/if}

        <!-- Open action items -->
        {#if briefing.content.detailed_sections.open_action_items.items.length > 0}
          <div class="detail-section">
            <h4 class="detail-section__heading">
              Open action items
              <Badge variant={qualityVariant(briefing.content.detailed_sections.open_action_items.quality_status)}>
                {qualityLabel(briefing.content.detailed_sections.open_action_items.quality_status)}
              </Badge>
            </h4>
            <ol class="detail-section__list">
              {#each briefing.content.detailed_sections.open_action_items.items as item}
                <li>{escapeHtml(item)}</li>
              {/each}
            </ol>
          </div>
        {/if}

        <!-- Unresolved risks/blockers -->
        {#if briefing.content.detailed_sections.unresolved_risks_blockers.items.length > 0}
          <div class="detail-section">
            <h4 class="detail-section__heading">
              Unresolved risks & blockers
              <Badge variant={qualityVariant(briefing.content.detailed_sections.unresolved_risks_blockers.quality_status)}>
                {qualityLabel(briefing.content.detailed_sections.unresolved_risks_blockers.quality_status)}
              </Badge>
            </h4>
            <ol class="detail-section__list">
              {#each briefing.content.detailed_sections.unresolved_risks_blockers.items as item}
                <li>{escapeHtml(item)}</li>
              {/each}
            </ol>
          </div>
        {/if}

        <!-- Open questions -->
        {#if briefing.content.detailed_sections.open_questions.items.length > 0}
          <div class="detail-section">
            <h4 class="detail-section__heading">
              Open questions
              <Badge variant={qualityVariant(briefing.content.detailed_sections.open_questions.quality_status)}>
                {qualityLabel(briefing.content.detailed_sections.open_questions.quality_status)}
              </Badge>
            </h4>
            <ol class="detail-section__list">
              {#each briefing.content.detailed_sections.open_questions.items as item}
                <li>{escapeHtml(item)}</li>
              {/each}
            </ol>
          </div>
        {/if}

        <!-- Suggested questions -->
        {#if briefing.content.detailed_sections.suggested_questions.items.length > 0}
          <div class="detail-section">
            <h4 class="detail-section__heading">Suggested questions</h4>
            <ol class="detail-section__list">
              {#each briefing.content.detailed_sections.suggested_questions.items as item}
                <li>{escapeHtml(item)}</li>
              {/each}
            </ol>
          </div>
        {/if}

        <!-- Suggested agenda -->
        {#if briefing.content.detailed_sections.suggested_agenda.items.length > 0}
          <div class="detail-section">
            <h4 class="detail-section__heading">Suggested agenda</h4>
            <ol class="detail-section__list">
              {#each briefing.content.detailed_sections.suggested_agenda.items as item}
                <li>{escapeHtml(item)}</li>
              {/each}
            </ol>
          </div>
        {/if}
      </div>
    </details>
  {/if}

  <!-- Source list with exclude action -->
  {#if briefing.content.sources.length > 0 && !briefing.content.no_prior_memory_shell?.generated}
    <BriefingSourceList
      sources={briefing.content.sources}
      onExclude={onExcludeSource}
      excludedSourceIds={new Set(briefing.excluded_sources.map(s => s.source_meeting_id))}
    />
  {/if}

  <!-- Excluded sources with restore action -->
  {#if briefing.excluded_sources.length > 0}
    <ExcludedSources
      excludedSources={briefing.excluded_sources}
      onRestore={onRestoreSource}
    />
  {/if}

  <!-- Conflict review queue -->
  {#if briefing.conflicts.length > 0}
    <BriefingConflictReview
      conflicts={briefing.conflicts}
      {onResolveConflict}
    />
  {/if}
</div>

<style>
  .briefing {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .briefing__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    flex-wrap: wrap;
  }

  .briefing__header-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .briefing__header-right {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .briefing__version {
    font-size: var(--text-xs);
    font-family: var(--font-mono);
    color: var(--color-text-muted);
  }

  .briefing__stale-notice {
    font-size: var(--text-xs);
    color: var(--color-warning);
  }

  /* Concise summary */
  .concise-summary {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .concise-summary__heading {
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .concise-summary__status-line {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    font-size: var(--text-sm);
  }

  .concise-summary__status-label {
    color: var(--color-text-muted);
  }

  .concise-summary__status-value {
    color: var(--color-text-primary);
    text-transform: capitalize;
  }

  .concise-summary__sections {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .concise-summary__field {
    display: grid;
    grid-template-columns: 140px 1fr;
    gap: var(--space-2);
  }

  .concise-summary__field-label {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .concise-summary__field-value {
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    margin: 0;
    line-height: 1.5;
  }

  /* Detailed sections */
  .detailed-sections {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    overflow: hidden;
  }

  .detailed-sections__toggle {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    background: var(--color-background-subtle);
    cursor: pointer;
    font-family: var(--font-sans);
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-secondary);
    border: none;
    width: 100%;
    text-align: left;
    transition: color var(--transition-fast);
  }

  .detailed-sections__toggle:hover {
    color: var(--color-text-primary);
  }

  .detailed-sections__toggle:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: -2px;
  }

  .detailed-sections__chevron {
    flex-shrink: 0;
    color: var(--color-text-muted);
    transition: transform var(--transition-fast);
  }

  .detailed-sections__chevron:not(.detailed-sections__chevron--open) {
    transform: rotate(-90deg);
  }

  .detailed-sections__chevron--open {
    transform: rotate(0deg);
  }

  .detailed-sections__content {
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
    border-top: 1px solid var(--color-border);
  }

  .detail-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .detail-section__heading {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  .detail-section__statement {
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    margin: 0;
    line-height: 1.6;
  }

  .detail-section__list {
    margin: 0;
    padding-left: var(--space-5);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .detail-section__list li {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    line-height: 1.5;
  }

  /* No prior memory shell */
  .no-prior-memory {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .no-prior-memory__badge {
    display: inline-flex;
    align-items: center;
    background: var(--color-background-subtle);
    border: 1px solid var(--color-border);
    color: var(--color-text-muted);
    padding: 0.125rem var(--space-2);
    border-radius: var(--radius-full);
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .no-prior-memory__description {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.5;
  }

  .no-prior-memory__section {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .no-prior-memory__section-label {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .no-prior-memory__section-text {
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    margin: 0;
    line-height: 1.5;
  }

  .no-prior-memory__questions {
    margin: 0;
    padding-left: var(--space-5);
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .no-prior-memory__questions li {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
  }
</style>