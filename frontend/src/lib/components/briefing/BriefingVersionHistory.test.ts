import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import BriefingVersionHistory from './BriefingVersionHistory.svelte';
import type { BriefingVersionSummary } from '$lib/api/briefing';

describe('BriefingVersionHistory', () => {
  const mockVersions: BriefingVersionSummary[] = [
    {
      id: 'br-2',
      version_number: 2,
      status: 'active',
      is_active: true,
      result: 'ready',
      preparation_status: 'ready',
      source_count: 2,
      trigger_type: 'manual_regenerate',
      created_at: new Date(Date.now() - 3600000).toISOString(),
    },
    {
      id: 'br-1',
      version_number: 1,
      status: 'superseded',
      is_active: false,
      result: 'ready_with_caveats',
      preparation_status: 'ready_with_caveats',
      source_count: 2,
      trigger_type: 'auto',
      created_at: new Date(Date.now() - 7200000).toISOString(),
    },
    {
      id: 'br-fail-1',
      version_number: 0,
      status: 'superseded',
      is_active: false,
      result: 'failed',
      preparation_status: 'failed',
      source_count: 0,
      trigger_type: 'auto',
      created_at: new Date(Date.now() - 10800000).toISOString(),
    },
  ];

  it('renders heading with version count', () => {
    render(BriefingVersionHistory, {
      versions: mockVersions,
      activeVersionNumber: 2,
      onSelectVersion: vi.fn(),
    });
    expect(screen.getByText('Version history')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('renders empty message when no versions', () => {
    render(BriefingVersionHistory, {
      versions: [],
      activeVersionNumber: null,
      onSelectVersion: vi.fn(),
    });
    expect(screen.getByText('No versions yet.')).toBeInTheDocument();
  });

  it('renders active badge for active version', () => {
    render(BriefingVersionHistory, {
      versions: mockVersions,
      activeVersionNumber: 2,
      onSelectVersion: vi.fn(),
    });
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('renders Ready badge for ready result', () => {
    render(BriefingVersionHistory, {
      versions: [mockVersions[0]],
      activeVersionNumber: 2,
      onSelectVersion: vi.fn(),
    });
    expect(screen.getByText('Ready')).toBeInTheDocument();
  });

  it('renders Ready with caveats badge for ready_with_caveats result', () => {
    render(BriefingVersionHistory, {
      versions: [mockVersions[1]],
      activeVersionNumber: 2,
      onSelectVersion: vi.fn(),
    });
    expect(screen.getByText('Ready with caveats')).toBeInTheDocument();
  });

  it('renders Failed badge for failed result', () => {
    render(BriefingVersionHistory, {
      versions: [mockVersions[2]],
      activeVersionNumber: 2,
      onSelectVersion: vi.fn(),
    });
    expect(screen.getByText('Failed')).toBeInTheDocument();
  });

  it('renders version details trigger label', () => {
    render(BriefingVersionHistory, {
      versions: [mockVersions[1]],
      activeVersionNumber: 2,
      onSelectVersion: vi.fn(),
    });
    // Click to expand details
    const allToggles = screen.getAllByRole('button');
    allToggles[0].click();
    // The Trigger row in expanded details
    expect(screen.getByText('Auto-generated')).toBeInTheDocument();
  });

  it('renders source count per version', () => {
    render(BriefingVersionHistory, {
      versions: [mockVersions[0]],
      activeVersionNumber: 2,
      onSelectVersion: vi.fn(),
    });
    expect(screen.getByText('2 sources')).toBeInTheDocument();
  });

  it('calls onSelectVersion with version number when view button clicked', async () => {
    const onSelectVersion = vi.fn();
    render(BriefingVersionHistory, {
      versions: [mockVersions[1]],
      activeVersionNumber: 2,
      onSelectVersion,
    });
    // Toggle to expand the details
    const toggle = screen.getAllByRole('button')[0];
    await toggle.click();
    const viewBtn = await screen.findByText('View this version');
    await viewBtn.click();
    expect(onSelectVersion).toHaveBeenCalledWith(mockVersions[1].version_number);
  });
});