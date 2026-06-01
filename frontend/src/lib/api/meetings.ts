// API client for meeting operations.
// Calls the SvelteKit API route (/api/meetings) which proxies to the Go backend.
// All feature-flag enforcement lives server-side in the Go handler.

export interface ParticipantInput {
  displayName: string;
  email?: string;
  organization?: string;
  role?: string;
}

export interface CreateMeetingPayload {
  title: string;
  completedAt: string; // ISO 8601
  participants: ParticipantInput[];
  transcript?: string;
  notes?: string;
  idempotencyToken: string;
}

export interface MeetingCreated {
  id: string;
  redirect: string;
}

export interface ValidationError {
  field: string;
  message: string;
}

export interface ApiError {
  message?: string;
  errors?: ValidationError[];
  error?: string;
  meetingId?: string;
  originalOutcome?: string;
}

export async function createMeeting(
  payload: CreateMeetingPayload,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: MeetingCreated } | { ok: false; status: number; data: ApiError }> {
  const res = await fetch_fn('/api/meetings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  const data = await res.json().catch(() => ({}));

  if (res.ok) {
    return { ok: true, data };
  }

  return { ok: false, status: res.status, data };
}