// GET /api/upcoming/[meetingId]/briefing
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md (BRD-04 integration)
// Returns the active/latest briefing version for an upcoming meeting.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const STUB_SOURCES = [
  {
    source_meeting_id: '00000000-0000-0000-0000-000000000010',
    relatedness_reasons: ['Same participant: alice@example.com', 'Similar title: Q2 Planning Session'],
    quality_status: 'strong_evidence' as const,
  },
  {
    source_meeting_id: '00000000-0000-0000-0000-000000000011',
    relatedness_reasons: ['Same organization: Acme Corp', 'Close chronology: 28 days ago'],
    quality_status: 'weak_evidence' as const,
  },
];

const STUB_CONTENT = {
  concise_summary: {
    objective: 'Review Q3 roadmap priorities and confirm resource allocation plan',
    preparation_status: 'ready_with_caveats',
    recommended_focus: 'Confirm sprint velocity assumptions with PM; clarify Acme dependency',
    top_prior_context: 'Q2 planning identified 3 key blockers; 1 remains unresolved (vendor API)',
    open_actions: '2 pending: Alice to update template, Bob to schedule follow-up review',
    risks_questions: 'Tech debt capacity — budget approval still outstanding; open question on Q3 scope',
  },
  detailed_sections: {
    previous_relevant_context: {
      statement: 'Last session covered sprint 23 retro and Q2 resource planning. Three blockers were raised; one (vendor API) remains open.',
      quality_status: 'strong_evidence' as const,
    },
    important_prior_decisions: {
      items: [
        'Adopt 2-week sprint cadence for Q3',
        'Target 80% test coverage before Q3 release',
      ],
      quality_status: 'strong_evidence' as const,
    },
    open_action_items: {
      items: [
        'Alice — update sprint planning template with new velocity assumptions (due: 2026-06-15)',
        'Bob — schedule follow-up review in 4 weeks',
      ],
      quality_status: 'weak_evidence' as const,
    },
    unresolved_risks_blockers: {
      items: [
        'Third-party API deprecation may affect integration tests',
      ],
      quality_status: 'weak_evidence' as const,
    },
    open_questions: {
      items: [
        'Should we allocate dedicated capacity for tech debt reduction?',
        'Has Acme confirmed their Q3 timeline?',
      ],
      quality_status: 'insufficient_evidence' as const,
    },
    stakeholder_notes: {
      items: [],
      quality_status: 'strong_evidence' as const,
    },
    suggested_questions: {
      items: [
        'What is the current status of the Acme dependency?',
        'Are we still on track for the Q3 release date?',
        'Who owns the tech debt prioritization decision?',
      ],
    },
    suggested_agenda: {
      items: [
        'Q3 roadmap review (15 min)',
        'Resource allocation confirmation (10 min)',
        'Open blockers / Acme update (10 min)',
        'Tech debt capacity discussion (10 min)',
        'Wrap-up and action items (5 min)',
      ],
    },
  },
  sources: STUB_SOURCES,
  no_prior_memory_shell: null,
};

const STUB_ACTIVE_BRIEFING = {
  version: {
    id: 'br-1',
    version_number: 1,
    status: 'active' as const,
    is_active: true,
    result: 'ready_with_caveats' as const,
    preparation_status: 'ready_with_caveats' as const,
    source_count: 2,
    trigger_type: 'auto' as const,
    created_at: new Date().toISOString(),
  },
  content: STUB_CONTENT,
  is_stale: false,
  stale_source_count: 0,
  excluded_sources: [],
  conflicts: [],
};

export const GET: RequestHandler = async ({ params, fetch }) => {
  void fetch;
  return json(STUB_ACTIVE_BRIEFING);
};