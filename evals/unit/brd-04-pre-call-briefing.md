# Unit Eval: brd-04-pre-call-briefing

> 🔴 Failing — implementation pending

## Scope

Unit tests for Pre-Call Briefing (BRD-04): briefing generation logic (signal matching, relatedness scoring, two-signal threshold), preparation status state machine, source selection criteria and exclusions, briefing length enforcement and section capping, quality signal derivation, and character count validation.

Source of truth: `specs/domain/brd-04-pre-call-briefing.md`

---

## Signal Matching Tests

### Two-Signal Threshold

| Scenario | Input | Expected |
|----------|-------|----------|
| Two signals — same participant + similar title | Upcoming M, Prior P1 (shared email + Levenshtein ≤3) | P1 qualifies |
| Two signals — similar title + close chronology | Upcoming M, Prior P2 (Levenshtein ≤3 + completed within 90d) | P2 qualifies |
| Two signals — same org/client + close chronology | Upcoming M, Prior P3 (same org + within 90d) | P3 qualifies |
| Two signals — same participant + same org | Upcoming M, Prior P4 (shared email + same org) | P4 qualifies |
| One signal — same participant only | Upcoming M, Prior P5 (shared email only) | P5 does NOT qualify |
| One signal — similar title only | Upcoming M, Prior P6 (Levenshtein ≤3 only) | P6 does NOT qualify |
| One signal — same org only | Upcoming M, Prior P7 (same org only) | P7 does NOT qualify |
| One signal — close chronology only | Upcoming M, Prior P8 (within 90d only) | P8 does NOT qualify |
| Zero signals | Upcoming M, Prior P9 (no matching signals) | P9 does NOT qualify |
| Future-dated source meeting | Prior P10 completes after upcoming M's scheduled start | P10 does NOT qualify regardless of signal count |
| Email unavailable, display name only | Prior P11 shares only display name (no email match) | P11 does NOT qualify via participant signal |

---

### Signal Types

#### Same Participant Signal

| Scenario | Input | Expected |
|----------|-------|----------|
| Exact email match | Upcoming participant: alice@example.com; Prior participant: alice@example.com | Signal passes |
| Case-insensitive email match | Upcoming: ALICE@EXAMPLE.COM; Prior: alice@example.com | Signal passes |
| Partial/typo email match | Upcoming: alice@example.com; Prior: alice@exampel.com (typo) | Signal does NOT pass |
| Display name match only | Upcoming: "Alice"; Prior: "Alice" (no email) | Signal does NOT pass |
| No shared participants | No email address in common | Signal does NOT pass |

#### Similar Title Signal

| Scenario | Input | Expected |
|----------|-------|----------|
| Levenshtein distance ≤3 | "Q3 Planning Review" vs "Q3 Planning" | Signal passes |
| Levenshtein distance =4 | "Q3 Planning Review" vs "Q3 Planing" (typo) | Signal does NOT pass |
| Common token sequence ≥4 consecutive words | "Annual Budget Review FY2026" vs "Annual Budget Review" | Signal passes (4-word match) |
| Common token sequence =3 consecutive words | "Annual Budget Review" vs "Annual Budget" | Signal does NOT pass (3-word match only) |
| Normalized title used | Both titles lowercased and trimmed before comparison | Normalization applied |
| BRD-03 constrained title rule | Both titles follow BRD-03 normalization | Rule applied consistently |

#### Same Organization/Client Signal

| Scenario | Input | Expected |
|----------|-------|----------|
| Exact org match after normalization | Org: "Acme Corp" vs "acme corp" | Signal passes |
| Upcoming metadata explicit org matches prior org | Upcoming: org="Acme"; Prior participant org="Acme" | Signal passes |
| Mismatched orgs | Org: "Acme" vs "Beta LLC" | Signal does NOT pass |
| No org data | Neither meeting has org data | Signal does NOT pass |

#### Close Chronology Signal

