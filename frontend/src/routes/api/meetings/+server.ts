import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';
import { getCookie } from '$lib/utils/cookies';

// ── Feature flag ────────────────────────────────────────────────
const FF_ENABLED = () => env.FF_ENABLE_MANUAL_MEETING_IMPORT === 'true';

// ── Helpers ─────────────────────────────────────────────────────
function uuidRegex(s: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s);
}

function makeError(field: string, message: string) {
  return { field, message };
}

// ── GET /api/meetings ────────────────────────────────────────────
// AC-11: paginated meetings list for frontend load function.
// Proxies to Go backend which implements offset-based pagination.
// Response shape: { meetings: Meeting[], total: number, page: number, limit: number }
export const GET: RequestHandler = async ({ fetch, url, request }) => {
  const page = Math.max(1, parseInt(url.searchParams.get('page') ?? '1', 10));
  const limit = Math.min(100, Math.max(1, parseInt(url.searchParams.get('limit') ?? '20', 10)));

  const SERVER_URL = env.SERVER_URL ?? 'http://localhost:3000';

  // Backend requires X-User-ID header (Phase 1 auth). Prefer the request
  // header when present (e.g. set by an upstream middleware); fall back to
  // the cookie that the demo login flow sets. Without this forward, the
  // backend returns 401 and the list page renders the "Failed to load
  // meetings" error branch.
  const userId = request.headers.get('X-User-ID') ?? getCookie(request, 'X-User-ID') ?? '';

  try {
    const res = await fetch(
      `${SERVER_URL}/api/v1/meetings?page=${page}&limit=${limit}`,
      {
        credentials: 'include',
        headers: userId ? { 'X-User-ID': userId } : {},
      }
    );

    if (!res.ok) {
      return json({ meetings: [], total: 0, page, limit }, { status: res.status });
    }

    const data = await res.json();
    return json(data, { status: 200 });
  } catch {
    return json({ meetings: [], total: 0, page, limit }, { status: 200 });
  }
};

// ── POST /api/meetings ──────────────────────────────────────────
export const POST: RequestHandler = async ({ request }) => {
  // Feature flag check
  if (!FF_ENABLED()) {
    return json({ error: 'disabled' }, { status: 403 });
  }

  let body: Record<string, unknown>;
  try {
    body = await request.json();
  } catch {
    return json({ message: 'Invalid JSON body' }, { status: 400 });
  }

  const { title, completedAt, participants, transcript, notes, idempotencyToken } = body as {
    title?: string;
    completedAt?: string;
    participants?: Array<{ displayName?: string; email?: string; organization?: string; role?: string }>;
    transcript?: string;
    notes?: string;
    idempotencyToken?: string;
  };

  const errors: ReturnType<typeof makeError>[] = [];

  // ── Title ────────────────────────────────────────────────────
  const titleVal = (title ?? '').trim();
  if (!titleVal) {
    errors.push(makeError('title', 'Title is required'));
  } else if (titleVal.length > 500) {
    errors.push(makeError('title', 'Title must be 500 characters or fewer'));
  }

  // ── completedAt ─────────────────────────────────────────────
  if (!completedAt) {
    errors.push(makeError('completedAt', 'Completed date/time is required'));
  } else if (isNaN(Date.parse(completedAt))) {
    errors.push(makeError('completedAt', 'Invalid ISO 8601 date/time'));
  }

  // ── Participants ─────────────────────────────────────────────
  if (!Array.isArray(participants) || participants.length === 0) {
    errors.push(makeError('participants', 'At least one participant is required'));
  } else {
    for (let i = 0; i < participants.length; i++) {
      const p = participants[i];
      const name = (p.displayName ?? '').trim();
      if (!name) {
        errors.push(makeError(`participants[${i}].displayName`, 'Display name is required'));
      }
    }
  }

  // ── Content (transcript + notes) ────────────────────────────
  const transcriptVal = (transcript ?? '').trim();
  const notesVal = (notes ?? '').trim();
  const combinedLen = transcriptVal.length + notesVal.length;
  if (!transcriptVal && !notesVal) {
    errors.push(makeError('content', 'Either transcript or notes is required'));
  } else if (combinedLen > 50_000) {
    errors.push(makeError('content', `Combined content exceeds 50,000 character limit (${combinedLen.toLocaleString()} entered)`));
  }

  // ── Idempotency token ────────────────────────────────────────
  if (!idempotencyToken || !uuidRegex(idempotencyToken)) {
    return json({ message: 'Invalid or missing idempotencyToken' }, { status: 400 });
  }

  if (errors.length > 0) {
    return json({ errors, values: { title: titleVal, completedAt, participants, transcript: transcriptVal, notes: notesVal } }, { status: 400 });
  }

  // ── Forward to Go backend ─────────────────────────────────────
  const SERVER_URL = env.SERVER_URL ?? 'http://localhost:3000';
  // Prefer X-User-ID from request header (set by authenticated clients),
  // fall back to cookie set during login for demo/demo flows
  const userId = request.headers.get('X-User-ID') ?? getCookie(request, 'X-User-ID') ?? '';

  const res = await fetch(`${SERVER_URL}/api/v1/meetings`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-User-ID': userId,
    },
    body: JSON.stringify(body),
  });

  // Forward 201/400/401/403/409 as-is; map 500→500
  if (res.status === 201) {
    const data = await res.json();
    return json(data, { status: 201 });
  }
  if (res.status === 400 || res.status === 401 || res.status === 403 || res.status === 409) {
    const data = await res.json();
    return json(data, { status: res.status });
  }
  if (res.status === 500) {
    return json({ message: 'Internal server error' }, { status: 500 });
  }

  // Unexpected status — pass through
  return json({ message: 'Unexpected response from backend' }, { status: 502 });
};