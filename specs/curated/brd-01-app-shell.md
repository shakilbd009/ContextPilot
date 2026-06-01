# Curated Spec Package: BRD-01 App Shell

> Curated by: pm
> Date: 2026-05-19
> Source BRD: `specs/ui/brd-01-app-shell.md`
> Status: Curated / Ready for Phase 1 implementation planning
> Feature flag: `ff_enable_app_shell`
> Server env: `FF_ENABLE_APP_SHELL`
> Browser env: `PUBLIC_VITE_FF_ENABLE_APP_SHELL`

---

## 1. Package Purpose

This package consolidates the BRD, architect handoff, and refiner handoff for BRD-01 App Shell into a single implementation-ready reference. It preserves the original BRD as the source of record while making final PM decisions explicit where architect/refiner output diverged.

Primary implementation goal: deliver the ContextPilot application shell for authenticated app users while preserving a landing page for unauthenticated users and keeping all shell behavior behind the dual namespace feature flag contract.

---

## 2. Inputs Curated

| Source | Task | Contribution | Disposition |
|--------|------|--------------|-------------|
| Original BRD | `specs/ui/brd-01-app-shell.md` | Product scope, routes, layout, design tokens, acceptance criteria | Accepted as source BRD |
| Architect handoff | `t_ed8e8330` | ADRs, routing/design token/flag design, data flows, failure modes, observability, security, scale considerations | Accepted with one PM override on dashboard pagination status |
| Refiner handoff | `t_1523b8b9` | Corrected flag naming, auth routing, route gaps, NFR clarification, metrics, error correlation format, unresolved gaps | Accepted; PM resolved OQ-3 in this package |
| Feature flag registry | `specs/feature-flags.md` | Confirms `ff_enable_app_shell`, `FF_ENABLE_APP_SHELL`, `PUBLIC_VITE_FF_ENABLE_APP_SHELL` | Verified consistent |
| Project rules | `AGENTS.md` | Spec-driven/eval-driven/feature-flagged operating rules | Applied |

---

## 3. Final Scope

### In Scope for BRD-01

1. Top-level app shell layout:
   - Fixed header with logo, primary navigation, and user menu.
   - Main content region centered at max-width 1200px on desktop.
   - Mobile drawer navigation triggered below 640px.
   - Sidebar deferred to Phase 2.

2. Route structure:
   - `/` as conditional route: landing for unauthenticated users, dashboard for authenticated users when enabled.
   - `/meetings` meeting list.
   - `/meetings/new` meeting creation entry point.
   - `/meetings/[id]` meeting detail.
   - `/meetings/[id]/briefing` pre-call briefing.
   - `/settings` user settings.
   - `/login` public login.
   - `/signup` public signup.
   - `/404` friendly not found route.

3. Component baseline:
   - `Button`, `Input`, `Card`, `Badge`, `Avatar`, `Alert`, `Spinner`, `Drawer`, `Hamburger`.
   - Required states and variants as defined in the source BRD.
   - WCAG 2.1 AA focus behavior for all interactive components.

4. Design token system:
   - CSS custom properties for colors, typography, spacing, border radius, shadows, and transitions.
   - Token values are defined in `specs/ui/brd-01-app-shell.md` and should be implemented without drift.

5. Feature flag integration:
   - Server-side route evaluation uses `FF_ENABLE_APP_SHELL`.
   - Browser-side shell embedding uses `PUBLIC_VITE_FF_ENABLE_APP_SHELL`.
   - Both default to `false` and must be set together for local demo behavior.

6. Observability:
   - Metrics listed in section 8 of this package.
   - `/ready` and `/live` health endpoints.
   - Error boundary behavior with correlation IDs.

### Out of Scope for BRD-01

1. Phase 2 sidebar implementation.
2. Provider connectors for Teams, Meet, Zoom, or calendars.
3. Meeting memory processing internals.
4. Pre-call briefing generation internals.
5. Final auth provider selection beyond required public/protected route behavior.
6. Final landing page marketing copy beyond minimum shell-safe placeholder content.

---

## 4. Final PM Decisions

