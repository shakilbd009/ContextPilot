// GET/POST /api/upcoming — list and create upcoming meetings
// Feature flag: FF_ENABLE_UPCOMING_MEETINGS
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md
// Ref: evals/integration/brd-05-manual-upcoming-meeting-creation.md
// Ref: evals/unit/brd-05-manual-upcoming-meeting-creation.md

import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { validateCreateUpcoming } from '$lib/validation/upcoming';

// Module-level store shared across all /api/upcoming/* routes via globalThis.
// SvelteKit route modules for different route segments do NOT share module scope,
// so we must use globalThis to persist data within a running server process.
declare global {
  // eslint-disable-next-line no-var
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

// Seed a sample meeting for demo (only if store is empty)
const DEMO_ID = '11111111-1111-1111-1111-111111111111';

export const GET: RequestHandler = async ({ url, fetch }) => {
  void fetch;
  const flag = process.env.FF_ENABLE_UPCOMING_MEETINGS ?? 'false';
  if (flag !== 'true') {
    return json({ error: 'feature_disabled', message: 'Upcoming meetings are not enabled.' }, { status: 403 });
  }

  const store = getStore();

  // Seed demo data if store is empty
  if (store.size === 0) {
    const demoScheduledStart = new Date(Date.now() + 2 * 24 * 60 * 60 * 1000).toISOString();
    store.set(DEMO_ID, {
      id: DEMO_ID,
      title: 'Q3 Planning Review',
      scheduledStart: demoScheduledStart,
      description: 'Review Q3 roadmap priorities and confirm resource allocation plan',
      clientOrOrganization: 'Acme Corp',
      status: 'scheduled',
      createdBy: 'user-1',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      participants: [
        { id: 'p-1', displayName: 'Alice Smith', email: 'alice@example.com', organization: 'Acme Corp', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
        { id: 'p-2', displayName: 'Bob Jones', email: 'bob@example.com', organization: 'Beta LLC', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
      ],
    });
  }

  const status = url.searchParams.get('status') ?? 'scheduled';
  const meetings = Array.from(store.values())
    .filter(m => status === 'all' ? true : m.status === status)
    .sort((a, b) => new Date(a.scheduledStart).getTime() - new Date(b.scheduledStart).getTime());

  return json({ meetings, total: meetings.length });
};

export const POST: RequestHandler = async ({ request, fetch }) => {
  void fetch;
  const flag = process.env.FF_ENABLE_UPCOMING_MEETINGS ?? 'false';
  if (flag !== 'true') {
    return json({ error: 'feature_disabled', message: 'Upcoming meetings are not enabled.' }, { status: 403 });
  }

  try {
    const body = await request.json();

    // ── Static validation (CWE-20 — finding F5 of t_793ea842) ──
    // Length caps, control character rejection, HTML-tag rejection,
    // type/format checks. Pure deterministic rules — testable without
    // a clock. The shared validator is the security boundary; the
    // time-window check below is the only handler-local rule.
    const errors = validateCreateUpcoming(body);

    // ── Time-window validation (non-deterministic, lives here) ──
    const { scheduledStart } = body;
    if (typeof scheduledStart === 'string') {
      const scheduled = new Date(scheduledStart);
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

    const id = crypto.randomUUID();
    const now = new Date().toISOString();
    const meeting = {
      id,
      title: title.trim(),
      scheduledStart,
      description: description?.trim() || undefined,
      clientOrOrganization: clientOrOrganization?.trim() || undefined,
      status: 'scheduled' as const,
      createdBy: 'user-1', // stub: would come from auth in production
      createdAt: now,
      updatedAt: now,
      participants: (participants ?? []).map((p: { displayName?: string; email?: string; organization?: string }) => ({
        id: crypto.randomUUID(),
        displayName: p.displayName?.trim() ?? '',
        email: p.email?.trim() || undefined,
        organization: p.organization?.trim() || undefined,
        createdAt: now,
        updatedAt: now,
      })),
    };
    const store = getStore();
    store.set(id, meeting);
    return json(meeting, { status: 201 });
  } catch {
    return json({ error: 'invalid_request', message: 'Invalid request body.' }, { status: 400 });
  }
};