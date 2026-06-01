# E2E Eval: brd-04-pre-call-briefing

> 🔴 Failing — implementation pending

## Scope

End-to-end scenarios for Pre-Call Briefing (BRD-04): async generation, related-meeting matching, source exclusions/restores, versioned briefing history, stale behavior, evidence policy, stakeholder notes handling, failure/retry, and authorization boundary.

Source of truth: `specs/curated/brd-04-pre-call-briefing/brd.md` AC table (lines 299–322).

---

## Scenario: AC-01 — Briefing entry points hidden when both flags are false

### Given
- The app is running with `FF_ENABLE_PRE_CALL_BRIEFING=false`
- `VITE_FF_ENABLE_PRE_CALL_BRIEFING=false`
- User is authenticated
- User has an upcoming meeting in the system

### When
User navigates to the upcoming meeting detail page

### Then
- No briefing entry point is visible in the upcoming meeting detail page
- No "Briefing" tab, link, or CTA is rendered
- Backend returns a safe feature-disabled response for any briefing API request (e.g., GET `/upcoming/{id}/briefing`)

---

## Scenario: AC-01 — Backend rejects briefing requests when flag is disabled

### Given
- The app is running with `FF_ENABLE_PRE_CALL_BRIEFING=false`
- `VITE_FF_ENABLE_PRE_CALL_BRIEFING=false` (browser hidden)
- User is authenticated with an upcoming meeting UUID

### When
User POSTs to `/upcoming/{id}/briefing/regenerate` directly

### Then
- Server returns 403 or a structured feature-disabled JSON response (not a raw error page)
- No briefing data is generated or persisted
- Operator diagnostics do not expose raw meeting content or participant PII

---

## Scenario: AC-02 — Upcoming meeting creation queues async briefing job without blocking

### Given
- The app is running with `FF_ENABLE_PRE_CALL_BRIEFING=true`
- `VITE_FF_ENABLE_PRE_CALL_BRIEFING=true`
- User is authenticated and on the upcoming meeting creation page
- Valid upcoming meeting data is filled (title, scheduledAt, participants, optional description)

### When
User clicks Save and is redirected to the upcoming meeting detail page

### Then
- Redirect completes without waiting for briefing generation
- Upcoming meeting detail page is accessible immediately
- Briefing entry point shows `generating` status
- User can navigate away and return; page remains responsive
- Async job is visible in the processing queue

---

## Scenario: AC-03 — Manual regenerate queues new job while latest completed briefing remains visible

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with an existing completed briefing (status: `ready` or `ready_with_caveats`)
- User is on the upcoming meeting detail page viewing the current briefing

### When
User clicks the Regenerate or "Refresh Briefing" action

### Then
- Regeneration is queued as an async job
- Current completed briefing remains visible and readable
- Briefing status transitions to `regenerating`; latest completed version is NOT removed
- User can continue navigating and is not blocked
- After regeneration completes, new version becomes active; old version remains in history

---

## Scenario: AC-04 — Related-meeting matching requires at least two signals

### Given
- The app is running with both flags enabled
- A fixture of prior meetings with varying signal combinations:
  - Meeting P1: same participant + similar title (2 signals — should qualify)
  - Meeting P2: same participant only (1 signal — should NOT qualify)
  - Meeting P3: similar title + close chronology (2 signals — should qualify)
  - Meeting P4: same organization/client + close chronology (2 signals — should qualify)
  - Meeting P5: no matching signals (0 signals — should NOT qualify)
- User has an upcoming meeting scheduled

### When
Briefing generation runs for the upcoming meeting

### Then
- Only P1, P3, and P4 are selected as source meetings
- P2 and P5 are NOT selected
- Each selected source meeting has its relatedness reasons recorded and displayable
- Relatedness reasons are human-readable (e.g., "Shared participant: alice@example.com", "Similar title: Q3 Planning")

---

## Scenario: AC-05 — Briefing generation uses at most top 3 qualifying prior meetings

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with 5 qualifying prior meetings (P1–P5 all meet the ≥2 signal threshold)
- Each qualifying meeting has a deterministic rank order

### When
Briefing generation runs

### Then
- No more than 3 source meetings are selected
- The top 3 by ranking order are used (highest signal count → most recent → strongest title similarity → stable UUID tie-breaker)
- If fewer than 3 qualify, all available qualifying meetings are used
- Source count field in the briefing version reflects the actual number used

---

## Scenario: AC-06 — No qualifying prior memory generates a preparation shell without implied continuity

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with NO prior meetings meeting the ≥2 signal threshold
- No excluded sources were restored that would change the qualification outcome

### When
Briefing generation runs for the upcoming meeting

### Then
- A briefing is generated with `result: no_prior_memory`
- Concise summary sections show explicit insufficient-context states (not invented content)
- The briefing does NOT imply: prior continuity, prior decisions, prior actions, stakeholder memory, or historical risks
- `no_prior_memory_shell` block is populated in the briefing content JSON:
  - `generated: true`
  - `objective` derived from upcoming meeting metadata
  - `recommended_focus` derived from upcoming meeting metadata
  - `suggested_prep_questions` derived from upcoming meeting metadata

