import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import ExcludedSources from './ExcludedSources.svelte';
import type { ExcludedSource } from '$lib/api/briefing';

describe('ExcludedSources', () => {
  const mockExcluded: ExcludedSource[] = [
    {
      source_meeting_id: '00000000-0000-0000-0000-000000000010',
      excluded_at: '2026-05-23T10:00:00Z',
      restored_at: null,
      relatedness_reasons: ['Same participant: alice@example.com'],
      prior_meeting_title: 'Q2 Planning Session',
    },
  ];

  it('renders nothing when excludedSources is empty', () => {
    const { container } = render(ExcludedSources, {
      excludedSources: [],
      onRestore: vi.fn(),
    });
    expect(container.children.length).toBe(0);
  });

  it('renders toggle button with count', () => {
    render(ExcludedSources, {
      excludedSources: mockExcluded,
      onRestore: vi.fn(),
    });
    expect(screen.getByText('1 source excluded')).toBeInTheDocument();
  });

  it('renders plural form for multiple excluded sources', () => {
    render(ExcludedSources, {
      excludedSources: [
        { ...mockExcluded[0] },
        { ...mockExcluded[0], source_meeting_id: '00000000-0000-0000-0000-000000000011' },
      ],
      onRestore: vi.fn(),
    });
    expect(screen.getByText('2 sources excluded')).toBeInTheDocument();
  });

  it('shows excluded list when expanded', async () => {
    render(ExcludedSources, {
      excludedSources: mockExcluded,
      onRestore: vi.fn(),
    });
    const toggle = screen.getByRole('button');
    await toggle.click();
    expect(screen.getByText('Q2 Planning Session')).toBeInTheDocument();
  });

  it('calls onRestore with source id when restore is clicked', async () => {
    const onRestore = vi.fn();
    render(ExcludedSources, {
      excludedSources: mockExcluded,
      onRestore,
    });
    const toggle = screen.getByRole('button');
    await toggle.click();
    const restoreBtn = screen.getByText('Restore');
    await restoreBtn.click();
    expect(onRestore).toHaveBeenCalledWith('00000000-0000-0000-0000-000000000010');
  });

  it('renders relatedness reasons in expanded list', async () => {
    render(ExcludedSources, {
      excludedSources: mockExcluded,
      onRestore: vi.fn(),
    });
    const toggle = screen.getByRole('button');
    await toggle.click();
    expect(screen.getByText('Same participant: alice@example.com')).toBeInTheDocument();
  });
});