| Scenario | Input | Expected |
|----------|-------|----------|
| Prior completed within 90 days | Prior completed 89 days before upcoming scheduled start | Signal passes |
| Prior completed at exactly 90 days | Prior completed exactly 90 days before | Signal passes |
| Prior completed at 91 days | Prior completed 91 days before | Signal does NOT pass |
| Prior completed in future | Prior scheduled start is after upcoming's scheduled start | Signal does NOT pass |
| Prior has no completed_at | Prior has no completed_at timestamp | Signal does NOT pass |

---

## Relatedness Scoring Tests

| Scenario | Setup | Expected |
|----------|-------|----------|
| More signals = higher rank | P1: 2 signals; P2: 3 signals; P3: 2 signals | P2 ranked first |
| Tie: more recent first | P1 (2 signals, older); P2 (2 signals, newer) | P2 ranked above P1 |
| Tie: stronger title similarity | P1 (2 signals, LevD=3); P2 (2 signals, LevD=1) | P2 ranked above P1 |
| Tie: stable UUID as final tie-breaker | P1 and P2 identical signal count, recency, title similarity | Deterministic stable ordering by UUID |
| Top 3 limit enforced | 5 qualifying meetings (P1–P5, all ≥2 signals) | Only top 3 by ranking used |
| Fewer than 3 qualify | 2 qualifying meetings (P1, P2) | Both P1 and P2 used |
| Zero qualify | No meetings meet ≥2 signal threshold | No source selection; no-prior-memory shell triggered |
| Only top 3 qualify | 4 meetings meet ≥2 signals | Top 3 selected; 4th excluded |

---

## Preparation Status State Machine Tests

### Valid State Transitions

| Scenario | Current State | Event | Expected Next State |
|----------|--------------|-------|---------------------|
| Start generation | `pending` | Job queued on upcoming meeting creation | `generating` |
| Generation success | `generating` | All sources processed, no weak evidence | `ready` |
| Generation success with weak evidence | `generating` | Processed with weak evidence items | `ready_with_caveats` |
| No prior memory | `generating` | No qualifying sources found | `no_prior_memory` |
| Generation failure | `generating` | Provider error, timeout, or validation error | `failed` |
| User requests regenerate | `ready` | Manual regenerate requested | `regenerating` |
| User requests regenerate | `ready_with_caveats` | Manual regenerate requested | `regenerating` |
| Source memory changes | `ready` | Source reprocessed or edited | `stale` |
| Source memory changes | `ready_with_caveats` | Source reprocessed or edited | `stale` |
| Regeneration success | `regenerating` | New version generated successfully | `ready` (new version active) |
| Regeneration failure | `regenerating` | New attempt fails | `failed` (previous version remains available) |
| Retry from failed | `failed` | User clicks retry | `generating` |
| Undo exclusion before regenerate | `ready` | User restores excluded source | Same state, source reconsidered on next regenerate |

### Invalid State Transitions

| Scenario | Current State | Event | Expected Behavior |
|----------|--------------|-------|------------------|
| Regenerate while already regenerating | `regenerating` | User clicks regenerate again | Rejected or ignored (no duplicate job queued) |
| State shown while no briefing exists | No briefing yet | State queried | `pending` returned |

### Status Display Labels

| Scenario | Status | Expected Label |
|----------|--------|---------------|
| Generating in progress | `generating` | "Preparing briefing..." or similar |
| Ready with no caveats | `ready` | "Briefing ready" |
| Ready with weak evidence | `ready_with_caveats` | "Briefing ready with caveats" |
| No prior memory found | `no_prior_memory` | "No prior memory found" |
| Stale briefing | `stale` | "Briefing may be outdated" |
| Failed generation | `failed` | "Briefing could not be prepared" |
| User requested regenerate | `regenerating` | "Updating briefing..." |

---

## Source Selection Criteria Tests

### Authorization Boundary

