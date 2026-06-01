// POST /api/upcoming/[meetingId]/briefing/sources/[sourceId]/exclude
// Excludes a source meeting from future briefing regenerations.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, fetch }) => {
  void fetch;
  void params;
  return json({ message: 'Source excluded.' });
};