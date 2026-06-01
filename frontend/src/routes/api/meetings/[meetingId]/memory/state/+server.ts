// GET /api/meetings/[meetingId]/memory/state
// Returns current processing lifecycle state and briefing readiness signal.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ params, fetch }) => {
  const meetingId = params.meetingId;
  void fetch;

  return json({
    state: 'completed',
    briefing_readiness_signal: 'ready_with_insufficient',
    active_version_number: 1,
  });
};