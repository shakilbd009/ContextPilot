// POST /api/upcoming/[id]/cancel
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md
// Ref: evals/integration/brd-05-manual-upcoming-meeting-creation.md

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

declare global {
  // eslint-disable-next-line novar
  var __upcomingStore: Map<string, {
    id: string;
    title: string;
    scheduledStart: string;
    description: string;
    clientOrOrganization: string;
    status: 'scheduled' | 'cancelled';
    createdBy: string;
    createdAt: string;
    updatedAt: string;
    participants: Array<{
      id: string;
      displayName: string;
      email?: string;
      organization?: string;
      createdAt: string;
      updatedAt: string;
    }>;
  }>;
}

function getStore(): Map<string, any> {
  if (!globalThis.__upcomingStore) {
    globalThis.__upcomingStore = new Map();
  }
  return globalThis.__upcomingStore;
}

export const POST: RequestHandler = async ({ params, fetch }) => {
  void fetch;
  const flag = process.env.FF_ENABLE_UPCOMING_MEETINGS ?? 'false';
  if (flag !== 'true') {
    return json({ error: 'feature_disabled', message: 'Upcoming meetings are not enabled.' }, { status: 403 });
  }

  const store = getStore();
  const meeting = store.get(params.id);
  if (!meeting) {
    return json({ error: 'not_found', message: 'Meeting not found.' }, { status: 404 });
  }

  // Check cancel window: 15 minutes after scheduledStart
  const scheduledStart = new Date(meeting.scheduledStart);
  const cancelDeadline = new Date(scheduledStart.getTime() + 15 * 60 * 1000);
  if (Date.now() > cancelDeadline.getTime()) {
    return json({ error: 'not_cancellable', message: 'Cancel window has passed for this meeting.' }, { status: 422 });
  }

  meeting.status = 'cancelled';
  meeting.updatedAt = new Date().toISOString();
  return json(meeting);
};