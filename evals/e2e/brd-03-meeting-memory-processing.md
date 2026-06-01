# E2E Eval: brd-03-meeting-memory-processing

> 🔴 Failing — implementation pending

## Scope

End-to-end scenarios for Meeting Memory Processing (BRD-03): processing queue, briefing readiness, prior-memory visibility, conflict review, reprocessing, and flag-gated access control.

---

## Scenario: AC-01 — Route hidden when flag is disabled

### Given
- The app is running with `FF_ENABLE_MEETING_MEMORY_PROCESSING=false`
- User is authenticated
- User has a meeting in the system

### When
User inspects the meeting detail page for memory-related entry points

### Then
- No "Memory" tab or `/meetings/{id}/memory` link is visible in the navigation
- No "Conflicts" or "Review Queue" entry point is visible

---

## Scenario: AC-01 — Direct access blocked when flag is disabled

### Given
- The app is running with `FF_ENABLE_MEETING_MEMORY_PROCESSING=false`
- User is authenticated
- User has a meeting with a known UUID

### When
User navigates directly to `/meetings/{id}/memory`

### Then
- Server returns 403 Forbidden or redirects to an error page
- No memory content is rendered

---

## Scenario: AC-02 — Route accessible when flags are enabled

### Given
- The app is running with `FF_ENABLE_MEETING_MEMORY_PROCESSING=true`
- `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=true`
- User is authenticated
- User has a meeting with `memory.state = completed`

### When
User navigates to `/meetings/{id}/memory`

### Then
- HTTP 200 is returned
- Memory content is rendered with all 7 categories visible
- Briefing-readiness signal is displayed

---

## Scenario: AC-02b — Conflicts route accessible when flags are enabled

### Given
- The app is running with both flags enabled
- User has a meeting with `memory.state = completed` and at least one pending conflict

### When
User navigates to `/meetings/{id}/memory/conflicts`

### Then
- HTTP 200 is returned
- Conflict review queue is displayed with conflicting evidence snippets and source attribution

---

## Scenario: AC-03 — Import does not block on processing

### Given
- The app is running with `FF_ENABLE_MEETING_MEMORY_PROCESSING=true`
- `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING=true`
- User is authenticated and on `/meetings/new`
- Valid meeting data is filled (title, completedAt, participant, transcript)

### When
User clicks Save and is redirected to `/meetings/{id}`

### Then
- Redirect completes without waiting for processing
- Meeting detail page shows `memory.state = queued` immediately after import
- User can navigate away and return; state remains accessible

---

## Scenario: AC-11 — Memory versions are listed

### Given
- The app is running with both flags enabled
- User has a meeting with 2+ memory versions (original + 1 reprocess)

### When
User navigates to `/meetings/{id}/memory/versions`

### Then
- HTTP 200 is returned
- Version list shows version number, created_at, trigger_type, status
- Active version is marked as such

---

## Scenario: AC-14 — User-visible lifecycle states

### Given
- The app is running with both flags enabled
- User has meetings in various memory states: `queued`, `processing`, `completed`, `completed_with_insufficient_evidence`, `failed`, `stale`, `retrying`, `retry_exhausted`

### When
User navigates to each meeting's memory view

### Then
- Each meeting shows the correct state indicator
- State labels match: `not_processed`, `queued`, `processing`, `completed`, `completed_with_insufficient_evidence`, `failed`, `stale`, `reprocessing`, `retrying`, `retry_exhausted`

---

## Scenario: AC-15 — Prior memories used are visible without raw content

### Given
- The app is running with both flags enabled
- User has Meeting A that matched Meeting B as a prior memory during processing

### When
User views the memory for Meeting A

### Then
- Prior memory input is visible (meeting title, date, match confidence: `safe_match` or `uncertain`)
- Raw transcript/notes content of the prior meeting is NOT exposed in UI or visible in logs
- User can exclude the prior memory and trigger reprocessing

---

## Scenario: AC-16 — User can exclude prior memory and reprocess

### Given
- The app is running with both flags enabled
- User has a meeting with a `safe_match` prior memory that is incorrect

### When
User excludes the prior memory and clicks Reprocess

### Then
- Reprocess is queued as `trigger_type = manual_retry`
- New memory version is created after reprocess completes
- Excluded prior memory does not influence the new version

---

## Scenario: AC-18 — Conflicting evidence appears in review queue

### Given
- The app is running with both flags enabled
- User has a meeting with a conflict in `decisions` category
- Conflict has `review_status = pending`

### When
User navigates to `/meetings/{id}/memory/conflicts`

### Then
- Conflicting item from current version and prior version are both shown
- Evidence snippets are attributed to source meeting
- Category is identified (`decisions`)
- Resolution note field is present
- "Mark Reviewed" button is available

---

## Scenario: AC-19 — Conflict resolution creates new version

### Given
- The app is running with both flags enabled
- User has a meeting with a pending conflict
- User fills in a resolution note

### When
User clicks "Mark Reviewed" on a conflict

### Then
- HTTP 200 response from `POST /meetings/{id}/memory/conflicts/{conflictId}/resolve`
- New memory version is created with `trigger_type = conflict_resolution`
- Original conflicting sources and resolution note are preserved
- Conflict is removed from pending queue
- Event `meeting_memory.conflict.reviewed` is emitted

---

## Scenario: AC-20 — Briefing readiness signal

### Given
- The app is running with both flags enabled
- User has two meetings: one with all 5 core categories at `strong_evidence` or `weak_evidence` and no pending conflicts; another with a pending conflict blocking core categories

### When
User views each meeting's memory state endpoint

### Then
- First meeting shows `briefing_readiness: ready` or `ready_with_weak`
- Second meeting shows `briefing_readiness: needs_review`
- Stakeholder notes or next recommended focus being `insufficient` does NOT block core briefing readiness

---

## Scenario: AC-22 — Worker unavailability preserves import and active memory

### Given
- The app is running with both flags enabled
- Processing worker is unavailable
- User has a meeting with `memory.state = completed` (active memory exists)

### When
User navigates to `/meetings/{id}/memory`

### Then
- Active memory version is still accessible
- New import does not block on worker availability
- Queued jobs remain queued; no false failure state shown

---

## Scenario: AC-23 — No sensitive content in logs/metrics

### Given
- The app is running with both flags enabled
- A metrics/log collection endpoint is available
- User has a meeting with processed memory

### When
Memory processing completes and metrics/logs are emitted

### Then
- Metric labels do NOT contain transcript text, notes text, evidence snippets, participant PII, memory content, or resolution text
- Log event fields do NOT contain raw meeting content, evidence snippets, or PII
- Source locations (char offsets) do NOT appear in logs or metrics

---

## Running

```bash
# Requires services up
docker compose up -d
pnpm exec playwright test --reporter=list
```