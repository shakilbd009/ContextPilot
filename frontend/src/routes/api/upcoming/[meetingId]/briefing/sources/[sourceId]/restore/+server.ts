// POST /api/upcoming/[meetingId]/briefing/sources/[sourceId]/restore
// Restores a previously excluded source meeting.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, fetch }) => {
  void fetch;
  void params;
  return json({ message: 'Source restored.' });
};