| Decision | Final Position | Rationale |
|----------|----------------|-----------|
| Landing page vs separate marketing site | The app shell disabled state is the landing page; no separate marketing site for BRD-01 | Matches `AGENTS.md` and refiner correction |
| Phase 1 navigation | Top-nav only on tablet/desktop; hamburger drawer on mobile; sidebar is Phase 2 | Keeps Phase 1 small and avoids premature IA complexity |
| Dashboard pagination threshold | Dashboard shows up to 20 meetings before pagination or "View all meetings" affordance appears; threshold must be configurable in implementation | Resolves OQ-3 and aligns with refiner suggestion |
| Feature flag naming | Registry name `ff_enable_app_shell`; server env `FF_ENABLE_APP_SHELL`; browser env `PUBLIC_VITE_FF_ENABLE_APP_SHELL` | Matches `specs/feature-flags.md` and project dual namespace rule |
| Auth details | BRD-01 requires route guard behavior but does not choose OAuth/email/SSO | Keeps auth provider selection out of shell scope |
| Landing copy | Implementation may use minimal product-safe placeholder copy until design/PM supplies final copy | Avoids blocking shell implementation on content polish |

---

## 5. Conflict Resolution

The architect handoff reported all 3 BRD open questions resolved. The refiner handoff reported OQ-3, dashboard pagination threshold, still open.

PM resolution: treat the refiner handoff as more precise for OQ-3 because it identified the missing concrete threshold. This package resolves OQ-3 with a default threshold of 20 meetings and a requirement that the limit be configurable in implementation.

No other conflicts remain between architect and refiner output.

---

## 6. Feature Flag Contract

| Layer | Name | Default | Required behavior |
|-------|------|---------|-------------------|
| Registry | `ff_enable_app_shell` | `false` | Registered in `specs/feature-flags.md` |
| Server env | `FF_ENABLE_APP_SHELL` | `false` | Controls route guard and authenticated shell availability |
| Browser env | `PUBLIC_VITE_FF_ENABLE_APP_SHELL` | `false` | Controls browser shell embedding and client-visible behavior |

Flag behavior matrix:

| `FF_ENABLE_APP_SHELL` | `PUBLIC_VITE_FF_ENABLE_APP_SHELL` | Auth state | Result |
|-----------------------|----------------------------|-----------|--------|
| `false` | `false` | Unauthenticated | Landing page at `/` |
| `false` | `false` | Authenticated | Landing page at `/`; dashboard gated |
| `true` | `true` | Unauthenticated | Landing page at `/`; protected routes redirect to `/login` |
| `true` | `true` | Authenticated | Dashboard/app shell at `/` and protected routes accessible |

Implementation guardrail: server and browser flags must stay synchronized for demos, local dev, and CI. If they diverge, server behavior is authoritative for protected route access.

Local demo command:

```bash
FF_ENABLE_APP_SHELL=true PUBLIC_VITE_FF_ENABLE_APP_SHELL=true make dev
```

---

## 7. Route and Auth Contract

| Route | Page | Auth required | Flag behavior |
|-------|------|---------------|---------------|
| `/` | Landing or dashboard | Conditional | Dashboard only when authenticated and app shell enabled |
| `/meetings` | Meeting List | Yes | Redirect unauthenticated users to `/login` |
| `/meetings/new` | New Meeting | Yes | Redirect unauthenticated users to `/login` |
| `/meetings/[id]` | Meeting Detail | Yes | Redirect unauthenticated users to `/login` |
| `/meetings/[id]/briefing` | Pre-Call Briefing | Yes | Redirect unauthenticated users to `/login` |
| `/settings` | Settings | Yes | Redirect unauthenticated users to `/login` |
| `/login` | Login | No | Always accessible |
| `/signup` | Signup | No | Always accessible |
| `/404` | Friendly Not Found | Always | Must link back to dashboard if authenticated, landing if unauthenticated |

Protected routes: all routes except `/`, `/login`, `/signup`, and `/404`.

SvelteKit guard location: `hooks.server.ts` should evaluate the server feature flag and route auth state before protected route rendering.

---

## 8. Observability Contract

Metric naming convention: `cp_<feature>_<metric_type>_<unit>`.

| Metric | Type | Description | Required labels |
|--------|------|-------------|-----------------|
| `cp_app_shell_paint_duration_ms` | histogram | First Contentful Paint and Time to Interactive | `route`, `phase` |
| `cp_app_shell_nav_total` | counter | Navigation events by route | `route` |
| `cp_app_shell_error_total` | counter | Uncaught errors by component | `component`, `route` |
| `cp_app_shell_flag_eval_duration_ms` | histogram | Server-side feature flag evaluation latency | `flag`, `result` |

Health endpoints:

| Endpoint | Success condition |
|----------|-------------------|
| `GET /ready` | App shell is mounted and SvelteKit routing is functional |
| `GET /live` | Process is alive |

Error boundary requirements:

1. Render a friendly error card rather than a blank page.
2. Log before rendering fallback UI.
3. Include correlation ID format `cp-<uuid>-<timestamp>`.
4. Include `window.location.href`, `navigator.userAgent`, and component stack for client-side errors.

---

