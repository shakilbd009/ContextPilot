// GET /api/upcoming/[meetingId]/briefing/versions
// Returns version history list for an upcoming meeting's briefings.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

const STUB_VERSIONS = [
  {
    id: 'br-2',
    version_number: 2,
    status: 'active' as const,
    is_active: true,
    result: 'ready' as const,
    preparation_status: 'ready' as const,
    source_count: 2,
    trigger_type: 'manual_regenerate' as const,
    created_at: new Date(Date.now() - 3600000).toISOString(),
  },
  {
    id: 'br-1',
    version_number: 1,
    status: 'superseded' as const,
    is_active: false,
    result: 'ready_with_caveats' as const,
    preparation_status: 'ready_with_caveats' as const,
    source_count: 2,
    trigger_type: 'auto' as const,
    created_at: new Date(Date.now() - 7200000).toISOString(),
  },
  {
    id: 'br-fail-1',
    version_number: 0,
    status: 'superseded' as const,
    is_active: false,
    result: 'failed' as const,
    preparation_status: 'failed' as const,
    source_count: 0,
    trigger_type: 'auto' as const,
    created_at: new Date(Date.now() - 10800000).toISOString(),
  },
];

export const GET: RequestHandler = async ({ params, fetch }) => {
  void fetch;
  return json(STUB_VERSIONS);
};