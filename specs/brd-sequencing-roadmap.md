# ContextPilot BRD Sequencing Roadmap

> Purpose: Working roadmap for turning the master ContextPilot business requirement into implementation-ready BRDs. This is not an implementation plan. Each item should still go through BRD drafting, review, approval, curation, and implementation handoff.

---

## Source Vision

The master product vision is the meeting intelligence system described in `contextPilot.md`:

**Core promise:** Never walk into a meeting cold again.

The product should help users capture prior meeting context, turn it into durable business memory, and generate useful preparation before future related meetings.

---

## Recommended BRD Order

### 1. BRD-02 Manual Meeting Import

**Purpose:** Let users manually create completed/past meetings, add structured participants, and paste transcript or notes.

**Status:** Draft exists at `specs/domain/brd-02-manual-meeting-import.md`.

**Discussion needed:** Resolved for draft review.

**Resolved product decisions:**

- Manual import is limited to completed/past meetings; future meeting creation belongs to BRD-04 Pre-Call Briefing.
- Transcript plus notes have a combined Phase 1 limit of 50,000 characters.
- Participants are stored as structured meeting participant records.
- Participant display name is required.
- Participant email, organization/company, and role/title are optional and appear behind an advanced affordance.
- Successful import redirects directly to the saved meeting detail page.
- Global contacts, participant deduplication, identity resolution, CRM-style relationship history, AI processing, and pre-call briefing are out of scope for BRD-02.

---

### 2. BRD-03 Meeting Memory Processing

**Purpose:** Process a meeting into reusable memory: summary, decisions, action items, risks, blockers, open questions, stakeholder notes, and next recommended focus.

**Status:** Accepted draft exists at `specs/domain/brd-03-meeting-memory-processing.md`; ready for curation.

**Discussion needed:** Resolved for draft review.

**Why next:** This is the durable memory foundation that later briefings depend on.

**Resolved product decisions:**

- MVP uses an evidence-first full memory foundation.
- Every extracted item requires strict source evidence.
- Evidence includes snippet plus stable source location.
- MVP memory categories are summary, decisions, action items, risks/blockers, open questions, stakeholder notes, and next recommended focus.
- Weak categories still appear in a complete structure and are marked insufficient evidence rather than omitted or guessed.
- Action items are structured with description, owner if explicit, due date if explicit, status, evidence, and quality status.
- Decisions are change-aware, but confirming/changing/reversing prior decisions requires traceable current evidence plus eligible prior memory.
- Stakeholder notes are relationship-oriented but limited to meeting-relevant, source-supported business context.
- Processing is automatically queued after import and supports manual retry/reprocess.
- Reprocessing creates versioned memory; failed runs do not replace the active successful version.
- Source meeting edits automatically queue reprocessing and mark existing memory stale.
- Processing states include not processed, queued, processing, completed, completed with insufficient evidence, failed, stale, reprocessing, retrying, and retry exhausted.
- Prior memories may be selected through safe automatic matching using constrained signals such as same participants, similar title, and close chronology.
- Users can see, exclude, include, and reprocess prior-memory inputs.
- Item and category quality statuses are strong evidence, weak evidence, insufficient evidence, and conflicting evidence.
- Conflicting evidence requires user review, appears in a separate review queue, and is excluded from downstream briefing input until resolved.
- Conflict review uses evidence-backed resolution notes.
- Failure behavior includes up to 3 retries with exponential backoff, retry exhausted state, user-safe failure reasons, operator diagnostics, and manual retry.
- Processing latency is architecture-defined; processing must be asynchronous, observable, and non-blocking.
- Retention for memory versions and evidence follows the source meeting for now, with final policy deferred to BRD-06.
- Feature flag shape is architecture-defined, with minimum server flag `FF_ENABLE_MEETING_MEMORY_PROCESSING` and browser flag if UI/actions ship.
- Observability covers standard job lifecycle events and metrics; quality distribution metrics are deferred to curation.
- The memory view emphasizes briefing-readiness.
- Briefing-ready requires summary, decisions, action items, risks/blockers, and open questions to have strong or weak evidence and no unresolved conflict blocking core categories.
- Strict unsupported-claim evals are required.
- The BRD is provider-agnostic.

---

### 3. BRD-04 Pre-Call Briefing

**Purpose:** Generate the flagship preparation briefing for an upcoming meeting using prior related meeting memory.

**Status:** Approved draft exists at `specs/domain/brd-04-pre-call-briefing.md`; ready for curation.

**Discussion needed:** Resolved for curation.

**Why next:** This proves the main value proposition: users can prepare without manually reviewing old notes.

**Expected briefing contents:**

- Meeting title and recommended objective.
- Previous relevant context.
- Important prior decisions.
- Open action items and commitments.
- Risks, blockers, and unresolved questions.
- Stakeholder notes in expandable details only.
- Suggested questions to ask.
- Suggested agenda.
- Source details and relatedness reasons behind expand/details controls.

**Resolved product decisions:**

- MVP includes automatic balanced related-meeting matching.
- A prior meeting qualifies when at least two signals match: same participant, similar title, same organization/client, and close chronology.
- MVP uses up to the top 3 qualifying related prior meetings.
- Briefing format is progressive: concise summary first, expandable details below.
- Concise summary always includes meeting objective, preparation status, recommended focus, top prior context, open actions, and risks/questions.
- Source annotations and relatedness reasons are available behind expand/details, not inline in every summary item.
- Weak-evidence items may appear with caveats; unresolved conflicting memory is excluded from briefing advice.
- No related prior memory produces a clearly marked preparation shell from upcoming meeting metadata only.
- Generation is async, automatic on upcoming meeting creation, manually regeneratable, versioned, and p95 under 60 seconds for up to 3 prior meetings.
- Users can exclude automatically selected source meetings at the upcoming-meeting level and undo exclusions.
- Stakeholder notes are allowed only in expandable details and must be professional, evidence-backed, caveated when weak, and non-conflicting.

