# E2E Eval: brd-01-app-shell

> 🔴 Failing — implementation pending

## Scope

End-to-end scenarios for the App Shell (BRD-01): layout, navigation, routing, design tokens, responsive behavior, and accessibility.

## Scenario: Dashboard renders with no meetings

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`
- No meetings exist in the system

### When
User navigates to `/`

### Then
- Dashboard page renders without error
- Empty state message is shown: "No meetings yet"
- Nav shows: Dashboard, Meetings, Settings

---

## Scenario: Navigation links work

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`

### When
User clicks the "Meetings" nav link

### Then
- URL changes to `/meetings`
- Meeting list page renders

---

## Scenario: Mobile hamburger menu

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`
- Viewport is set to 390x844 (mobile)

### When
User taps the hamburger menu icon

### Then
- Drawer slides in from the left
- Nav links (Dashboard, Meetings, Settings) are visible
- Tapping outside or pressing Escape closes the drawer

---

## Scenario: Unauthenticated user redirected to login

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`
- User is unauthenticated (no active session)

### When
User navigates to `/meetings`

### Then
- User is redirected to `/login`
- URL shows `/login`
- No meeting list content is visible

---

## Scenario: Public routes are accessible without auth

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`
- User is unauthenticated (no active session)

### When
User navigates to `/login` and `/signup`

### Then
- `/login` page renders without redirect
- `/signup` page renders without redirect
- No auth gate or redirect to `/login` occurs on either route

---

## Scenario: Dashboard list shows "View all" at 20+ meetings threshold

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`
- The database contains 20 or more meetings

### When
User navigates to `/meetings`

### Then
- Meeting list renders with pagination or a "View all meetings" affordance
- The list does not render all meetings at once without pagination
- Threshold behavior matches the 20-meeting limit defined in AC-11

---

## Scenario: 404 renders with nav back to dashboard

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`

### When
User navigates to a non-existent route `/this-does-not-exist`

### Then
- 404 page renders with message "Page not found"
- Link back to Dashboard is visible and works

---

## Scenario: App shell is disabled when flag is false

### Given
- The app is running with `FF_ENABLE_APP_SHELL=false`

### When
User navigates to `/`

### Then
- Landing/marketing page renders (not the app shell)
- CTA to create account or sign in is visible

---

## Scenario: Design tokens are applied

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`

### When
Page loads at any route

### Then
- CSS custom properties from `:root` are present
- Color values match the design system in BRD-01
- Typography scale is correct

---

## Scenario: Accessibility — keyboard navigation

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`

### When
User navigates using only the Tab key

### Then
- Focus order is logical (header → nav → main content)
- Focus indicator is visible
- No focus trap in any element

---

## Scenario: Accessibility — screen reader

### Given
- The app is running with `FF_ENABLE_APP_SHELL=true`

### When
Page is read by a screen reader (axe-core)

### Then
- All images have alt text
- Form inputs have associated labels
- ARIA landmarks are present
- No critical violations

---

## Running

```bash
# Requires services up
docker compose up -d
pnpm exec playwright test --reporter=list
```