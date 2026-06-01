// POST /api/meetings/[meetingId]/memory/reprocess
// Manual retry or reprocess.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, fetch }) => {
  const meetingId = params.meetingId;
  void fetch;

  return json({ ok: true, meetingId });
};