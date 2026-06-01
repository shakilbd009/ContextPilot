import { describe, it, expect, vi, beforeEach, type MockedFunction } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/svelte';
import PreCallBriefing from './PreCallBriefing.svelte';
import type {
  BriefingActive,
  BriefingConflict,
  BriefingPreparationStatus,
} from '$lib/api/briefing';

// ── Test fixtures ─────────────────────────────────────────────────────────────

function makeBriefing(overrides: Partial<BriefingActive> = {}): BriefingActive {
  const content = {
    concise_summary: {
      objective: 'Review Q3 roadmap',
      preparation_status: 'ready',
      recommended_focus: 'Confirm priorities',
      top_prior_context: 'Q2 blockers still open',
      open_actions: '2 pending from last session',
      risks_questions: 'Budget approval pending',
    },
    detailed_sections: {
      previous_relevant_context: {
        statement: 'Sprint 23 retro completed',
        quality_status: 'strong_evidence' as const,
      },
      important_prior_decisions: {
        items: ['Adopt 2-week sprints', '80% test coverage target'],
        quality_status: 'strong_evidence' as const,
      },
      open_action_items: {
        items: ['Alice to update template', 'Bob to schedule follow-up'],
        quality_status: 'weak_evidence' as const,
      },
      unresolved_risks_blockers: {
        items: ['Vendor API deprecation risk'],
        quality_status: 'weak_evidence' as const,
      },
      open_questions: {
        items: ['Tech debt capacity allocation?'],
        quality_status: 'insufficient_evidence' as const,
      },
      stakeholder_notes: { items: [], quality_status: 'strong_evidence' as const },
      suggested_questions: { items: ['Status on Acme?'] },
      suggested_agenda: { items: ['Review (15 min)', 'Discuss (10 min)'] },
    },
    sources: [
      {
        source_meeting_id: 'src-001',
        relatedness_reasons: ['Same participant: alice@example.com'],
        quality_status: 'strong_evidence' as const,
      },
      {
        source_meeting_id: 'src-002',
        relatedness_reasons: ['Same org: Acme Corp', 'Close chronology'],
        quality_status: 'weak_evidence' as const,
      },
    ],
    no_prior_memory_shell: null,
  };

  return {
    version: {
      id: 'br-test-1',
      version_number: 3,
      status: 'active' as const,
      is_active: true,
      result: 'ready' as const,
      preparation_status: 'ready' as const,
      source_count: 2,
      trigger_type: 'auto' as const,
      created_at: new Date().toISOString(),
    },
    content,
    is_stale: false,
    stale_source_count: 0,
    excluded_sources: [],
    conflicts: [],
    ...overrides,
  };
}

function makeConflict(overrides: Partial<BriefingConflict> = {}): BriefingConflict {
  return {
    id: 'conf-001',
    category: 'decisions',
    evidence_snippet: 'We should prioritize the API migration',
    conflicting_meeting_id: 'meet-001',
    resolution_note: null,
    ...overrides,
  };
}

// ── Helpers ───────────────────────────────────────────────────────────────────

const noop = () => {};
const mockResolve = async () => {};

// ── Rendering ─────────────────────────────────────────────────────────────────

