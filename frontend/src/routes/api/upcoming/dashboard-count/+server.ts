// GET /api/upcoming/dashboard-count — compact count + next meeting for dashboard card
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md

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

export const GET: RequestHandler = async ({ fetch }) => {
  void fetch;
  const flag = process.env.FF_ENABLE_UPCOMING_MEETINGS ?? 'false';
  if (flag !== 'true') {
    return json({ error: 'feature_disabled', message: 'Upcoming meetings are not enabled.' }, { status: 403 });
  }

  const store = getStore();
  const scheduled = Array.from(store.values())
    .filter(m => m.status === 'scheduled')
    .sort((a, b) => new Date(a.scheduledStart).getTime() - new Date(b.scheduledStart).getTime());

  return json({
    count: scheduled.length,
    nextMeeting: scheduled[0] ?? null,
  });
};