---

### 4. New BRD: Related Meeting Detection

**Likely filename:** `specs/domain/brd-08-related-meeting-detection.md`

**Purpose:** Decide which past meetings are relevant to an upcoming meeting.

**Discussion needed:** High.

**Why separate:** Relatedness quality directly controls briefing quality. It may be too important to hide inside BRD-04.

**Possible relatedness signals:**

- Same attendees.
- Same client/company.
- Same project/topic.
- Similar meeting title.
- Recurring meeting pattern.
- Open action items or risks linked to the meeting.

**Recommended MVP stance:** Start simple and explainable; avoid opaque or over-broad matching until user trust is established.

---

### 5. New BRD: Continuity / What Changed Since Last Time

**Likely filename:** `specs/domain/brd-09-meeting-continuity-changes.md`

**Purpose:** Show what changed since the previous related meeting.

**Discussion needed:** Medium to high.

**Examples:**

- Action item completed.
- Risk remains open.
- Decision changed.
- New blocker appeared.
- New attendee joined.
- Open question was resolved or carried forward.

**Why later:** This depends on meeting memory processing and related meeting detection.

---

### 6. BRD-06 Privacy & Retention Controls

**Purpose:** Define user/org data boundaries, retention, redaction, deletion, and sensitive meeting data handling.

**Discussion needed:** High.

**Recommended timing:** Move earlier if real customer or enterprise data will be used soon.

**Why important:** Meeting transcripts, stakeholder notes, participant data, and commitments are sensitive. Trust rules should be explicit before advanced memory features are broadly used.

---

### 7. BRD-07 Manual Memory Correction

**Purpose:** Let users correct extracted summaries, decisions, action items, risks, open questions, stakeholder notes, and other memory artifacts.

**Discussion needed:** Medium to high.

**Why later:** Correction UX is more valuable after extraction exists, but may become important quickly if users need to trust generated memory.

**Key question:** Is correction required for MVP trust, or can MVP rely on source-grounded extraction and clear caveats?

---

### 8. BRD-05 Provider Connectors

**Purpose:** Integrate with external platforms such as Teams, Google Meet, Zoom, calendar, email, documents, CRM, and project management tools.

**Discussion needed:** Later.

**Why later:** The first value proof should be manual. Automated collection can come after the core memory and briefing loop is validated.

---

## Recommended Release Approach

### Recommended: Option A with a disciplined scope

1. Finish and approve BRD-02 Manual Meeting Import.
2. Write BRD-03 Meeting Memory Processing as the durable memory foundation.
3. Write BRD-04 Pre-Call Briefing as the flagship user value.
4. Split Related Meeting Detection into its own BRD if BRD-04 becomes too large.
5. Pull Privacy & Retention earlier if the product will handle real customer or enterprise data soon.

**Why:** This is the fastest path to proving the product promise while still avoiding a demo-only shortcut.

---

## Alternative Approaches Considered

### Option A: Fastest proof of value

**Scope:**

- BRD-02 Manual Meeting Import.
- BRD-03 Meeting Memory Processing.
- BRD-04 Pre-Call Briefing.
- Manual flow only.
- No platform integrations.
- No advanced correction UX initially.

**Trade-off:** Fastest route to proving “Never walk into a meeting cold again,” but memory quality may be imperfect without correction workflows.

---

### Option B: Trust-first MVP

**Scope:**

- BRD-02 Manual Meeting Import.
- BRD-03 Meeting Memory Processing.
- BRD-06 Privacy & Retention Controls.
- BRD-07 Manual Memory Correction.
- Then BRD-04 Pre-Call Briefing.

**Trade-off:** Better data trust and user confidence, but delays the flagship pre-call briefing experience.

---

### Option C: Briefing-first vertical slice

**Scope:**

- BRD-02 Manual Meeting Import.
- Minimal BRD-03 extraction only enough to support briefing.
- BRD-04 Pre-Call Briefing.
- Related meeting detection kept simple: same attendees/title/project.

**Trade-off:** Strong demo path, but risks under-specifying the durable memory model and causing rework.

---

## Requirements That Are Clean Enough With Light Discussion

These areas are relatively clear from the master business requirement and can usually proceed with limited discussion once placed into the right BRD:

- Manual meeting input.
- Meeting summary.
- Decision tracking.
- Action item tracking.
- Risk/blocker tracking.
- Open question tracking.
- Follow-up summary.
- Dashboard at a high level.
- Meeting detail view at a high level.
- Future integrations as later scope.

---

## Requirements That Need Deeper Discussion

These areas should be clarified carefully before BRD approval:

- How strict the system must be about not inventing facts.
- What counts as a decision versus discussion.
- What confidence/source evidence each extracted memory item needs.
- Whether stakeholder memory is allowed in MVP.
- How related meetings are identified.
- Whether “what changed since last time” uses only internal meeting memory or external sources too.
- How much user correction is required before users trust the system.
- Privacy, retention, and access boundaries.

---

## Suggested Next Discussion Question

For the first usable release, what should be the primary proof of value?

1. A user can paste one past meeting and get structured memory from it.
2. A user can paste a past meeting, create a future meeting, and get a pre-call briefing.
3. A user can manage a small set of recurring meetings with memory, open items, and briefings.
4. Enterprise-safe meeting memory with privacy/retention controls from the start.