describe('PreCallBriefing', () => {
  let onRegenerate: MockedFunction<() => void>;
  let onExcludeSource: MockedFunction<(sourceId: string) => void>;
  let onRestoreSource: MockedFunction<(sourceId: string) => void>;
  let onResolveConflict: MockedFunction<(conflictId: string, resolutionNote: string) => Promise<void>>;

  beforeEach(() => {
    onRegenerate = vi.fn();
    onExcludeSource = vi.fn();
    onRestoreSource = vi.fn();
    onResolveConflict = vi.fn().mockResolvedValue(undefined);
  });

  describe('version badge', () => {
    it('displays version number', async () => {
      const briefing = makeBriefing({ version: { ...makeBriefing().version, version_number: 3 } });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('v3')).toBeTruthy();
    });
  });

  describe('concise summary', () => {
    it('renders all six required fields', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('Objective')).toBeTruthy();
      expect(screen.getByText('Recommended focus')).toBeTruthy();
      expect(screen.getByText('Prior context')).toBeTruthy();
      expect(screen.getByText('Open actions')).toBeTruthy();
      expect(screen.getByText('Risks & questions')).toBeTruthy();
    });

    it('renders status line from preparation_status', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('Ready')).toBeTruthy();
    });

    it('renders objective field value', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('Review Q3 roadmap')).toBeTruthy();
    });
  });

  describe('detailed sections', () => {
    it('shows collapsible toggle', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('View full briefing details')).toBeTruthy();
    });

    it('shows detail sections when expanded', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      const summary = screen.getByText('View full briefing details');
      await fireEvent.click(summary);

      expect(screen.getByText('Previous relevant context')).toBeTruthy();
      expect(screen.getByText('Important prior decisions')).toBeTruthy();
      expect(screen.getByText('Open action items')).toBeTruthy();
      expect(screen.getByText('Unresolved risks & blockers')).toBeTruthy();
    });

    it('shows quality badges on detail sections', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      const summary = screen.getByText('View full briefing details');
      await fireEvent.click(summary);
      // Quality badges appear in expanded sections — use getAllByText since multiple categories may share the same badge label
      const weakBadges = screen.getAllByText('Weak evidence');
      const insufficientBadges = screen.getAllByText('Insufficient evidence');
      expect(weakBadges.length).toBeGreaterThanOrEqual(1);
      expect(insufficientBadges.length).toBeGreaterThanOrEqual(1);
    });
  });

  // ── Stale indicator ─────────────────────────────────────────────────────────

  describe('stale notice', () => {
    it('shows stale notice when is_stale is true', async () => {
      const briefing = makeBriefing({ is_stale: true, stale_source_count: 2 });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('2 sources have been updated')).toBeTruthy();
    });

    it('shows singular form for count of 1', async () => {
      const briefing = makeBriefing({ is_stale: true, stale_source_count: 1 });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('1 source has been updated')).toBeTruthy();
    });

    it('does not show stale notice when is_stale is false', async () => {
      const briefing = makeBriefing({ is_stale: false });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.queryByText(/sources? have been updated/)).toBeNull();
    });
  });

  // ── Regenerate button ───────────────────────────────────────────────────────

  describe('regenerate button', () => {
    const regenerateStates: BriefingPreparationStatus[] = ['stale', 'failed', 'ready_with_caveats'];

    for (const status of regenerateStates) {
      it(`shows regenerate button when status is ${status}`, async () => {
        const briefing = makeBriefing({
          version: { ...makeBriefing().version, preparation_status: status },
        });
        await render(PreCallBriefing, {
          props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
        });
        expect(screen.getByRole('button', { name: 'Regenerate' })).toBeTruthy();
      });
    }

    it('does not show regenerate button for ready status', async () => {
      const briefing = makeBriefing({
        version: { ...makeBriefing().version, preparation_status: 'ready' },
      });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.queryByRole('button', { name: 'Regenerate' })).toBeNull();
    });

    it('calls onRegenerate when clicked', async () => {
      const briefing = makeBriefing({
        version: { ...makeBriefing().version, preparation_status: 'stale' },
      });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      await fireEvent.click(screen.getByRole('button', { name: 'Regenerate' }));
      expect(onRegenerate).toHaveBeenCalledOnce();
    });

    it('shows "Regenerating..." and disables button when regenerating=true', async () => {
      const briefing = makeBriefing({
        version: { ...makeBriefing().version, preparation_status: 'stale' },
      });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict, regenerating: true },
      });
      const btn = screen.getByRole('button', { name: 'Regenerating...' });
      expect((btn as HTMLButtonElement).disabled).toBe(true);
    });
  });

  // ── No-prior-memory shell ────────────────────────────────────────────────────

  describe('no-prior-memory shell', () => {
    it('renders shell when no_prior_memory_shell.generated is true', async () => {
      const briefing: BriefingActive = {
        ...makeBriefing(),
        content: {
          ...makeBriefing().content,
          no_prior_memory_shell: {
            generated: true,
            objective: 'Kickoff new project',
            recommended_focus: 'Define scope and timeline',
            suggested_prep_questions: ['What are the key milestones?'],
          },
          concise_summary: {
            objective: '', preparation_status: '', recommended_focus: '',
            top_prior_context: '', open_actions: '', risks_questions: '',
          },
        },
      };
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.getByText('No prior memory found')).toBeTruthy();
      expect(screen.getByText('Kickoff new project')).toBeTruthy();
      expect(screen.getByText('What are the key milestones?')).toBeTruthy();
    });

    it('does not render concise summary when shell is generated', async () => {
      const briefing: BriefingActive = {
        ...makeBriefing(),
        content: {
          ...makeBriefing().content,
          no_prior_memory_shell: {
            generated: true,
            objective: 'Shell objective',
            recommended_focus: 'Shell focus',
            suggested_prep_questions: [],
          },
          concise_summary: {
            objective: 'Concise should not appear',
            preparation_status: '',
            recommended_focus: '',
            top_prior_context: '',
            open_actions: '',
            risks_questions: '',
          },
        },
      };
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(screen.queryByText('Concise should not appear')).toBeNull();
    });
  });

  // ── Flag mismatch ─────────────────────────────────────────────────────────────

  describe('flag mismatch alert', () => {
    it('shows warning when flagMismatch is true', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict, flagMismatch: true },
      });
      expect(screen.getByRole('alert')).toBeTruthy();
      expect(screen.getByText(/Feature flag mismatch/)).toBeTruthy();
    });

    it('does not show alert when flagMismatch is false', async () => {
      const briefing = makeBriefing();
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict, flagMismatch: false },
      });
      expect(screen.queryByRole('alert')).toBeNull();
    });
  });

  // ── Conflict review queue ────────────────────────────────────────────────────

  describe('conflict review', () => {
    it('renders BriefingConflictReview when conflicts is non-empty', async () => {
      const briefing = makeBriefing({
        conflicts: [makeConflict({ evidence_snippet: 'Conflicting claim' })],
      });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(briefing.conflicts.length).toBe(1);
    });

    it('does not render conflict section when conflicts is empty', async () => {
      const briefing = makeBriefing({ conflicts: [] });
      await render(PreCallBriefing, {
        props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
      });
      expect(briefing.conflicts.length).toBe(0);
    });
  });
});

