<script lang="ts">
  import { Card, Badge } from '$lib/components/ui';
  import { escapeHtml } from '$lib/utils/html';
  import EvidencePanel from './EvidencePanel.svelte';
  import type {
    MemoryContent,
    DecisionItem,
    ActionItem,
    RiskBlockerItem,
    OpenQuestionItem,
    StakeholderNoteItem,
    QualityStatus,
  } from '$lib/api/memory';

  interface Props {
    content: MemoryContent;
    meetingId: string;
  }

  let { content, meetingId }: Props = $props();

  // Core categories for briefing readiness
  const coreCategories = ['summary', 'decisions', 'action_items', 'risks_blockers', 'open_questions'];
  const allCategories = ['summary', 'decisions', 'action_items', 'risks_blockers', 'open_questions', 'stakeholder_notes', 'next_recommended_focus'] as const;

  const categoryLabels: Record<string, string> = {
    summary: 'Summary',
    decisions: 'Decisions',
    action_items: 'Action Items',
    risks_blockers: 'Risks & Blockers',
    open_questions: 'Open Questions',
    stakeholder_notes: 'Stakeholder Notes',
    next_recommended_focus: 'Next Recommended Focus',
  };

  function qualityVariant(q: QualityStatus): 'success' | 'warning' | 'danger' | 'neutral' {
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

  function hasItems(category: keyof MemoryContent): boolean {
    const cat = content[category] as { items?: unknown[]; statement?: string } | undefined;
    if (!cat) return false;
    if ('items' in cat && Array.isArray(cat.items)) return cat.items.length > 0;
    return !!cat.statement;
  }

  function itemCount(category: keyof MemoryContent): number {
    const cat = content[category] as { items?: unknown[]; statement?: string } | undefined;
    if (!cat) return 0;
    if ('items' in cat && Array.isArray(cat.items)) return cat.items.length;
    return cat.statement ? 1 : 0;
  }

  function categoryQuality(category: keyof MemoryContent): QualityStatus {
    const cat = content[category] as { quality_status?: QualityStatus; items?: Array<{ quality_status: QualityStatus }> } | undefined;
    if (!cat) return 'insufficient_evidence';
    if ('quality_status' in cat) return cat.quality_status as QualityStatus;
    if ('items' in cat && Array.isArray(cat.items) && cat.items.length > 0) {
      return cat.items.reduce((worst: QualityStatus, item) => {
        const itemQs = item.quality_status;
        const order: QualityStatus[] = ['conflicting_evidence', 'weak_evidence', 'insufficient_evidence', 'strong_evidence'];
        return order.indexOf(itemQs) < order.indexOf(worst) ? itemQs : worst;
      }, 'strong_evidence' as QualityStatus);
    }
    return 'insufficient_evidence';
  }

  function changeStatusLabel(s: string): string {
    switch (s) {
      case 'standalone': return 'Standalone';
      case 'confirms_prior': return 'Confirms prior';
      case 'changes_prior': return 'Changes prior';
      case 'reverses_prior': return 'Reverses prior';
      default: return s;
    }
  }

  function changeStatusVariant(s: string): 'success' | 'warning' | 'danger' | 'neutral' {
    switch (s) {
      case 'standalone': return 'neutral';
      case 'confirms_prior': return 'success';
      case 'changes_prior': return 'warning';
      case 'reverses_prior': return 'danger';
      default: return 'neutral';
    }
  }
</script>

<div class="memory-view">
  <div class="memory-view__categories">
    {#each allCategories as catKey}
      {@const quality = categoryQuality(catKey as keyof MemoryContent)}
      {@const hasItemsForCat = hasItems(catKey as keyof MemoryContent)}
      {@const label = categoryLabels[catKey] ?? catKey}
      {@const count = itemCount(catKey as keyof MemoryContent)}
      <div class="category-row" data-category={catKey}>
        <div class="category-row__header">
          <span class="category-row__name">{label}</span>
          <Badge variant={qualityVariant(quality)}>{qualityLabel(quality)}</Badge>
          {#if hasItemsForCat && catKey !== 'summary' && catKey !== 'next_recommended_focus'}
            <span class="category-row__count">{count} {count === 1 ? 'item' : 'items'}</span>
          {/if}
        </div>

        {#if catKey === 'summary'}
          <p class="category-row__statement">{escapeHtml(content.summary.statement)}</p>
        {/if}

        {#if catKey === 'decisions' && content.decisions.items.length > 0}
          <ul class="item-list" role="list">
            {#each content.decisions.items as item (item.id)}
              <li class="item-list__item">
                <div class="item-header">
                  <p class="item-text">{escapeHtml(item.decision_statement)}</p>
                  <Badge variant={changeStatusVariant(item.change_status)}>{changeStatusLabel(item.change_status)}</Badge>
                </div>
                <EvidencePanel evidence={item.evidence} />
              </li>
            {/each}
          </ul>
        {/if}

        {#if catKey === 'action_items' && content.action_items.items.length > 0}
          <ul class="item-list" role="list">
            {#each content.action_items.items as item (item.id)}
              <li class="item-list__item">
                <div class="item-header">
                  <p class="item-text">{escapeHtml(item.description)}</p>
                  <Badge variant={qualityVariant(item.quality_status)}>{qualityLabel(item.quality_status)}</Badge>
                </div>
                {#if item.owner || item.due_date}
                  <div class="item-meta">
                    {#if item.owner}
                      <span class="item-meta__chip">Owner: {item.owner}</span>
                    {/if}
                    {#if item.due_date && item.due_date_specified}
                      <span class="item-meta__chip">Due: {item.due_date}</span>
                    {/if}
                    <span class="item-meta__chip">Status: {item.status}</span>
                  </div>
                {/if}
                <EvidencePanel evidence={item.evidence} />
              </li>
            {/each}
          </ul>
        {/if}

        {#if catKey === 'risks_blockers' && content.risks_blockers.items.length > 0}
          <ul class="item-list" role="list">
            {#each content.risks_blockers.items as item (item.id)}
              <li class="item-list__item">
                <div class="item-header">
                  <span class="item-type-badge item-type-badge--{item.type}">{item.type}</span>
                  <p class="item-text">{escapeHtml(item.description)}</p>
                  <Badge variant={qualityVariant(item.quality_status)}>{qualityLabel(item.quality_status)}</Badge>
                </div>
                <EvidencePanel evidence={item.evidence} />
              </li>
            {/each}
          </ul>
        {/if}

        {#if catKey === 'open_questions' && content.open_questions.items.length > 0}
          <ul class="item-list" role="list">
            {#each content.open_questions.items as item (item.id)}
              <li class="item-list__item">
                <div class="item-header">
                  <p class="item-text">{escapeHtml(item.question)}</p>
                  <Badge variant={qualityVariant(item.quality_status)}>{qualityLabel(item.quality_status)}</Badge>
                </div>
                <EvidencePanel evidence={item.evidence} />
              </li>
            {/each}
          </ul>
        {/if}

        {#if catKey === 'stakeholder_notes' && content.stakeholder_notes.items.length > 0}
          <ul class="item-list" role="list">
            {#each content.stakeholder_notes.items as item (item.id)}
              <li class="item-list__item">
                <div class="item-header">
                  <p class="item-text">Participant: {item.participant_id}</p>
                  <Badge variant={qualityVariant(item.quality_status)}>{qualityLabel(item.quality_status)}</Badge>
                </div>
                {#if item.preferences}
                  <p class="stakeholder-field"><strong>Preferences:</strong> {escapeHtml(item.preferences)}</p>
                {/if}
                {#if item.concerns}
                  <p class="stakeholder-field"><strong>Concerns:</strong> {escapeHtml(item.concerns)}</p>
                {/if}
                {#if item.commitments}
                  <p class="stakeholder-field"><strong>Commitments:</strong> {escapeHtml(item.commitments)}</p>
                {/if}
                {#if item.influence_stake}
                  <p class="stakeholder-field"><strong>Influence:</strong> {escapeHtml(item.influence_stake)}</p>
                {/if}
                <EvidencePanel evidence={item.evidence} />
              </li>
            {/each}
          </ul>
        {/if}

        {#if catKey === 'next_recommended_focus'}
          <p class="category-row__statement">{escapeHtml(content.next_recommended_focus.statement)}</p>
          {#if content.next_recommended_focus.supporting_item_ids.length > 0}
            <p class="supporting-ids">
              Supporting items: {content.next_recommended_focus.supporting_item_ids.join(', ')}
            </p>
          {/if}
          <EvidencePanel evidence={content.next_recommended_focus.evidence} />
        {/if}
      </div>
    {/each}
  </div>
</div>

<style>
  .memory-view {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .memory-view__categories {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .category-row {
    padding: var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-background);
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  }

  .category-row:hover {
    border-color: var(--color-text-muted);
    box-shadow: var(--shadow-sm);
  }

  .category-row__header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
    flex-wrap: wrap;
  }

  .category-row__name {
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--color-text-primary);
  }

  .category-row__count {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    margin-left: auto;
  }

  .category-row__statement {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    line-height: 1.6;
    margin: 0;
  }

  .item-list {
    list-style: none;
    margin: var(--space-2) 0 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .item-list__item {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    padding: var(--space-3);
    background: var(--color-background-subtle);
  }

  .item-header {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  .item-text {
    flex: 1;
    font-size: var(--text-sm);
    color: var(--color-text-primary);
    margin: 0;
    line-height: 1.5;
  }

  .item-type-badge {
    font-size: var(--text-xs);
    font-weight: 600;
    padding: 0.0625rem var(--space-2);
    border-radius: var(--radius-full);
    text-transform: capitalize;
    flex-shrink: 0;
  }

  .item-type-badge--risk {
    background: rgb(217 119 6 / 0.1);
    color: var(--color-warning);
  }

  .item-type-badge--blocker {
    background: rgb(220 38 38 / 0.1);
    color: var(--color-danger);
  }

  .item-meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-1);
    flex-wrap: wrap;
  }

  .item-meta__chip {
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    background: var(--color-background);
    border: 1px solid var(--color-border);
    padding: 0.0625rem var(--space-2);
    border-radius: var(--radius-full);
  }

  .stakeholder-field {
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    margin: var(--space-1) 0 0 0;
    line-height: 1.5;
  }

  .supporting-ids {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    font-family: var(--font-mono);
    margin: var(--space-1) 0 0 0;
  }
</style>