---

## Scenario: AC-07 — Concise summary includes all required sections with explicit insufficient-context states

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with qualifying prior memory
- The briefing has been generated successfully

### When
User views the concise summary on the upcoming meeting detail page

### Then
- Concise summary displays all 6 required sections:
  - `objective`
  - `preparation_status`
  - `recommended_focus`
  - `top_prior_context`
  - `open_actions`
  - `risks_questions`
- Any section without strong evidence shows an explicit insufficient-context state (e.g., "Insufficient prior memory to determine open actions")
- No section is blank or omits the status indicator

---

## Scenario: AC-08 — Reviewer can understand recommended meeting focus from concise summary in under 60 seconds

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with qualifying prior memory and a completed briefing
- The scenario uses the prepared fixture: upcoming meeting with 2–3 qualifying prior meetings

### When
A reviewer reads the concise summary on the upcoming meeting detail page (timed)

### Then
- The recommended meeting focus is identifiable without expanding detailed sections
- The concise summary is scannable without inline source clutter
- The reviewer can summarize the meeting's purpose and top prep items within 60 seconds using only the concise summary

---

## Scenario: AC-09 — Every briefing item is traceable through expand/details source information

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with qualifying prior memory and a completed briefing
- Briefing contains items derived from prior memory (e.g., prior context, decisions, actions)

### When
User inspects any briefing item in the concise summary or expandable detailed sections

### Then
- Each item has an expand/details affordance
- Expanding the item shows source meeting details (title, date, source meeting ID)
- The `sources` array in the briefing content JSON is populated for each qualifying prior meeting
- Traceability holds for every item in `detailed_sections` (previous relevant context, important prior decisions, open action items, unresolved risks/blockers, open questions)

---

## Scenario: AC-10 — Source details show human-readable relatedness reasons

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with qualifying prior memory and a completed briefing
- At least one source meeting was selected via the ≥2 signal matching rule

### When
User expands the source details panel for a selected source meeting

### Then
- Human-readable relatedness reasons are displayed (e.g., "Same participant: bob@acme.com", "Similar title: Q2 Review", "Same organization: Acme Corp", "Recent: 14 days ago")
- No opaque internal scoring numbers are shown
- Relatedness reasons match the actual matching signals that qualified the meeting

---

## Scenario: AC-11 — Weak-evidence items appear with caveats; insufficient-evidence items are not presented as facts

### Given
- The app is running with both flags enabled
- User has an upcoming meeting where some briefing items have `quality_status: weak_evidence` and others have `quality_status: insufficient_evidence`

### When
User views the briefing (concise summary and expandable sections)

### Then
- Items with `quality_status: weak_evidence` appear with a visible caveat indicator (e.g., "⚠ This context is based on weak evidence")
- Items with `quality_status: insufficient_evidence` are NOT presented as facts in the briefing advice
- The caveat language is visible but does not block readability of the overall briefing

---

## Scenario: AC-12 — Unresolved conflicting memory items are excluded from briefing advice and stakeholder notes

### Given
- The app is running with both flags enabled
- User has a source meeting with an unresolved conflict in any category (e.g., `decisions` category has `review_status: pending` with `quality_status: conflicting_evidence`)
- The conflicting source meeting qualifies for the upcoming meeting's briefing via the ≥2 signal rule

### When
Briefing generation runs

### Then
- The conflicting memory item is NOT included in briefing advice (concise summary or detailed sections)
- The `conflicting_evidence` item does not appear in `detailed_sections`
- `stakeholder_notes` in detailed sections do not include the unresolved conflicting content
- The source meeting may still be listed as a qualifying source for transparency, but its conflicting items are excluded from the briefing output

---

## Scenario: AC-13 — Stakeholder notes appear only in expandable details, never in concise summary

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with qualifying prior memory where at least one source meeting has stakeholder notes

### When
User views the briefing

### Then
- The concise summary does NOT contain stakeholder note content
- Stakeholder notes appear only in the expandable `detailed_sections.stakeholder_notes` section
- Each stakeholder note is professional, meeting-relevant, evidence-backed, and caveated when weak
- Sensitive personal attributes, unsupported personality inference, private/irrelevant details, and unresolved conflicting stakeholder notes are excluded

---

## Scenario: AC-14 — Excluding a source meeting persists for the upcoming meeting across future regenerations

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with a completed briefing using 2–3 qualifying source meetings

### When
User excludes one of the selected source meetings from the briefing page

### Then
- The exclusion is persisted in `briefing_source_exclusions` table with the upcoming meeting ID
- User triggers regeneration (manual or auto after exclusion)
- Regeneration runs and the excluded source meeting is NOT selected again
- A new briefing version is generated using the remaining qualifying meetings
- The exclusion applies ONLY to this upcoming meeting (not global)

---

