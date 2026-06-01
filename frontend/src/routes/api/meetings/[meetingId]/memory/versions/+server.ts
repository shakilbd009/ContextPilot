// GET /api/meetings/[meetingId]/memory/versions
// Returns version history list for a meeting.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ params, fetch }) => {
  const meetingId = params.meetingId;
  void fetch;

  const now = new Date();
  const versions = [
    {
      id: 'mem-ver-3',
      version_number: 3,
      status: 'superseded' as const,
      is_active: false,
      trigger_type: 'reprocess' as const,
      created_at: new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString(),
      created_by: null,
    },
    {
      id: 'mem-ver-2',
      version_number: 2,
      status: 'active' as const,
      is_active: true,
      trigger_type: 'stale_reprocess' as const,
      created_at: new Date(now.getTime() - 1 * 60 * 60 * 1000).toISOString(),
      created_by: null,
    },
    {
      id: 'mem-ver-1',
      version_number: 1,
      status: 'superseded' as const,
      is_active: false,
      trigger_type: 'import' as const,
      created_at: new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString(),
      created_by: null,
    },
  ];

  return json(versions);
};