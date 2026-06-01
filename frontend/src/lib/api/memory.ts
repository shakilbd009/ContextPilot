// Memory API client — frontend calls the SvelteKit API route (/api/meetings/{id}/memory)
// which proxies to the Go backend. All feature-flag enforcement lives server-side.

export type MemoryState =
  | 'not_processed'
  | 'queued'
  | 'processing'
  | 'completed'
  | 'completed_with_insufficient_evidence'
  | 'failed'
  | 'stale'
  | 'reprocessing'
  | 'retrying'
  | 'retry_exhausted';

export type BriefingReadinessSignal =
  | 'ready'
  | 'ready_with_weak'
  | 'ready_with_insufficient'
  | 'needs_review'
  | 'processing'
  | 'reprocessing'
  | 'failed'
  | 'retry_exhausted'
  | 'stale';

export type QualityStatus = 'strong_evidence' | 'weak_evidence' | 'insufficient_evidence' | 'conflicting_evidence';

export type ChangeStatus = 'standalone' | 'confirms_prior' | 'changes_prior' | 'reverses_prior';

export type SourceType = 'transcript' | 'notes' | 'prior_memory_reference';

// ── Evidence ─────────────────────────────────────────────────────────────────

export interface EvidenceItem {
  snippet: string;
  source_type: SourceType;
  source_location: { type: 'char_offset'; start: number; end: number };
  source_meeting_id: string;
}

// ── Category items ────────────────────────────────────────────────────────────

export interface DecisionItem {
  id: string;
  decision_statement: string;
  change_status: ChangeStatus;
  prior_decision_ref: string | null;
  quality_status: QualityStatus;
  evidence: EvidenceItem[];
}

export interface ActionItem {
  id: string;
  description: string;
  owner: string | null;
  owner_specified: boolean;
  due_date: string | null; // ISO 8601 date
  due_date_specified: boolean;
  status: 'pending' | 'in_progress' | 'completed' | 'deferred';
  quality_status: QualityStatus;
  evidence: EvidenceItem[];
}

export interface RiskBlockerItem {
  id: string;
  type: 'risk' | 'blocker';
  description: string;
  quality_status: QualityStatus;
  evidence: EvidenceItem[];
}

export interface OpenQuestionItem {
  id: string;
  question: string;
  quality_status: QualityStatus;
  evidence: EvidenceItem[];
}

export interface StakeholderNoteItem {
  id: string;
  participant_id: string;
  preferences: string | null;
  concerns: string | null;
  commitments: string | null;
  influence_stake: string | null;
  quality_status: QualityStatus;
  evidence: EvidenceItem[];
}

// ── Memory content ────────────────────────────────────────────────────────────

export interface SummaryCategory {
  statement: string;
  quality_status: QualityStatus;
  items: never[]; // summary has no separate items array
}

export interface DecisionsCategory {
  items: DecisionItem[];
}

export interface ActionItemsCategory {
  items: ActionItem[];
}

export interface RisksBlockersCategory {
  items: RiskBlockerItem[];
}

export interface OpenQuestionsCategory {
  items: OpenQuestionItem[];
}

export interface StakeholderNotesCategory {
  items: StakeholderNoteItem[];
}

export interface NextRecommendedFocus {
  statement: string;
  quality_status: QualityStatus;
  supporting_item_ids: string[];
  evidence: EvidenceItem[];
}

export interface BriefingReadiness {
  ready: boolean;
  core_categories_ready: boolean;
  blocking_conflicts: string[]; // conflict IDs
  insufficient_categories: string[]; // category names
}

export interface MemoryContent {
  summary: SummaryCategory;
  decisions: DecisionsCategory;
  action_items: ActionItemsCategory;
  risks_blockers: RisksBlockersCategory;
  open_questions: OpenQuestionsCategory;
  stakeholder_notes: StakeholderNotesCategory;
  next_recommended_focus: NextRecommendedFocus;
  briefing_readiness: BriefingReadiness;
}

// ── API shapes ───────────────────────────────────────────────────────────────

