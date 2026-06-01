// POST /api/upcoming/[meetingId]/briefing/regenerate
// Triggers async regeneration of the briefing.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, fetch }) => {
  void fetch;
  // Stub: in production this queues an async job.
  // Return 202 Accepted to indicate the request was received.
  return json({ message: 'Regeneration queued.' }, { status: 202 });
};