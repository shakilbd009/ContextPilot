// GET /api/meetings/[meetingId]/memory/versions/[versionNumber]
// Returns a specific memory version.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const FAKE_MEETING_ID = '00000000-0000-0000-0000-000000000001';

const STUB_MEMORY_CONTENT = {
  summary: {
    statement: 'Team agreed on Q3 roadmap priorities and resource allocation plan.',
    quality_status: 'strong_evidence',
    items: [],
  },
  decisions: { items: [] },
  action_items: { items: [] },
  risks_blockers: { items: [] },
  open_questions: { items: [] },
  stakeholder_notes: { items: [] },
  next_recommended_focus: {
    statement: 'Follow up on Q3 resource allocation.',
    quality_status: 'strong_evidence',
    supporting_item_ids: [],
    evidence: [],
  },
  briefing_readiness: {
    ready: false,
    core_categories_ready: false,
    blocking_conflicts: [],
    insufficient_categories: ['open_questions'],
  },
};

export const GET: RequestHandler = async ({ params, fetch }) => {
  const { meetingId, versionNumber } = params;
  void fetch;

  const version = parseInt(versionNumber, 10);

  return json({
    id: `mem-ver-${version}`,
    meeting_id: meetingId ?? FAKE_MEETING_ID,
    version_number: version,
    status: version === 2 ? 'active' : 'superseded',
    is_active: version === 2,
    content: STUB_MEMORY_CONTENT,
    created_at: new Date().toISOString(),
    created_by: null,
    trigger_type: version === 1 ? 'import' : 'reprocess',
  });
};