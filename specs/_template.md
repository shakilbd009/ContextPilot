# BRD Template

> Use this template for every BRD. Copy and rename. Fill all sections.

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | `brd-XX` |
| Title | ... |
| Author | ... |
| Created | YYYY-MM-DD |
| Status | Draft / In Review / Accepted |
| Priority | P0 / P1 / P2 |
| Phase | Phase 0 / Phase 1 / Phase 2 |

---

## Overview

What does this feature do? Why does it matter? Keep it to 2-3 sentences.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-1 | ... | ... | ... |

---

## Functional Requirements

### Must Have

- ...

### Should Have

- ...

### Could Have

- ...

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Latency | < Xms for briefing generation |
| Availability | 99.9% uptime |
| Data retention | X days, then PII removal |

---

## Observability

### Metrics to emit

- `cp_<feature>_<action>_total` — counter
- `cp_<feature>_<action>_duration_ms` — histogram

### Log events

- `meeting.import.started`
- `meeting.import.completed`
- `meeting.import.failed`

### Health/readiness endpoints

- `GET /ready` — returns 200 when feature is operational
- `GET /live` — returns 200 when process is alive

---

## Feature Flag

- **Flag name:** `ff_enable_<feature>`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_<FEATURE>`
- **Browser env:** `VITE_FF_ENABLE_<FEATURE>`

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | ... | ... |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| ... | ... | ... | ... |

---

## Relations

- **Parent feature:** ...
- **Blocked by:** ...
- **Blocks:** ...

---

## Open Questions

- ...