import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import BriefingSourceList from './BriefingSourceList.svelte';
import type { BriefingSource } from '$lib/api/briefing';

describe('BriefingSourceList', () => {
  const mockSources: BriefingSource[] = [
    {
      source_meeting_id: '00000000-0000-0000-0000-000000000010',
      relatedness_reasons: ['Same participant: alice@example.com', 'Similar title: Q2 Planning'],
      quality_status: 'strong_evidence',
    },
    {
      source_meeting_id: '00000000-0000-0000-0000-000000000011',
      relatedness_reasons: ['Same organization: Acme Corp', 'Close chronology: 28 days ago'],
      quality_status: 'weak_evidence',
    },
  ];

  it('renders heading', () => {
    render(BriefingSourceList, {
      sources: mockSources,
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    expect(screen.getByText('Source meetings')).toBeInTheDocument();
  });

  it('renders each source with a toggle button', () => {
    render(BriefingSourceList, {
      sources: mockSources,
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    expect(screen.getAllByRole('button').length).toBeGreaterThanOrEqual(2);
  });

  it('renders strong evidence badge for strong_evidence sources', () => {
    render(BriefingSourceList, {
      sources: [mockSources[0]],
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    expect(screen.getByText('Strong')).toBeInTheDocument();
  });

  it('renders weak evidence badge for weak_evidence sources', () => {
    render(BriefingSourceList, {
      sources: [mockSources[1]],
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    expect(screen.getByText('Weak')).toBeInTheDocument();
  });

  it('shows exclude button for non-excluded sources', () => {
    render(BriefingSourceList, {
      sources: mockSources,
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    expect(screen.getAllByText('Exclude').length).toBe(2);
  });

  it('shows excluded label for already-excluded sources', () => {
    render(BriefingSourceList, {
      sources: [mockSources[0]],
      onExclude: vi.fn(),
      excludedSourceIds: new Set(['00000000-0000-0000-0000-000000000010']),
    });
    expect(screen.getByText('Excluded')).toBeInTheDocument();
  });

  it('calls onExclude with source id when exclude is clicked', async () => {
    const onExclude = vi.fn();
    render(BriefingSourceList, {
      sources: [mockSources[0]],
      onExclude,
      excludedSourceIds: new Set<string>(),
    });
    const excludeBtn = screen.getByText('Exclude');
    await excludeBtn.click();
    expect(onExclude).toHaveBeenCalledWith('00000000-0000-0000-0000-000000000010');
  });

  it('renders empty message when no sources', () => {
    render(BriefingSourceList, {
      sources: [],
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    expect(screen.getByText('No source meetings selected.')).toBeInTheDocument();
  });

  it('expands source details when toggle is clicked', async () => {
    render(BriefingSourceList, {
      sources: [mockSources[0]],
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    const toggle = screen.getAllByRole('button')[0];
    await toggle.click();
    expect(screen.getByText('Why this meeting was selected:')).toBeInTheDocument();
  });

  it('renders relatedness reasons in expanded details', async () => {
    render(BriefingSourceList, {
      sources: [mockSources[0]],
      onExclude: vi.fn(),
      excludedSourceIds: new Set<string>(),
    });
    const toggle = screen.getAllByRole('button')[0];
    await toggle.click();
    expect(screen.getByText('Same participant: alice@example.com')).toBeInTheDocument();
  });
});