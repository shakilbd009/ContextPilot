# Feature Flags

> Registry of all feature flags in ContextPilot. All default to `false`.

Lifecycle stages: `Planned` → `In Dev` → `Active` → `Deprecated` → `Removed`

> **State vocabulary:** **Implemented** = code exists, **Gates Green** = committed code passes `make eval` on the committed ref, **Production-Ready** = `done-auditor` returned Trustworthy / Mostly trustworthy with no P0/P1. The **Lifecycle** column below describes implementation maturity; the **Gates Green on commit** column tells you whether a real ref passes the gates. Disagreement means recovery is incomplete. See [STATUS.md](../STATUS.md) → "Recovery state" for the current honest picture (as of 2026-06-01 no feature is Gates Green on a commit; the recovery work is in an uncommitted working tree on `ops/restore-ci-baseline`).

---

## UI Layer

| Flag | Env (server) | Env (browser) | Phase | Status | Notes |
|------|-------------|---------------|-------|--------|-------|
| `ff_enable_app_shell` | `FF_ENABLE_APP_SHELL` | `VITE_FF_ENABLE_APP_SHELL` | Phase 0 | In Dev | App shell, layout, routing, design tokens |

---

## Domain Layer

| Flag | Env (server) | Env (browser) | Phase | Status | Notes |
|------|-------------|---------------|-------|--------|-------|
| `ff_enable_manual_meeting_import` | `FF_ENABLE_MANUAL_MEETING_IMPORT` | `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT` | Phase 1 | In Dev | Manual meeting creation with transcript or notes import |
| `ff_enable_meeting_memory_processing` | `FF_ENABLE_MEETING_MEMORY_PROCESSING` | `VITE_FF_ENABLE_MEETING_MEMORY_PROCESSING` | Phase 2 | In Implementation | Meeting → memory (summary, decisions, action items, risks, questions, stakeholder notes, next recommended focus) with version history, conflict review queue, and evidence-grounding |
| `ff_enable_pre_call_briefing` | `FF_ENABLE_PRE_CALL_BRIEFING` | `VITE_FF_ENABLE_PRE_CALL_BRIEFING` | Phase 2 | Planned | Briefing generation |
| `ff_enable_upcoming_meetings` | `FF_ENABLE_UPCOMING_MEETINGS` | `VITE_FF_ENABLE_UPCOMING_MEETINGS` | Phase 2 | Active | Manual upcoming meeting creation — BRD-05 |

| `ff_enable_manual_memory_correction` | `FF_ENABLE_MANUAL_MEMORY_CORRECTION` | `VITE_FF_ENABLE_MANUAL_MEMORY_CORRECTION` | Phase 2 | Planned | User corrections to meeting summaries |

---

## Infra Layer

| Flag | Env (server) | Env (browser) | Phase | Status | Notes |
|------|-------------|---------------|-------|--------|-------|
| `ff_enable_provider_connectors` | `FF_ENABLE_PROVIDER_CONNECTORS` | `VITE_FF_ENABLE_PROVIDER_CONNECTORS` | Future | Planned | Teams, Meet, Zoom, calendar integrations |

---

## Security Layer

| Flag | Env (server) | Env (browser) | Phase | Status | Notes |
|------|-------------|---------------|-------|--------|-------|
| `ff_enable_privacy_retention_controls` | `FF_ENABLE_PRIVACY_RETENTION_CONTROLS` | `VITE_FF_ENABLE_PRIVACY_RETENTION_CONTROLS` | Phase 2 | Planned | PII masking, retention policies, data export |

---

## Adding a New Flag

1. Add row to the appropriate section above
2. Add `FF_ENABLE_<FEATURE>=false` to `.env.example`
3. Add `VITE_FF_ENABLE_<FEATURE>=false` to `.env.example` if browser-exposed
4. Create BRD in `specs/<domain>/brd-XX-<slug>.md`
5. Create eval in `evals/e2e/brd-XX-<slug>.md`
6. Add to `specs/feature-flags.md` lifecycle tracking

---

## Local Demo Override

```bash
# Enable app shell for local demo
FF_ENABLE_APP_SHELL=true make dev
```

---

## Removal Process

1. Mark `Deprecated` in this registry
2. Remove all code references behind the flag
3. Confirm zero remaining `ff_enable_<feature>` in codebase
4. Remove from registry and mark `Removed`
