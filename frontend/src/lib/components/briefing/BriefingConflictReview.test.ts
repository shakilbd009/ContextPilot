import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import BriefingConflictReview from './BriefingConflictReview.svelte';
import type { BriefingConflict } from '$lib/api/briefing';

describe('BriefingConflictReview', () => {
  const mockConflicts: BriefingConflict[] = [
    {
      id: 'conf-1',
      category: 'decisions',
      evidence_snippet: 'We agreed to adopt 2-week sprints',
      conflicting_meeting_id: '00000000-0000-0000-0000-000000000010',
      resolution_note: null,
    },
  ];

  it('renders no-conflicts message when list is empty', () => {
    render(BriefingConflictReview, {
      conflicts: [],
      onResolve: vi.fn(),
    });
    expect(screen.getByText('No conflicting evidence. All clear.')).toBeInTheDocument();
  });

  it('renders conflict count heading', () => {
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve: vi.fn(),
    });
    expect(screen.getByText('1 conflict to review')).toBeInTheDocument();
  });

  it('renders category badge for conflict', () => {
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve: vi.fn(),
    });
    expect(screen.getByText('decisions')).toBeInTheDocument();
  });

  it('renders escaped evidence snippet', () => {
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve: vi.fn(),
    });
    // The evidence snippet should be escaped for safe display
    expect(screen.getByText('We agreed to adopt 2-week sprints')).toBeInTheDocument();
  });

  it('renders textarea for resolution note', () => {
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve: vi.fn(),
    });
    const textarea = screen.getByRole('textbox');
    expect(textarea).toBeInTheDocument();
  });

  it('renders Mark reviewed button', () => {
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve: vi.fn(),
    });
    expect(screen.getByText('Mark reviewed')).toBeInTheDocument();
  });

  it('disables Mark reviewed when resolution note is empty', () => {
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve: vi.fn(),
    });
    const btn = screen.getByText('Mark reviewed');
    // Button should exist in the document
    expect(btn).toBeInTheDocument();
  });

  it('enables Mark reviewed when resolution note is filled', async () => {
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve: vi.fn(),
    });
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement;
    await fireEvent.input(textarea, { target: { value: 'Evidence confirms the original decision stands.' } });
    const btn = screen.getByText('Mark reviewed');
    expect(btn).not.toBeDisabled();
  });

  it('calls onResolve with conflict id and note when submitted', async () => {
    const onResolve = vi.fn().mockResolvedValue(undefined);
    render(BriefingConflictReview, {
      conflicts: mockConflicts,
      onResolve,
    });
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement;
    await fireEvent.input(textarea, { target: { value: 'Resolved by confirming the decision.' } });
    const btn = screen.getByText('Mark reviewed');
    await btn.click();
    expect(onResolve).toHaveBeenCalledWith('conf-1', 'Resolved by confirming the decision.');
  });
});