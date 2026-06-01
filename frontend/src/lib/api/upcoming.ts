// API client for upcoming meeting operations.
// Feature flag: VITE_FF_ENABLE_UPCOMING_MEETINGS (browser) / FF_ENABLE_UPCOMING_MEETINGS (server)

export interface ParticipantInput {
  displayName: string;
  email?: string;
  organization?: string;
}

export interface CreateUpcomingMeetingPayload {
  title: string;
  scheduledStart: string; // ISO 8601
  description?: string;
  clientOrOrganization?: string;
  participants?: ParticipantInput[];
}

export type BriefingStatusValue =
  | 'queued'
  | 'generating'
  | 'ready'
  | 'failed'
  | 'stale'
  | 'disabled'
  | 'unavailable'
  | 'unknown'
  | 'absent';

export interface UpcomingMeeting {
  id: string;
  title: string;
  scheduledStart: string;
  description?: string;
  clientOrOrganization?: string;
  status: 'scheduled' | 'cancelled';
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  participants?: UpcomingMeetingParticipant[];
  briefingStatus?: BriefingStatusValue;
  isBriefingStale?: boolean;
}

export interface UpcomingMeetingParticipant {
  id: string;
  displayName: string;
  email?: string;
  organization?: string;
}

export interface ValidationError {
  field: string;
  message: string;
}

export interface ApiError {
  message?: string;
  errors?: ValidationError[];
  error?: string;
  featureDisabled?: boolean;
}

export interface ListUpcomingMeetingsResult {
  meetings: UpcomingMeeting[];
  total: number;
}

// GET /api/upcoming — list upcoming meetings
export async function listUpcomingMeetings(
  fetch_fn: typeof fetch,
  options?: { status?: 'scheduled' | 'cancelled' | 'all' }
): Promise<{ ok: true; data: ListUpcomingMeetingsResult } | { ok: false; status: number; data: ApiError }> {
  const url = new URL('/api/upcoming', window.location.origin);
  if (options?.status && options.status !== 'all') {
    url.searchParams.set('status', options.status);
  }
  const res = await fetch_fn(url.toString());
  const data = await res.json().catch(() => ({}));
  if (res.ok) return { ok: true, data };
  return { ok: false, status: res.status, data };
}

// GET /api/upcoming/[id] — get single upcoming meeting
export async function getUpcomingMeeting(
  id: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: UpcomingMeeting } | { ok: false; status: number; data: ApiError }> {
  const res = await fetch_fn(`/api/upcoming/${id}`);
  const data = await res.json().catch(() => ({}));
  if (res.ok) return { ok: true, data };
  return { ok: false, status: res.status, data };
}

// POST /api/upcoming — create upcoming meeting
export async function createUpcomingMeeting(
  payload: CreateUpcomingMeetingPayload,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: UpcomingMeeting } | { ok: false; status: number; data: ApiError }> {
  const res = await fetch_fn('/api/upcoming', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const data = await res.json().catch(() => ({}));
  if (res.ok) return { ok: true, data };
  return { ok: false, status: res.status, data };
}

// PATCH /api/upcoming/[id] — update upcoming meeting
export async function updateUpcomingMeeting(
  id: string,
  payload: Partial<CreateUpcomingMeetingPayload>,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: UpcomingMeeting } | { ok: false; status: number; data: ApiError }> {
  const res = await fetch_fn(`/api/upcoming/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const data = await res.json().catch(() => ({}));
  if (res.ok) return { ok: true, data };
  return { ok: false, status: res.status, data };
}

// POST /api/upcoming/[id]/cancel — cancel upcoming meeting
export async function cancelUpcomingMeeting(
  id: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: UpcomingMeeting } | { ok: false; status: number; data: ApiError }> {
  const res = await fetch_fn(`/api/upcoming/${id}/cancel`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  });
  const data = await res.json().catch(() => ({}));
  if (res.ok) return { ok: true, data };
  return { ok: false, status: res.status, data };
}

// GET /api/upcoming/dashboard-count — get dashboard count of upcoming meetings
export async function getDashboardUpcomingCount(
  fetch_fn: typeof fetch
): Promise<{ ok: true; count: number; nextMeeting: UpcomingMeeting | null } | { ok: false; status: number; data: ApiError }> {
  const res = await fetch_fn('/api/upcoming/dashboard-count');
  const data = await res.json().catch(() => ({}));
  if (res.ok) return { ok: true, ...data };
  return { ok: false, status: res.status, data };
}