export interface MemoryVersion {
  id: string;
  meeting_id: string;
  version_number: number;
  status: 'active' | 'superseded' | 'conflict_review';
  is_active: boolean;
  content: MemoryContent;
  created_at: string;
  created_by: string | null;
  trigger_type: 'import' | 'reprocess' | 'stale_reprocess' | 'manual_retry' | 'conflict_resolution';
}

export interface ActiveMemory {
  memory: MemoryVersion;
  state: MemoryState;
  briefing_readiness_signal: BriefingReadinessSignal;
  prior_inputs: PriorMemoryInput[];
}

export interface MemoryStateResponse {
  state: MemoryState;
  briefing_readiness_signal: BriefingReadinessSignal;
  active_version_number: number | null;
}

export interface MemoryVersionSummary {
  id: string;
  version_number: number;
  status: 'active' | 'superseded' | 'conflict_review';
  is_active: boolean;
  trigger_type: string;
  created_at: string;
  created_by: string | null;
}

export interface ConflictingItem {
  id: string;
  item_text: string;
  category: string;
  quality_status: QualityStatus;
  evidence: EvidenceItem[];
}

export interface MemoryConflict {
  id: string;
  meeting_id: string;
  memory_version_id: string | null;
  conflicting_items: {
    current: ConflictingItem[];
    prior: ConflictingItem[];
  };
  quality_status: QualityStatus;
  review_status: 'pending' | 'reviewed';
  resolution_note: string | null;
  resolved_at: string | null;
  resolved_by: string | null;
  created_at: string;
}

export interface PriorMemoryInput {
  id: string;
  prior_memory_version_id: string;
  match_confidence: 'safe_match' | 'uncertain' | null;
  included_by_user: boolean;
  excluded_by_user: boolean;
  prior_meeting_title?: string;
  prior_meeting_completed_at?: string;
}

// ── API helpers ───────────────────────────────────────────────────────────────

async function memoryFetch<T>(
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

export async function getActiveMemory(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: ActiveMemory } | { ok: false; status: number; message: string }> {
  return memoryFetch<ActiveMemory>(`/api/meetings/${meetingId}/memory`, fetch_fn);
}

export async function getMemoryState(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: MemoryStateResponse } | { ok: false; status: number; message: string }> {
  return memoryFetch<MemoryStateResponse>(`/api/meetings/${meetingId}/memory/state`, fetch_fn);
}

export async function getMemoryVersions(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: MemoryVersionSummary[] } | { ok: false; status: number; message: string }> {
  return memoryFetch<MemoryVersionSummary[]>(`/api/meetings/${meetingId}/memory/versions`, fetch_fn);
}

export async function getMemoryVersion(
  meetingId: string,
  versionNumber: number,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: MemoryVersion } | { ok: false; status: number; message: string }> {
  return memoryFetch<MemoryVersion>(`/api/meetings/${meetingId}/memory/versions/${versionNumber}`, fetch_fn);
}

export async function reprocessMemory(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true } | { ok: false; status: number; message: string }> {
  const res = await fetch_fn(`/api/meetings/${meetingId}/memory/reprocess`, { method: 'POST' });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    return { ok: false, status: res.status, message: body?.message ?? res.statusText };
  }
  return { ok: true };
}

export async function getMemoryConflicts(
  meetingId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true; data: MemoryConflict[] } | { ok: false; status: number; message: string }> {
  return memoryFetch<MemoryConflict[]>(`/api/meetings/${meetingId}/memory/conflicts`, fetch_fn);
}

export async function resolveConflict(
  meetingId: string,
  conflictId: string,
  resolutionNote: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true } | { ok: false; status: number; message: string }> {
  const res = await fetch_fn(`/api/meetings/${meetingId}/memory/conflicts/${conflictId}/resolve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ resolution_note: resolutionNote }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    return { ok: false, status: res.status, message: body?.message ?? res.statusText };
  }
  return { ok: true };
}

export async function removePriorMemoryMatch(
  meetingId: string,
  priorMemoryVersionId: string,
  fetch_fn: typeof fetch
): Promise<{ ok: true } | { ok: false; status: number; message: string }> {
  const res = await fetch_fn(`/api/meetings/${meetingId}/memory/prior-memories/${priorMemoryVersionId}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    return { ok: false, status: res.status, message: body?.message ?? res.statusText };
  }
  return { ok: true };
}
