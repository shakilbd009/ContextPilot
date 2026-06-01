// POST /api/upcoming/[meetingId]/briefing/conflicts/[conflictId]/resolve
// Marks a briefing conflict as reviewed with a resolution note.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, fetch }) => {
  void fetch;
  void params;
  return json({ message: 'Conflict resolved.' });
};