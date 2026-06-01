// Briefing API client — frontend calls the SvelteKit API route
// (/api/upcoming/{meetingId}/briefing) which proxies to the Go backend.
// All feature-flag enforcement lives server-side.

export type BriefingPreparationStatus =
  | 'generating'
  | 'ready'
  | 'ready_with_caveats'
  | 'no_prior_memory'
  | 'stale'
  | 'failed'
  | 'regenerating';

export type BriefingResult = 'ready' | 'ready_with_caveats' | 'no_prior_memory' | 'failed';

export type BriefingStatus = 'active' | 'superseded';

export type TriggerType = 'auto' | 'manual_regenerate';

// ── Quality status (BRD-03 conventions) ───────────────────────────────────────

export type QualityStatus = 'strong_evidence' | 'weak_evidence' | 'insufficient_evidence' | 'conflicting_evidence';

// ── Source relatedness ────────────────────────────────────────────────────────

export interface BriefingSource {
  source_meeting_id: string;
  relatedness_reasons: string[];
  quality_status: QualityStatus;
}

// ── Brief content ─────────────────────────────────────────────────────────────

export interface ConciseSummary {
  objective: string;
  preparation_status: string;
  recommended_focus: string;
  top_prior_context: string;
  open_actions: string;
  risks_questions: string;
}

// ── Detailed sections ─────────────────────────────────────────────────────────

export interface CategoryStatement {
  statement: string;
  quality_status: QualityStatus;
}

export interface CategoryItems {
  items: string[];
  quality_status: QualityStatus;
}

export interface DetailedSections {
  previous_relevant_context: CategoryStatement;
  important_prior_decisions: CategoryItems;
  open_action_items: CategoryItems;
  unresolved_risks_blockers: CategoryItems;
  open_questions: CategoryItems;
  stakeholder_notes: CategoryItems;
  suggested_questions: { items: string[] };
  suggested_agenda: { items: string[] };
}

// ── No-prior-memory shell ─────────────────────────────────────────────────────

export interface NoPriorMemoryShell {
  generated: boolean;
  objective: string;
  recommended_focus: string;
  suggested_prep_questions: string[];
}

// ── Full briefing content ──────────────────────────────────────────────────────

export interface BriefingContent {
  concise_summary: ConciseSummary;
  detailed_sections: DetailedSections;
  sources: BriefingSource[];
  no_prior_memory_shell: NoPriorMemoryShell | null;
}

// ── API shapes ───────────────────────────────────────────────────────────────

export interface BriefingVersionSummary {
  id: string;
  version_number: number;
  status: BriefingStatus;
  is_active: boolean;
  result: BriefingResult;
  preparation_status: BriefingPreparationStatus;
  source_count: number;
  trigger_type: TriggerType;
  created_at: string;
}

export interface BriefingActive {
  version: BriefingVersionSummary;
  content: BriefingContent;
  is_stale: boolean;
  stale_source_count: number;
  excluded_sources: ExcludedSource[];
  conflicts: BriefingConflict[];
}

export interface BriefingVersion {
  id: string;
  version_number: number;
  status: BriefingStatus;
  is_active: boolean;
  result: BriefingResult;
  preparation_status: BriefingPreparationStatus;
  source_count: number;
  trigger_type: TriggerType;
  content: BriefingContent;
  created_at: string;
}

export interface ExcludedSource {
  source_meeting_id: string;
  excluded_at: string;
  restored_at: string | null;
  relatedness_reasons: string[];
  prior_meeting_title?: string;
}

// ── Conflict ────────────────────────────────────────────────────────────────────

export interface BriefingConflict {
  id: string;
  category: string;
  evidence_snippet: string;
  conflicting_meeting_id: string;
  resolution_note: string | null;
}

// ── API helpers ───────────────────────────────────────────────────────────────

async function briefingFetch<T>(
  path: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: T } | { ok: false; status: number; message: string }> {
  const res = await fetch_fn(path);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    return { ok: false, status: res.status, message: body?.message ?? res.statusText };
  }
  const data = await res.json();
  return { ok: true, data };
}

export async function getActiveBriefing(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: BriefingActive } | { ok: false; status: number; message: string }> {
  return briefingFetch<BriefingActive>(`/api/upcoming/${meetingId}/briefing`, fetch_fn);
}

export async function getBriefingVersions(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: BriefingVersionSummary[] } | { ok: false; status: number; message: string }> {
  return briefingFetch<BriefingVersionSummary[]>(`/api/upcoming/${meetingId}/briefing/versions`, fetch_fn);
}

export async function getBriefingVersion(
  meetingId: string,
  versionNumber: number,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: BriefingVersion } | { ok: false; status: number; message: string }> {
  return briefingFetch<BriefingVersion>(`/api/upcoming/${meetingId}/briefing/versions/${versionNumber}`, fetch_fn);
}

export async function regenerateBriefing(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true } | { ok: false; status: number; message: string }> {
  const res = await fetch_fn(`/api/upcoming/${meetingId}/briefing/regenerate`, { method: 'POST' });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    return { ok: false, status: res.status, message: body?.message ?? res.statusText };
  }
  return { ok: true };
}

export async function excludeBriefingSource(
  meetingId: string,
  sourceId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true } | { ok: false; status: number; message: string }> {
  const res = await fetch_fn(`/api/upcoming/${meetingId}/briefing/sources/${sourceId}/exclude`, { method: 'POST' });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    return { ok: false, status: res.status, message: body?.message ?? res.statusText };
  }
  return { ok: true };
}

export async function restoreBriefingSource(
  meetingId: string,
  sourceId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true } | { ok: false; status: number; message: string }> {
  const res = await fetch_fn(`/api/upcoming/${meetingId}/briefing/sources/${sourceId}/restore`, { method: 'POST' });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    return { ok: false, status: res.status, message: body?.message ?? res.statusText };
  }
  return { ok: true };
}