## 9. Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| First Contentful Paint | Less than 1.5s on 3G |
| Time to Interactive | Less than 3s on 3G |
| Lighthouse Accessibility | Greater than or equal to 90 |
| WCAG compliance | WCAG 2.1 AA |
| Browser support | Chrome 90+, Firefox 90+, Safari 14+, Edge 90+ |

Accessibility implementation minimums:

1. Color contrast at least 4.5:1 for normal text.
2. Keyboard accessible navigation with no keyboard traps.
3. Visible focus indicators: 2px outline with `--color-primary` and appropriate offset.
4. Skip navigation support.
5. Native semantic HTML preferred over ARIA; ARIA only where native semantics are insufficient.

---

## 10. Acceptance Criteria for Implementation

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | Dashboard renders at `/` with upcoming meetings count, including zero state | `make eval-e2e` dashboard scenario |
| AC-2 | Navigation links route correctly | Playwright click test verifies URL changes |
| AC-3 | Mobile hamburger opens and closes drawer at 390x844 viewport | Playwright mobile viewport test |
| AC-4 | Design tokens match the BRD values | Visual regression against token reference page |
| AC-5 | 404 page renders with navigation back to dashboard or landing | `make eval-e2e` 404 scenario |
| AC-6 | App shell disabled state renders landing page and protected routes redirect correctly | Feature flag eval scenario |
| AC-7 | Lighthouse accessibility score is at least 90 | `make eval-a11y` |
| AC-8 | Components implement required focus, hover, disabled, loading, and open/closed states | Unit tests plus visual regression |
| AC-9 | Authenticated users can access `/meetings`; unauthenticated users redirect to `/login` | E2E auth flow test |
| AC-10 | `/login` and `/signup` are public | E2E public route test |
| AC-11 | Dashboard list applies the 20-meeting threshold before pagination or "View all meetings" affordance | E2E dashboard list threshold test |

---

## 11. Implementation Handoff

Recommended assignment split:

| Workstream | Owner profile | Done means |
|------------|---------------|------------|
| E2E and a11y eval contracts | `validator` or `spec-writer` | Failing evals exist for AC-1 through AC-11 before implementation |
| SvelteKit app shell implementation | `frontend-eng` | Routes, layout, design tokens, components, responsive nav, and flag behavior implemented |
| Route guard and health endpoints | `frontend-eng` with backend review if server integration is needed | `hooks.server.ts`, `/ready`, `/live`, and protected route redirects satisfy spec |
| Observability instrumentation | `frontend-eng` with `ops` review if metrics plumbing is shared | Metrics emit with required names and labels |
| Final validation | `validator` | E2E, a11y, visual, and flag parity checks pass |

Implementation must not bypass the feature flag. No app shell code should be considered complete unless the disabled flag path still works.

---

## 12. Traceability Matrix

| Requirement area | Source BRD section | Architect/refiner contribution | Acceptance coverage |
|------------------|--------------------|--------------------------------|---------------------|
| Layout and responsive shell | Functional Requirements, Responsive Breakpoints | Architect component/data flow design; refiner breakpoint clarification | AC-2, AC-3 |
| Routes and auth guard | Pages, User Stories | Refiner added `/login`, `/signup`, route guard pattern | AC-6, AC-9, AC-10 |
| Design tokens | Design Tokens | Architect ADR on token delivery | AC-4 |
| Components | Components | Architect component specs; refiner Drawer/Hamburger states | AC-8 |
| Feature flags | Feature Flag Integration | Architect dual namespace ADR; refiner naming correction | AC-6 |
| Accessibility | NFRs, Components | Refiner clarified WCAG/Lighthouse relationship | AC-7, AC-8 |
| Observability | Observability | Architect instrumentation; refiner metric naming/correlation fields | AC-1 through AC-11 plus metric checks |
| Dashboard pagination | Open Questions | Refiner left open; PM resolved in this package | AC-11 |

---

## 13. Remaining Follow-Up Outside This Package

These are not blockers for accepting the curated package, but they must be handled before a polished product release:

1. Final landing page copy/content: PM/design decision.
2. Final auth provider flow: PM/backend decision.
3. Sidebar IA: Phase 2 decision.
4. Provider connector routes and data: future BRDs.

---

## 14. Curated Decision Summary

BRD-01 is ready to move from review into implementation planning with one explicit PM decision added: the dashboard shows up to 20 meetings before pagination or a "View all meetings" affordance appears, and that limit must be configurable. The app shell must be implemented as a feature-flagged SvelteKit shell with top navigation for Phase 1, mobile drawer below 640px, WCAG 2.1 AA component behavior, and protected-route redirects through server-side guard logic.