| Scenario | Input | Expected |
|----------|-------|----------|
| User authorized for upcoming but not source | Source S2 ACL restricted from User A | S2 NOT selected even if ≥2 signals match |
| User authorized for both | S1 and S2 both pass ≥2 signals, user authorized for both | Both selected |
| Source without meeting ACL entry | Source S3 has no explicit ACL entry | Treated as unauthorized; NOT selected |
| Unauthorized source never appears in logs | S2 excluded due to auth | S2 not present in any log event fields |
| Unauthorized source not in version history | S2 excluded due to auth | S2 not listed in briefing sources list |

### Exclusion Persistence

| Scenario | Input | Expected |
|----------|-------|----------|
| Exclude source persists per upcoming meeting | User excludes S1 for upcoming M | Exclusion record created with `upcoming_meeting_id=M`, `source_meeting_id=S1` |
| Exclusion does not affect other upcoming meetings | S1 excluded for M1 | S1 still eligible for M2 (different upcoming meeting) |
| Exclusion not a global rule | S1 excluded for M1 | No global flag set on S1 itself |
| Exclusion reconsidered on restore | User restores S1 before regenerate | Exclusion record deleted; S1 re-evaluated for qualification |

### Source Selection with Exclusions

| Scenario | Input | Expected |
|----------|-------|----------|
| Excluded source not re-selected | S1 excluded, S1 would otherwise qualify | S1 NOT selected on regenerate |
| Excluded source visible in excluded area | S1 excluded | S1 shown in collapsed excluded-sources section with original relatedness reasons |
| Restore makes source eligible again | S1 restored, S1 still meets ≥2 signals | S1 re-selected on next regenerate |

---

## Briefing Length Enforcement Tests

### Concise Summary Length

| Scenario | Input | Expected |
|----------|-------|----------|
| Concise summary has maximum line limit | Summary generated | Concise summary ≤ ~300 words (readable in <60s) |
| Summary fits on one screen | Summary rendered | No scrolling required for concise summary |

### Section Cap

| Scenario | Input | Expected |
|----------|-------|----------|
| Top prior context capped | More than 3 prior context items | Top 3 shown with "plus more" indicator |
| Open actions capped | More than 5 action items | Top 5 shown |
| Risks/questions capped | More than 5 risk/question items | Top 5 shown |

### Expandable Details Section Limits

| Scenario | Input | Expected |
|----------|-------|----------|
| Detailed sections do not overwhelm concise summary | All sections expanded | Progressive disclosure: summary first, details collapsed by default |
| Source meeting list limited | More than 3 qualifying sources | Top 3 shown; rest not listed |

---

## Quality Signal Derivation Tests

### briefing_confidence

| Scenario | Input | Expected |
|----------|-------|----------|
| All sources strong evidence | All source memories have `strong_evidence` | `briefing_confidence = 'high'` |
| Some sources weak evidence | At least one source has `weak_evidence` | `briefing_confidence = 'medium'` |
| Any source has insufficient evidence | At least one source has `insufficient_evidence` | `briefing_confidence = 'low'` |
| No qualifying prior memory | Result is `no_prior_memory` | `briefing_confidence` not applicable or derived differently |

### evidence_confidence

| Scenario | Input | Expected |
|----------|-------|----------|
| Evidence from strong source | Item derived from `strong_evidence` memory item | `evidence_confidence = 'strong'` |
| Evidence from weak source | Item derived from `weak_evidence` memory item | `evidence_confidence = 'weak'` |
| Evidence from insufficient source | Item derived from `insufficient_evidence` memory item | Item NOT presented as fact (excluded from advice) |
| Evidence from conflicting source | Item derived from `conflicting_evidence` memory item | Item NOT presented as fact (excluded from advice) |

### source_quality

| Scenario | Input | Expected |
|----------|-------|----------|
| Source with strong evidence items | Source meeting memory has all items `strong_evidence` | `source_quality = 'strong'` |
| Source with weak evidence items | Source has at least one `weak_evidence` item | `source_quality = 'weak'` |
| Source with insufficient evidence items | Source has `insufficient_evidence` category | `source_quality = 'insufficient'` |

