// GET/PATCH/DELETE /api/upcoming/[id]
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md
// Ref: evals/integration/brd-05-manual-upcoming-meeting-creation.md

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { validateUpdateUpcoming } from '$lib/validation/upcoming';

// Reuse the same in-memory store from +server.ts in the same module scope
// NOTE: SvelteKit server routes in the same route segment share module scope,
// so we can reference the store directly since they share the same module resolution.
// For a real implementation this would be a database — stub here for UI dev.
declare global {
  // Module-level store shared across all /api/upcoming/* routes
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

export const GET: RequestHandler = async ({ params, fetch }) => {
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
  return json(meeting);
};

export const PATCH: RequestHandler = async ({ params, request, fetch }) => {
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

  // Check edit window: 15 minutes after scheduledStart
  const scheduledStart = new Date(meeting.scheduledStart);
  const editDeadline = new Date(scheduledStart.getTime() + 15 * 60 * 1000);
  if (Date.now() > editDeadline.getTime()) {
    return json({ error: 'not_editable', message: 'Edit window has passed for this meeting.' }, { status: 422 });
  }

  try {
    const body = await request.json();

    // ── Static validation (CWE-20 — finding F5 of t_793ea842) ──
    // Length caps, control character rejection, HTML-tag rejection,
    // type/format checks. Every field is optional on PATCH; the
    // validator only checks fields the caller actually provided.
    const errors = validateUpdateUpcoming(body);

    // ── Time-window validation (non-deterministic, lives here) ──
    const { scheduledStart: newStart } = body;
    if (typeof newStart === 'string') {
      const scheduled = new Date(newStart);
      const minTime = new Date(Date.now() - 15 * 60 * 1000);
      if (Number.isFinite(scheduled.getTime()) && scheduled < minTime) {
        errors.push({
          field: 'scheduledStart',
          message: 'Scheduled start must be at least 15 minutes in the future.',
        });
      }
    }

    if (errors.length > 0) {
      return json({ error: 'validation_failed', errors }, { status: 400 });
    }

    // Safe to read fields now that the validator passed.
    const { title, description, clientOrOrganization, participants } = body as Record<string, any>;

    // Apply updates
    if (title !== undefined) meeting.title = title.trim();
    if (newStart !== undefined) meeting.scheduledStart = newStart;
    if (description !== undefined) meeting.description = description?.trim() || undefined;
    if (clientOrOrganization !== undefined) meeting.clientOrOrganization = clientOrOrganization?.trim() || undefined;
    if (participants !== undefined) {
      const now = new Date().toISOString();
      meeting.participants = participants.map((p: { displayName?: string; email?: string; organization?: string }) => ({
        id: crypto.randomUUID(),
        displayName: p.displayName?.trim() ?? '',
        email: p.email?.trim() || undefined,
        organization: p.organization?.trim() || undefined,
        createdAt: now,
        updatedAt: now,
      }));
    }
    meeting.updatedAt = new Date().toISOString();

    return json(meeting);
  } catch {
    return json({ error: 'invalid_request', message: 'Invalid request body.' }, { status: 400 });
  }
};