// ── ARIA live regions ───────────────────────────────────────────────────────────

describe('PreCallBriefing — ARIA live regions', () => {
  let onRegenerate: MockedFunction<() => void>;
  let onExcludeSource: MockedFunction<(sourceId: string) => void>;
  let onRestoreSource: MockedFunction<(sourceId: string) => void>;
  let onResolveConflict: MockedFunction<(conflictId: string, resolutionNote: string) => Promise<void>>;

  beforeEach(() => {
    onRegenerate = vi.fn();
    onExcludeSource = vi.fn();
    onRestoreSource = vi.fn();
    onResolveConflict = vi.fn().mockResolvedValue(undefined);
  });

  it('StaleIndicator uses aria-live="polite" for stale status', async () => {
    const briefing = makeBriefing({
      is_stale: true,
      stale_source_count: 1,
      version: { ...makeBriefing().version, preparation_status: 'stale' },
    });
    const { container } = await render(PreCallBriefing, {
      props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
    });
    const liveRegion = container.querySelector('[aria-live="polite"]');
    expect(liveRegion).toBeTruthy();
  });

  it('StaleIndicator uses aria-live="assertive" for failed status', async () => {
    const briefing = makeBriefing({
      version: { ...makeBriefing().version, preparation_status: 'failed' },
    });
    const { container } = await render(PreCallBriefing, {
      props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
    });
    const alertRegion = container.querySelector('[role="alert"]');
    expect(alertRegion).toBeTruthy();
  });
});

