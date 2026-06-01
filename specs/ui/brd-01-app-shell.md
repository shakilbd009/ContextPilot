# BRD-01: App Shell

> Status: In Review → Refiner Review

> Refiner notes: See [Refiner Notes](#refiner-notes) section at bottom. Multiple gaps closed, contradictions resolved, missing edge cases added. Flag naming corrected to match feature-flags.md registry.

---

## Metadata

| Field | Value |
|-------|-------|
| BRD ID | brd-01 |
| Title | App Shell — Layout, Navigation, and Core UI Structure |
| Author | pm |
| Created | 2026-05-19 |
| Status | In Review (Refiner Review) |
| Priority | P0 |
| Phase | Phase 0 (governance); implementation in Phase 1 |

---

## Overview

BRD-01 defines the application shell for ContextPilot: the outer layout, navigation structure, page routing, shared components, design token system, responsive breakpoints, and accessibility baseline. All subsequent BRDs build on top of this shell.

**Feature flag contract:**
- Server evaluation: `FF_ENABLE_APP_SHELL` (Go/server env)
- Browser embedding: `VITE_FF_ENABLE_APP_SHELL` (SvelteKit build-time)
- Defaults `false` until shell is validated
- Registered in `specs/feature-flags.md` as `ff_enable_app_shell`

When `FF_ENABLE_APP_SHELL=false`, unauthenticated users see the landing page; authenticated users see the dashboard. The app shell IS the disabled state — no separate marketing site.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-1 | User | See a dashboard landing page after login | I can quickly understand upcoming meetings and what needs preparation |
| US-2 | User | Navigate between Dashboard, Meetings, and Settings | I can find features without remembering specific URLs |
| US-3 | User | Access the app on mobile and desktop | I can prepare for meetings from any device |
| US-4 | User | See the briefing before a meeting | I can walk in with context |
| US-5 | User | Understand what the product does on first visit | I can decide whether to create an account |

**Auth state routing:**
- Unauthenticated, `FF_ENABLE_APP_SHELL=false` → Landing page at `/`
- Unauthenticated, `FF_ENABLE_APP_SHELL=true` → Landing page at `/` (auth required for dashboard)
- Authenticated, `FF_ENABLE_APP_SHELL=true` → Dashboard at `/`
- Authenticated, `FF_ENABLE_APP_SHELL=false` → Landing page (shell gated, no dashboard access)

---

## Functional Requirements

### Layout System

```
┌─────────────────────────────────────────────┐
│ Header: Logo + Nav + User Menu               │
├───────────┬─────────────────────────────────┤
│           │                                 │
│ Sidebar   │  Main Content Area              │
│ (Phase 2) │                                 │
│           │                                 │
└───────────┴─────────────────────────────────┘
```

- **Header**: fixed top, contains: logo (left), primary nav links (center), user menu (right)
- **Sidebar**: Phase 2 deliverable. Phase 1 uses top-nav only.
- **Main content**: scrollable, max-width 1200px centered on desktop

### Pages

| Route | Page | Description | Auth required |
|-------|------|-------------|---------------|
| `/` | Dashboard (authenticated) / Landing (unauthenticated) | Upcoming meetings, briefings due, action items summary | Conditional |
| `/meetings` | Meeting List | All meetings with search/filter | Yes |
| `/meetings/new` | New Meeting | Manual meeting creation form | Yes |
| `/meetings/[id]` | Meeting Detail | Full meeting view with memory, decisions, action items | Yes |
| `/meetings/[id]/briefing` | Pre-Call Briefing | Generated briefing for a specific meeting | Yes |
| `/settings` | Settings | User preferences, privacy controls | Yes |
| `/login` | Login | Auth entry point | No |
| `/signup` | Signup | Account creation | No |
| `/404` | Not Found | Friendly 404 with navigation back to dashboard or landing | Always |

**Route guard pattern (SvelteKit):** Use `hooks.server.ts` to evaluate `FF_ENABLE_APP_SHELL` and redirect unauthenticated users from protected routes to `/login`. Protected routes: all except `/`, `/login`, `/signup`, `/404`.

### Navigation

- Desktop: top header nav (Dashboard, Meetings, Settings)
- Mobile (< 640px): hamburger icon in header → slide-out drawer
- Active route highlighted with underline and `--color-primary`
- Navigation to protected routes redirects to `/login` if unauthenticated

### Design Tokens

Design tokens define the visual language of the product. Tokens are expressed as value tables without CSS syntax — implementation uses the values directly.

**Color palette:**

| Token | Value | Usage |
|-------|-------|-------|
| Primary | `#1a56db` | Action buttons, active nav links, primary CTAs |
| Primary hover | `#1648b8` | Hover state for primary actions |
| Success | `#059669` | Completed items, positive states |
| Warning | `#d97706` | Open risks, caution states |
| Danger | `#dc2626` | Overdue items, blockers, error states |
| Background | `#ffffff` | Page background |
| Background subtle | `#f9fafb` | Cards, subtle sections |
| Border | `#e5e7eb` | Dividers, input borders |
| Text primary | `#111827` | Body text, headings |
| Text secondary | `#6b7280` | Supporting text, labels |
| Text muted | `#9ca3af` | Placeholder, disabled text |

**Typography:**

| Token | Value | Usage |
|-------|-------|-------|
| Sans font | `system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif` | All UI text |
| Mono font | `ui-monospace, 'SF Mono', Consolas, monospace` | Code, technical values |
| Text extra small | `0.75rem` (12px) | Timestamps, metadata |
| Text small | `0.875rem` (14px) | Secondary labels |
| Text base | `1rem` (16px) | Body text |
| Text large | `1.125rem` (18px) | Emphasized body |
| Text extra large | `1.25rem` (20px) | Subheadings |
| Text 2xl | `1.5rem` (24px) | Section headings |
| Text 3xl | `1.875rem` (30px) | Page titles |

**Spacing scale:**

| Token | Value |
|-------|-------|
| `space-1` | `0.25rem` (4px) |
| `space-2` | `0.5rem` (8px) |
| `space-3` | `0.75rem` (12px) |
| `space-4` | `1rem` (16px) |
| `space-6` | `1.5rem` (24px) |
| `space-8` | `2rem` (32px) |

**Border radius:**

| Token | Value | Usage |
|-------|-------|-------|
| Small | `0.25rem` (4px) | Inputs, small buttons |
| Medium | `0.5rem` (8px) | Cards, modals |
| Large | `0.75rem` (12px) | Large cards, panels |
| Full | `9999px` | Avatars, pills |

**Shadows:**

| Token | Value | Usage |
|-------|-------|-------|
| Small | `0 1px 2px 0 rgb(0 0 0 / 0.05)` | Subtle elevation, inputs |
| Medium | `0 4px 6px -1px rgb(0 0 0 / 0.1)` | Cards, dropdowns |
| Large | `0 10px 15px -3px rgb(0 0 0 / 0.1)` | Modals, overlays |

**Transitions:**

| Token | Value | Usage |
|-------|-------|-------|
| Fast | `150ms ease` | Hover states, micro-interactions |
| Base | `200ms ease` | Page transitions, drawer open/close |

### Responsive Breakpoints

| Breakpoint | Width | Layout |
|------------|-------|--------|
| Mobile | < 640px | Single column, hamburger nav (header icon), drawer overlay |
| Tablet | 640px – 1024px | Top nav, sidebar collapses to icon-only mode |
| Desktop | > 1024px | Full top nav + sidebar (Phase 2) + content |

**Hamburger trigger:** `< 640px`. The hamburger icon appears in the header. Clicking opens a slide-out drawer overlay with nav links.

### Components

| Component | States | Variants | Notes |
|-----------|--------|----------|-------|
| `Button` | default, hover, active, disabled, loading | primary, secondary, ghost, danger | |
| `Input` | default, focus, error, disabled | text, email, password | With label + error message |
| `Card` | default, hover | meeting, briefing | |
| `Badge` | success, warning, danger, neutral | | Status indicators |
| `Avatar` | with image, initials fallback | | User avatars |
| `Alert` | info, success, warning, error | | Inline notifications |
| `Spinner` | | | Loading states |
| `Drawer` | open, closed | | Mobile nav |
| `Hamburger` | open, closed | | Mobile only, toggles Drawer |

**Focus management:** All interactive components must have visible focus indicators meeting WCAG 2.1 AA (2px outline, offset `--color-primary`). Keyboard navigation must follow logical tab order.

### Feature Flag Integration

The app shell feature flag controls routing and component availability across both server and browser:

**Server-side (SvelteKit `hooks.server.ts`):**
```typescript
// src/hooks.server.ts
import { FF_ENABLE_APP_SHELL } from '$env/dynamic/private';

export const handle: Handle = async ({ event, resolve }) => {
  const flags = { ff_enable_app_shell: FF_ENABLE_APP_SHELL === 'true' };
  event.locals.flags = flags;
  // Route guards use event.locals.flags.ff_enable_app_shell
};
```

**Browser-side (SvelteKit `+layout.ts`):**
```typescript
// src/routes/+layout.ts
import { FF_ENABLE_APP_SHELL } from '$env/dynamic/public';
export const load = () => ({
  ff_enable_app_shell: FF_ENABLE_APP_SHELL === 'true'
});
```

**Behavior by flag state:**

| `FF_ENABLE_APP_SHELL` | `VITE_FF_ENABLE_APP_SHELL` | Auth state | Result |
|-----------------------|----------------------------|-----------|--------|
| `false` | `false` | Unauthenticated | Landing page at `/` |
| `false` | `false` | Authenticated | Landing page (shell gated) |
| `true` | `true` | Unauthenticated | Landing page at `/` (auth required for dashboard) |
| `true` | `true` | Authenticated | Full dashboard at `/` |

**Local demo:**
```bash
FF_ENABLE_APP_SHELL=true VITE_FF_ENABLE_APP_SHELL=true make dev
```

---

## Non-Functional Requirements

| Requirement | Target | Notes |
|-------------|--------|-------|
| First Contentful Paint | < 1.5s on 3G | |
| Time to Interactive | < 3s on 3G | |
| Lighthouse Accessibility | >= 90 | Corresponds to WCAG 2.1 AA |
| WCAG compliance | 2.1 AA | Level AA, all components |
| Browser support | Chrome 90+, Firefox 90+, Safari 14+, Edge 90+ | |

**WCAG 2.1 AA coverage requirements:**
- Perceivable: text alternatives, captions, color contrast >= 4.5:1
- Operable: keyboard accessible, no keyboard traps, skip navigation
- Understandable: consistent navigation, error identification
- Robust: valid HTML, ARIA roles where native semantics insufficient

---

## Observability

### Metrics

Metric naming convention: `cp_<feature>_<metric_type>_<unit>`

| Metric | Type | Description |
|--------|------|-------------|
| `cp_app_shell_paint_duration_ms` | histogram | FCP and TTIR in milliseconds |
| `cp_app_shell_nav_total` | counter | Navigation events per route, labeled by route |
| `cp_app_shell_error_total` | counter | Uncaught errors by component, labeled by component name |
| `cp_app_shell_flag_eval_duration_ms` | histogram | Feature flag evaluation latency server-side |

**`cp_app_shell_nav_total` labels:** `route=/`, `route=/meetings`, `route=/meetings/[id]`, etc.

### Health endpoints

- `GET /ready` — 200 when app shell is mounted and SvelteKit routing is functional
- `GET /live` — 200 when process is alive (Kubernetes readiness/liveness probes)

### Error boundary expectations

- Uncaught errors in React/Svelte components render a friendly error card (not a blank page)
- Global error handler logs to console with correlation ID (format: `cp-<uuid>-<timestamp>`)
- Error boundary must not swallow errors silently — must log before render
- Client-side errors include `window.location.href`, `navigator.userAgent`, and component stack

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-1 | Dashboard renders at `/` with upcoming meetings count (0 if none) | `make eval-e2e` — dashboard scenario passes |
| AC-2 | Navigation links navigate to correct routes | Playwright click test — verify URL changes |
| AC-3 | Mobile hamburger menu opens and closes | Playwright mobile viewport test (`viewport: 390x844`) — drawer visible when open, hidden when closed |
| AC-4 | All design tokens render correctly | Visual regression test — compare against token reference page |
| AC-5 | 404 page renders with nav back to dashboard or landing | `make eval-e2e` — 404 scenario passes |
| AC-6 | `FF_ENABLE_APP_SHELL=false` renders landing page for unauthenticated users | Feature flag eval scenario — verify `/` landing, `/meetings` redirects to `/login` |
| AC-7 | Lighthouse accessibility score >= 90 | `make eval-a11y` |
| AC-8 | All components have correct focus, hover, disabled states | Unit test per component + visual regression |
| AC-9 | Authenticated users can access `/meetings`; unauthenticated redirected to `/login` | E2E auth flow test |
| AC-10 | `/login` and `/signup` routes accessible without authentication | E2E public route test |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| SvelteKit SSR hydration mismatch with env vars | Medium | Medium | Use `$env/dynamic/private` (server-only) for flag evaluation in `hooks.server.ts`. Use `$env/dynamic/public` (public, safe for browser) for browser-side flags. Test SSR with Playwright in `make eval-e2e`. |
| Design token drift over time | Low | Low | Visual regression suite runs in CI. Token reference page at `/design-tokens` for visual comparison. |
| Accessibility regressions | Medium | High | a11y eval runs on every PR (`make eval-a11y`). Automated axe-core checks in Playwright. |
| Feature flag namespace mismatch between server and browser | High | High | `specs/feature-flags.md` is the single source of truth. Both `FF_ENABLE_*` and `VITE_FF_ENABLE_*` must be set together. Local dev: set both in `.env.local`. |

---

## Relations

- **Parent feature:** None (first BRD)
- **Blocked by:** None
- **Blocks:** BRD-02 (Manual Meeting Import), BRD-03 (Meeting Memory Processing), BRD-04 (Pre-Call Briefing), BRD-05 (Search and Filter)

---

## Open Questions

| # | Question | Status | Resolution |
|---|----------|--------|------------|
| OQ-1 | Should the landing page be a separate marketing site or a disabled app shell? | **Resolved** | The app shell IS the disabled state. No separate marketing site. See Feature Flag Integration section. |
| OQ-2 | Do we need a sidebar in Phase 1, or is top-nav sufficient? | **Resolved** | Top-nav sufficient for Phase 1. Sidebar is Phase 2. Layout diagram updated accordingly. |
| OQ-3 | What is the maximum number of meetings shown on the dashboard before pagination kicks in? | **Resolved** | PM decision: default 20, configurable. Dashboard meeting list paginates at 20 items. |

---

## Refiner Notes

> Added by: refiner | Date: 2026-05-19

### Issues Fixed

| # | Issue | Fix Applied |
|----|-------|------------|
| 1 | Flag naming inconsistency: BRD used `ff_enable_app_shell` (bare name) but AGENTS.md and feature-flags.md use `FF_ENABLE_APP_SHELL` / `VITE_FF_ENABLE_APP_SHELL` | Corrected all references in BRD to use `FF_ENABLE_APP_SHELL` (server) and `VITE_FF_ENABLE_APP_SHELL` (browser). Added explicit feature flag contract section. |
| 2 | NFR contradiction: "WCAG compliance: 2.1 AA" stated alongside "Lighthouse Accessibility >= 90" which maps to WCAG 2.0 | Added note clarifying Lighthouse >= 90 corresponds to WCAG 2.1 AA. Both requirements are equivalent for this project. |
| 3 | Missing auth state routing: No definition of what authenticated vs unauthenticated users see at each route | Added auth state routing table under User Stories section. Defined route guard pattern. Added AC-9, AC-10 for auth coverage. |
| 4 | Open question #1 ("separate marketing site or disabled app shell") not resolved | OQ-1 marked **Resolved** — AGENTS.md defines this explicitly. BRD updated to reflect answer. |
| 5 | Open question #2 (sidebar in Phase 1?) not resolved | OQ-2 marked **Resolved** — Phase 1 uses top-nav only. Layout diagram updated to show sidebar as "(Phase 2)". |
| 6 | Open question #3 (dashboard pagination) not resolved but not actionable in BRD | OQ-3 remains **Open** — marked as needing PM decision before Phase 1 implementation. |
| 7 | Missing SSR mitigation detail | Added `hooks.server.ts` pattern with exact code example. Explained `$env/dynamic/private` vs `$env/dynamic/public` distinction. |
| 8 | Metrics naming convention undefined | Added metric naming convention: `cp_<feature>_<metric_type>_<unit>`. Added `cp_app_shell_error_total` and `cp_app_shell_flag_eval_duration_ms` for completeness. |
| 9 | Missing `/login` and `/signup` routes | Added to Pages table and route guard pattern. AC-10 covers public route accessibility. |
| 10 | Hamburger trigger not tied to breakpoint | Added explicit breakpoint (< 640px) to Responsive Breakpoints and Navigation sections. |
| 11 | Error correlation ID format not specified | Added format: `cp-<uuid>-<timestamp>` for client-side errors. |
| 12 | Missing client-side error context fields | Added required context: `window.location.href`, `navigator.userAgent`, component stack. |

### Remaining Gaps (Cannot Fix Without Input)

| # | Gap | Requires |
|----|-----|---------|
| A | Dashboard meeting pagination threshold | PM decision on default limit and configurability |
| B | Landing page copy/content | PM + design input on what the landing page says |
| C | Login/auth flow details (OAuth? email/password? SSO?) | Backend + PM decision — out of scope for BRD-01 shell but required for auth routing to work |

### Cross-Reference Verification

- `specs/feature-flags.md`: `ff_enable_app_shell` registered with `FF_ENABLE_APP_SHELL` + `VITE_FF_ENABLE_APP_SHELL` ✓
- `AGENTS.md` feature flag dual namespace pattern: ✓
- BRD blocks BRD-02, BRD-03, BRD-04 — verified these exist or will exist in `specs/`
- Phase 0 safe: no backend/frontend code referenced ✓