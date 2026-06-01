// GET /api/meetings/[meetingId]/memory/conflicts
// Returns pending conflict review queue.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const STUB_CONFLICTS = [
  {
    id: 'conflict-1',
    meeting_id: '00000000-0000-0000-0000-000000000001',
    memory_version_id: 'mem-ver-2',
    conflicting_items: {
      current: [
        {
          id: 'dec-curr-1',
          item_text: 'Adopt 2-week sprint cadence for Q3',
          category: 'decisions',
          quality_status: 'strong_evidence',
          evidence: [
            {
              snippet: 'We agreed to stick with 2-week sprints',
              source_type: 'transcript' as const,
              source_location: { type: 'char_offset' as const, start: 120, end: 170 },
              source_meeting_id: '00000000-0000-0000-0000-000000000001',
            },
          ],
        },
      ],
      prior: [
        {
          id: 'dec-prior-1',
          item_text: 'Switch to 1-week sprint cadence',
          category: 'decisions',
          quality_status: 'strong_evidence',
          evidence: [
            {
              snippet: 'Previous decision to move to 1-week sprints',
              source_type: 'prior_memory_reference' as const,
              source_location: { type: 'char_offset' as const, start: 0, end: 60 },
              source_meeting_id: '00000000-0000-0000-0000-000000000002',
            },
          ],
        },
      ],
    },
    quality_status: 'conflicting_evidence',
    review_status: 'pending' as const,
    resolution_note: null,
    resolved_at: null,
    resolved_by: null,
    created_at: new Date().toISOString(),
  },
];

export const GET: RequestHandler = async ({ params, fetch }) => {
  const meetingId = params.meetingId;
  void fetch;

  return json(STUB_CONFLICTS);
};