### Conflict Exclusion in Quality Signals

| Scenario | Input | Expected |
|----------|-------|----------|
| Conflicting memory item excluded from advice | `conflicting_evidence` item in source memory | Item excluded from `top_prior_context`, `decisions`, `actions`, `risks` |
| Conflicting stakeholder notes excluded | Unresolved conflict on stakeholder note | Note not included in `stakeholder_notes` section |
| Source with only conflicting evidence | Source memory is entirely `conflicting_evidence` | Source may still be listed as qualifying source, but no advice items derived from it |

---

## Character Count Validation Tests

### Concise Summary Sections

| Scenario | Threshold | Expected |
|----------|-----------|----------|
| objective length warn | > 200 chars | Warning logged; section still displayed |
| objective length block | > 500 chars | Section content truncated or rejected at generation input |
| recommended_focus length warn | > 200 chars | Warning logged |
| recommended_focus length block | > 500 chars | Section content truncated or rejected |
| top_prior_context length warn | > 300 chars | Warning logged |
| top_prior_context length block | > 800 chars | Truncated at block threshold |

### Source Annotations

| Scenario | Threshold | Expected |
|----------|-----------|----------|
| Source relatedness reason text | Each reason string | Reason text human-readable, not exceeding ~100 chars |
| Source list item text | Each source name + date line | Total per-item text ≤ ~150 chars |

### Logging Character Limits

| Scenario | Expected |
|----------|----------|
| Metric label values | Low-cardinality enum only; no free-text content |
| Log event string fields | Hashed or opaque identifiers only; no raw content strings |
| Log event forbidden fields | Raw meeting titles, transcript text, notes text, briefing text, source snippets, participant PII, stakeholder note content |

---

## Briefing Versioning Tests (Unit-Level)

| Scenario | Input | Expected |
|----------|-------|----------|
| Distinct version created on each generation | Generation runs N times | N distinct version_number values (1, 2, ..., N) |
| Latest version marked active | Generation completes | Latest version has `is_active = true`, prior version has `is_active = false` |
| Version immutability | Two versions V1, V2 exist | V1 `content` JSONB unchanged after V2 is created |
| Version trigger type recorded | Manual regenerate vs auto | `trigger_type` in {'auto', 'manual_regenerate'} |
| Version result recorded | Result: ready, ready_with_caveats, no_prior_memory | `result` field set correctly per version |
| Stale flag set on affected version | Source memory reprocessed | Affected version `is_stale = true`, `stale_at` timestamp set |

---

## No-Prior-Memory Shell Tests (Unit-Level)

| Scenario | Input | Expected |
|----------|-------|----------|
| Shell generated when zero qualify | No prior meetings meet ≥2 signals | `result = 'no_prior_memory'`, shell content generated |
| Shell contains objective from upcoming | Upcoming meeting has title/description | `shell.objective` derived from upcoming metadata |
| Shell contains recommended_focus | Upcoming has description/goals | `shell.recommended_focus` derived from upcoming metadata |
| Shell contains suggested_prep_questions | Upcoming has no prior memory | `shell.suggested_prep_questions` populated |
| Shell does NOT imply prior continuity | Shell generated | No language implying "previously we decided..." |
| Shell does NOT contain prior decisions | Shell generated | `decisions`, `prior_decisions` sections absent or clearly marked as N/A |
| Shell does NOT contain prior actions | Shell generated | `open_actions` shows insufficient-context state, not prior items |
| Shell does NOT contain stakeholder memory | Shell generated | `stakeholder_notes` absent or clearly marked N/A |
| Shell does NOT contain historical risks | Shell generated | `risks_questions` shows insufficient-context state, not historical risks |

---

## Running

```bash
# Unit tests (Go)
go test ./internal/precall/... -v

# Gate: check-status-sync.sh must pass before implementation tasks are created
bash scripts/check-status-sync.sh
```