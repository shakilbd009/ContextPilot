# ADR-001: Record Architecture Decisions

> Status: Accepted

## Context

ContextPilot is a meeting intelligence system. The product requires a reliable backend for meeting memory storage, transcript processing, and briefing generation, paired with a responsive UI that delivers pre-call briefings to users. We need to select our core technology stack before work begins.

## Decision

We will use **Go (Echo framework)** for the backend REST API and **SvelteKit** for the frontend application.

### Backend — Go + Echo

- Language: **Go 1.26.3**
- Web framework: **Echo v4** (or compatible)
- Database: **PostgreSQL** (via Docker Compose)
- Cache: **Redis** (via Docker Compose)
- Migration tool: built-in Go migration runner

### Frontend — SvelteKit

- Framework: **SvelteKit** (latest stable)
- Package manager: **pnpm** (version pinned to 10.32.1)
- Runtime: **Node 22**
- SSR enabled for briefing generation pages

## Rationale

**Go** provides:
- Strong typing and compile-time safety for domain models (Meeting, ActionItem, Decision, Risk)
- Native concurrency for parallel briefing generation
- Fast cold-start for serverless/containers
- Mature ecosystem for PostgreSQL drivers and Redis clients

**SvelteKit** provides:
- File-based routing that maps cleanly to our page structure (Dashboard, Meeting Detail, Briefing, Settings)
- SSR for SEO and fast initial load on briefing pages
- Form actions for manual meeting input workflows
- Small bundle size and fast hydration

**Echo** over other Go frameworks:
- Simpler middleware model than Gin
- Built-in parameter binding reduces boilerplate
- Consistent API surface across handlers

## Trade-offs

| Choice | What we give up |
|--------|-----------------|
| Go | Faster initial development vs Python/FastAPI for simple CRUD |
| SvelteKit | Larger ecosystem vs Next.js; fewer hiring options |
| pnpm | Wide contributor familiarity vs npm/yarn |

## Consequences

1. Backend engineers need Go experience; onboarding docs will cover Echo patterns.
2. Frontend engineers need SvelteKit familiarity; component library conventions will be documented in BRD-01.
3. Both `FF_ENABLE_*` (server) and `VITE_FF_ENABLE_*` (browser) namespaces must be kept in sync for features that cross the server/browser boundary.
4. Phase 0 infra (Docker Compose) provisions PostgreSQL and Redis so Phase 1 starts with a runnable data layer.

## Status

Accepted — 2026-05-19

## Related Decisions

| Decision | Feature | Status |
|----------|---------|--------|
| [ADR-002](0002-manual-meeting-import-validation-strategy.md) | BRD-02 Manual Meeting Import — Validation Strategy | Accepted |
| [ADR-003](0003-manual-meeting-import-data-model.md) | BRD-02 Manual Meeting Import — Data Model | Accepted |
| [ADR-004](0004-manual-meeting-import-form-content-preservation.md) | BRD-02 Manual Meeting Import — Form Content Preservation | Accepted |