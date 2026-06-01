// GET /api/meetings/[meetingId]/memory
// Returns the active memory version with content, briefing readiness, prior-memory inputs.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const FAKE_MEETING_ID = '00000000-0000-0000-0000-000000000001';

// Stub response matching MemoryContent schema from BRD-03.
const STUB_MEMORY = {
  summary: {
    statement: 'Team agreed on Q3 roadmap priorities and resource allocation plan.',
    quality_status: 'strong_evidence',
    items: [],
  },
  decisions: {
    items: [
      {
        id: 'dec-1',
        decision_statement: 'Adopt the proposed sprint cadence for Q3',
        change_status: 'confirms_prior',
        prior_decision_ref: null,
        quality_status: 'strong_evidence',
        evidence: [
          {
            snippet: 'We agreed to stick with 2-week sprints for Q3',
            source_type: 'transcript',
            source_location: { type: 'char_offset', start: 120, end: 170 },
            source_meeting_id: FAKE_MEETING_ID,
          },
        ],
      },
      {
        id: 'dec-2',
        decision_statement: 'Increase test coverage to 80% before Q3 release',
        change_status: 'standalone',
        prior_decision_ref: null,
        quality_status: 'weak_evidence',
        evidence: [
          {
            snippet: 'We need better test coverage going forward',
            source_type: 'notes',
            source_location: { type: 'char_offset', start: 0, end: 50 },
            source_meeting_id: FAKE_MEETING_ID,
          },
        ],
      },
    ],
  },
  action_items: {
    items: [
      {
        id: 'ai-1',
        description: 'Update sprint planning template with new velocity assumptions',
        owner: 'Alice',
        owner_specified: true,
        due_date: '2026-06-15',
        due_date_specified: true,
        status: 'pending',
        quality_status: 'strong_evidence',
        evidence: [
          {
            snippet: 'Alice will own updating the template',
            source_type: 'transcript',
            source_location: { type: 'char_offset', start: 500, end: 550 },
            source_meeting_id: FAKE_MEETING_ID,
          },
        ],
      },
      {
        id: 'ai-2',
        description: 'Schedule follow-up review in 4 weeks',
        owner: null,
        owner_specified: false,
        due_date: null,
        due_date_specified: false,
        status: 'pending',
        quality_status: 'insufficient_evidence',
        evidence: [],
      },
    ],
  },
  risks_blockers: {
    items: [
      {
        id: 'rb-1',
        type: 'risk',
        description: 'Third-party API deprecation may affect integration tests',
        quality_status: 'weak_evidence',
        evidence: [
          {
            snippet: 'The vendor announced sunset for v2 API by end of year',
            source_type: 'notes',
            source_location: { type: 'char_offset', start: 80, end: 140 },
            source_meeting_id: FAKE_MEETING_ID,
          },
        ],
      },
    ],
  },
  open_questions: {
    items: [
      {
        id: 'oq-1',
        question: 'Should we allocate dedicated capacity for tech debt reduction?',
        quality_status: 'strong_evidence',
        evidence: [
          {
            snippet: 'Should we budget time for tech debt in Q3?',
            source_type: 'transcript',
            source_location: { type: 'char_offset', start: 900, end: 950 },
            source_meeting_id: FAKE_MEETING_ID,
          },
        ],
      },
    ],
  },
  stakeholder_notes: {
    items: [
      {
        id: 'sn-1',
        participant_id: '00000000-0000-0000-0000-000000000002',
        preferences: 'Prefers async updates over status meetings',
        concerns: 'Worried about resource constraints in Q3',
        commitments: 'Will lead the sprint planning effort',
        influence_stake: 'Decision-maker',
        quality_status: 'strong_evidence',
        evidence: [
          {
            snippet: 'Bob prefers async updates',
            source_type: 'notes',
            source_location: { type: 'char_offset', start: 0, end: 30 },
            source_meeting_id: FAKE_MEETING_ID,
          },
        ],
      },
    ],
  },
  next_recommended_focus: {
    statement: 'Follow up on Q3 resource allocation with finance team before sprint kickoff.',
    quality_status: 'strong_evidence',
    supporting_item_ids: ['ai-1', 'dec-1'],
    evidence: [
      {
        snippet: 'We need to confirm budget with finance',
        source_type: 'transcript',
        source_location: { type: 'char_offset', start: 1100, end: 1150 },
        source_meeting_id: FAKE_MEETING_ID,
      },
    ],
  },
  briefing_readiness: {
    ready: false,
    core_categories_ready: false,
    blocking_conflicts: [],
    insufficient_categories: ['open_questions'],
  },
};

// Stub active memory response
const STUB_ACTIVE_MEMORY = {
  memory: {
    id: 'mem-ver-1',
    meeting_id: FAKE_MEETING_ID,
    version_number: 1,
    status: 'active',
    is_active: true,
    content: STUB_MEMORY,
    created_at: new Date().toISOString(),
    created_by: null,
    trigger_type: 'import',
  },
  state: 'completed' as const,
  briefing_readiness_signal: 'ready_with_insufficient' as const,
};

// Stub prior memory inputs
const STUB_PRIOR_INPUTS = [
  {
    id: 'pm-1',
    prior_memory_version_id: 'mem-ver-prior-1',
    match_confidence: 'safe_match' as const,
    included_by_user: false,
    excluded_by_user: false,
    prior_meeting_title: 'Q2 Planning Session',
    prior_meeting_completed_at: '2026-04-10T14:00:00Z',
  },
  {
    id: 'pm-2',
    prior_memory_version_id: 'mem-ver-prior-2',
    match_confidence: 'uncertain' as const,
    included_by_user: false,
    excluded_by_user: false,
    prior_meeting_title: 'Sprint 23 Retro',
    prior_meeting_completed_at: '2026-05-02T16:30:00Z',
  },
];

export const GET: RequestHandler = async ({ params, fetch }) => {
  const meetingId = params.meetingId;

  // In production, this would call the Go backend.
  // For now, return the stub so the UI can be exercised.
  void fetch;

  return json({ ...STUB_ACTIVE_MEMORY, prior_inputs: STUB_PRIOR_INPUTS });
};