## Scenario: AC-15 — Users can undo a source exclusion before regeneration, and excluded sources remain visible in a collapsed area

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with a completed briefing where a source was previously excluded

### When
User clicks the undo/restore action on the excluded source

### Then
- The exclusion is removed (record deleted from `briefing_source_exclusions`)
- Excluded source is visible in a collapsed excluded-sources area with its original relatedness reasons
- User can collapse/expand the excluded-sources area
- If user triggers regeneration, the restored source is reconsidered for qualification

---

## Scenario: AC-16 — Each successful generation persists a distinct version; prior versions remain accessible

### Given
- The app is running with both flags enabled
- User has an upcoming meeting that has been briefed multiple times (v1, v2, v3)

### When
User navigates to the briefing version history

### Then
- GET `/upcoming/{id}/briefing/versions` returns a list of all versions with version_number, created_at, trigger_type, status, result
- The active/latest completed version is marked `is_active: true`
- User can navigate to a specific prior version: GET `/upcoming/{id}/briefing/versions/{versionNumber}`
- Each version's `content` JSONB is preserved immutably
- No version overwrites another; `version_number` is monotonically increasing

---

## Scenario: AC-17 — Source memory reprocessing marks affected briefing versions stale and offers regeneration

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with a completed briefing (version 1)
- One of the source meetings used in the briefing is later reprocessed (memory version updated or new conflict detected)

### When
User views the briefing for the upcoming meeting after the source reprocessing

### Then
- The existing briefing version shows a visible stale indicator
- `preparation_status` on the briefing version is `stale`
- Regeneration is offered via a UI affordance (button or prompt)
- Stale briefing remains readable
- Triggering regeneration creates a new version (version 2)

---

## Scenario: AC-18 — Failed generation shows safe failure reason, does not remove latest completed briefing

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with a completed briefing (status: `ready`)
- A subsequent regeneration attempt fails

### When
User views the briefing after the failed regeneration

### Then
- User sees a safe user-facing failure reason (e.g., "Briefing could not be updated at this time. Your previous briefing is still available.")
- Retry/regenerate affordance is offered
- The latest completed briefing (version N) remains visible and readable
- Operator diagnostics in logs do not expose raw meeting content or participant PII

---

## Scenario: AC-19 — Briefing generation completes within P95 < 60s for up to 3 qualifying prior meetings

### Given
- The app is running with both flags enabled
- Performance fixture is set up: upcoming meeting with 3 qualifying prior meetings
- Normal operating conditions (no artificial load, processing worker available)

### When
Briefing generation job runs from queue start to terminal success/failure

### Then
- P95 latency from job start to completion is under 60 seconds for up to 3 qualifying source meetings
- Measurement excludes queue wait time (measured from job start to terminal state)

---

## Scenario: AC-20 — Latest completed briefing view renders within P95 < 500ms

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with a cached completed briefing (no pending regeneration)
- Performance fixture is loaded

### When
User navigates to the upcoming meeting detail page and the briefing view renders

### Then
- Client-visible render time for the cached briefing is under 500ms (P95)
- Measurement excludes network variability outside the app (measured app-visible render)

---

## Scenario: AC-21 — Metrics and logs cover pipeline health and product usefulness events without raw content or PII

### Given
- The app is running with both flags enabled
- User has an upcoming meeting with a completed briefing
- Observability endpoints/logs are accessible

### When
Briefing generation completes and emits metrics/logs

### Then
- Metric labels use controlled enum values: `status` (generating, ready, ready_with_caveats, no_prior_memory, stale, failed), `failure_class` (if applicable), `source_count_bucket` (e.g., 0, 1, 2, 3), `trigger_type` (auto, manual_regenerate)
- High-cardinality content is banned from metric labels: no raw titles, no participant PII, no transcript text, no notes text, no evidence snippets, no briefing text
- Logs do not contain raw meeting content, source snippets, or participant PII
- Product usefulness events are emitted for: viewing a briefing, expanding details/source panels, requesting regeneration, excluding/restoring a source, generating a no-prior-memory shell

---

## Scenario: AC-22 — Authorization prevents unauthorized source meetings from being selected, displayed, logged, or used

### Given
- The app is running with both flags enabled
- User A is authorized for upcoming meeting M and source meeting S1, but NOT for source meeting S2
- S2 has the meeting ACL visibility restricted
- Both S1 and S2 would qualify based on the ≥2 signal matching rule for M

### When
Briefing generation runs for upcoming meeting M as User A

### Then
- S2 is NOT selected as a source meeting
- S2 does NOT appear in the briefing sources list or version history
- S2 is NOT logged in observability or diagnostic output
- S2 is NOT used in briefing content generation
- The briefing uses only S1 (and any other authorized qualifying meetings)
- Authorization is enforced at the meeting ACL level

---

## Running

```bash
# Requires services up
docker compose up -d

# E2E tests (Playwright)
pnpm exec playwright test --reporter=list

# Gate: check-status-sync.sh must pass before implementation tasks are created
bash scripts/check-status-sync.sh
```