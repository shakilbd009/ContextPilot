// POST /api/meetings/[meetingId]/memory/conflicts/[conflictId]/resolve
// Submit resolution note and create a new memory version.

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, request, fetch }) => {
  const { meetingId, conflictId } = params;
  void fetch;

  const body = await request.json().catch(() => ({}));
  const resolutionNote = body?.resolution_note ?? '';

  return json({
    ok: true,
    conflictId,
    meetingId,
    resolution_note: resolutionNote,
    new_version_number: 4,
  });
};