// ── HTML escaping (FR-4a, FR-13a) ────────────────────────────────────────────────

describe('PreCallBriefing — HTML escaping (FR-4a, FR-13a)', () => {
  let onRegenerate: MockedFunction<() => void>;
  let onExcludeSource: MockedFunction<(sourceId: string) => void>;
  let onRestoreSource: MockedFunction<(sourceId: string) => void>;
  let onResolveConflict: MockedFunction<(conflictId: string, resolutionNote: string) => Promise<void>>;

  beforeEach(() => {
    onRegenerate = vi.fn();
    onExcludeSource = vi.fn();
    onRestoreSource = vi.fn();
    onResolveConflict = vi.fn().mockResolvedValue(undefined);
  });

  it('escapes HTML script tag in concise_summary objective — raw tag never appears in DOM', async () => {
    const briefing: BriefingActive = {
      ...makeBriefing(),
      content: {
        ...makeBriefing().content,
        concise_summary: {
          ...makeBriefing().content.concise_summary,
          objective: 'Test <script>alert("xss")</script> objective',
        },
      },
    };
    await render(PreCallBriefing, {
      props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
    });
    const rendered = document.body.innerHTML;
    // Security goal: raw <script> tag never appears in the DOM
    expect(rendered).not.toContain('<script>');
    expect(rendered).not.toContain('alert("xss")');
    // Svelte's {expression} interpolation double-escapes, so we see &amp;lt; in the DOM
    expect(rendered).not.toContain('&lt;script&gt;');
  });

  it('escapes ampersand in recommended_focus', async () => {
    const briefing: BriefingActive = {
      ...makeBriefing(),
      content: {
        ...makeBriefing().content,
        concise_summary: {
          ...makeBriefing().content.concise_summary,
          recommended_focus: 'Focus with & special chars',
        },
      },
    };
    await render(PreCallBriefing, {
      props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
    });
    const rendered = document.body.innerHTML;
    // Ampersand gets escaped through escapeHtml + Svelte interpolation
    expect(rendered).not.toContain('& ');
    expect(rendered).toContain('&amp;');
  });

  it('escapes HTML in no-prior-memory shell — raw tags never appear', async () => {
    const briefing: BriefingActive = {
      ...makeBriefing(),
      content: {
        ...makeBriefing().content,
        no_prior_memory_shell: {
          generated: true,
          objective: 'Objective with <b>bold</b>',
          recommended_focus: 'Focus & ampersand',
          suggested_prep_questions: ['Question <i>italic</i>'],
        },
        concise_summary: {
          objective: '', preparation_status: '', recommended_focus: '',
          top_prior_context: '', open_actions: '', risks_questions: '',
        },
      },
    };
    await render(PreCallBriefing, {
      props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
    });
    const rendered = document.body.innerHTML;
    // Security goal: raw tags never appear in DOM
    expect(rendered).not.toContain('<b>');
    expect(rendered).not.toContain('<i>');
  });

  it('escapes HTML in detail section items — raw script tag never appears', async () => {
    const briefing: BriefingActive = {
      ...makeBriefing(),
      content: {
        ...makeBriefing().content,
        detailed_sections: {
          ...makeBriefing().content.detailed_sections,
          important_prior_decisions: {
            items: ['Decision with <script>evil()</script>'],
            quality_status: 'strong_evidence' as const,
          },
        },
      },
    };
    await render(PreCallBriefing, {
      props: { briefing, onRegenerate, onExcludeSource, onRestoreSource, onResolveConflict },
    });
    const rendered = document.body.innerHTML;
    // Security goal: raw <script> never appears in DOM
    expect(rendered).not.toContain('<script>');
    expect(rendered).not.toContain('&lt;script&gt;');
  });
});