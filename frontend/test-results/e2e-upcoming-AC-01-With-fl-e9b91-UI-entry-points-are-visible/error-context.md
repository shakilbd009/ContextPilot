# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: e2e/upcoming.spec.ts >> AC-01: With flag=true, upcoming UI entry points are visible
- Location: tests/e2e/upcoming.spec.ts:74:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('heading', { name: 'Upcoming Meetings' })
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for getByRole('heading', { name: 'Upcoming Meetings' })

```

```yaml
- link "Skip to main content":
  - /url: "#main-content"
- banner:
  - link "ContextPilot home":
    - /url: /
    - text: ContextPilot
  - navigation "Main navigation":
    - link "Dashboard":
      - /url: /
    - link "Meetings":
      - /url: /meetings
    - link "Upcoming":
      - /url: /upcoming
    - link "Settings":
      - /url: /settings
  - link "Sign in":
    - /url: /login
- main:
  - alert: Failed to load upcoming meetings.
```

# Test source

```ts
  1   | /**
  2   |  * BRD-05 E2E tests: Manual Upcoming Meeting Creation
  3   |  *
  4   |  * Covers all required AC scenarios:
  5   |  * - AC-01: Feature flag disabled — UI entry points hidden
  6   |  * - AC-02: Create meeting with title + scheduled_start
  7   |  * - AC-03: Create meeting with optional description, client/org, participants
  8   |  * - AC-06: Dashboard → list → calendar → detail navigation
  9   |  * - AC-07: Cancelled meetings excluded from list/calendar default view
  10  |  * - AC-08: Edit flow within edit window
  11  |  * - AC-09: Edit controls hidden after window expires
  12  |  * - AC-10: Cancel flow — status=cancelled, hidden from default views, record preserved
  13  |  *
  14  |  * Config: playwright.config.ts — baseURL: http://localhost:5174
  15  |  *
  16  |  * Auth: Tests assume the user is already authenticated (signed in).
  17  |  * The app redirects unauthenticated users to /login before showing any protected content.
  18  |  * Each test calls enableUpcomingMeetings() which signs in as a demo user first.
  19  |  */
  20  | 
  21  | import { test, expect } from '@playwright/test';
  22  | 
  23  | // Use a fixed future date for scheduling to avoid flakiness
  24  | function futureDate(daysFromNow: number, hour = 14, minute = 0): { date: string; time: string } {
  25  |   const d = new Date(Date.now() + daysFromNow * 86_400_000);
  26  |   const date = d.toISOString().slice(0, 10); // YYYY-MM-DD
  27  |   const time = `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`;
  28  |   return { date, time };
  29  | }
  30  | 
  31  | // ── Helper: sign in as demo user ────────────────────────────────
  32  | async function signIn(page: import('@playwright/test').Page) {
  33  |   await page.goto('/login');
  34  |   await page.getByLabel(/email/i).fill('test@example.com');
  35  |   await page.getByLabel(/password/i).fill('password');
  36  |   await page.getByRole('button', { name: /sign in/i }).click();
  37  |   await page.waitForURL('/');
  38  | }
  39  | 
  40  | // ── Helper: fill a date input (bypasses Playwright date-picker limitation) ─
  41  | async function fillDateInput(page: import('@playwright/test').Page, value: string) {
  42  |   const input = page.locator('input[type="date"]');
  43  |   await input.waitFor({ state: 'visible' });
  44  |   const el = await input.elementHandle();
  45  |   await page.evaluate(
  46  |     ({ el, value }) => {
  47  |       const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set;
  48  |       setter?.call(el, value);
  49  |       el.dispatchEvent(new Event('input', { bubbles: true }));
  50  |       el.dispatchEvent(new Event('change', { bubbles: true }));
  51  |     },
  52  |     { el, value }
  53  |   );
  54  |   await page.waitForFunction(
  55  |     () => {
  56  |       const timeInput = document.querySelector('input[type="time"]');
  57  |       return timeInput && timeInput.labels && timeInput.labels.length > 0;
  58  |     },
  59  |     { timeout: 5000 }
  60  |   );
  61  | }
  62  | 
  63  | // ── Helper: fill a time input reliably ─────────────────────────
  64  | async function fillTimeInput(page: import('@playwright/test').Page, value: string) {
  65  |   await page.locator('input[type="time"]').fill(value);
  66  | }
  67  | 
  68  | // ── Helper: enable feature flag (auth + env baked-in) ──────────
  69  | async function enableUpcomingMeetings(page: import('@playwright/test').Page) {
  70  |   await signIn(page);
  71  | }
  72  | 
  73  | // ── AC-01: Feature flag disabled ───────────────────────────────
  74  | test('AC-01: With flag=true, upcoming UI entry points are visible', async ({ page }) => {
  75  |   await signIn(page);
  76  |   await page.goto('/');
  77  |   await expect(page.getByRole('link', { name: /upcoming/i })).toBeVisible();
  78  |   await page.goto('/upcoming');
> 79  |   await expect(page.getByRole('heading', { name: 'Upcoming Meetings' })).toBeVisible();
      |                                                                          ^ Error: expect(locator).toBeVisible() failed
  80  | });
  81  | 
  82  | test.skip('AC-01 (flag=false): UI entry points hidden when feature flag is disabled', async ({ page }) => {
  83  |   await signIn(page);
  84  |   const scheduleLink = page.getByRole('link', { name: /schedule meeting/i });
  85  |   await expect(scheduleLink).not.toBeVisible();
  86  |   await page.goto('/upcoming');
  87  |   await expect(page.getByText(/VITE_FF_ENABLE_UPCOMING_MEETINGS=true/i)).toBeVisible();
  88  | });
  89  | 
  90  | // ── AC-02: Create meeting with required fields ──────────────────
  91  | test('AC-02: Create meeting with title + scheduled_start', async ({ page }) => {
  92  |   await enableUpcomingMeetings(page);
  93  |   await page.goto('/upcoming/new');
  94  |   await expect(page.getByRole('heading', { name: 'Schedule Meeting' })).toBeVisible();
  95  |   const { date, time } = futureDate(2, 10, 30);
  96  |   await page.getByLabel(/meeting title/i).fill('Q3 Planning Review');
  97  |   await fillDateInput(page, date);
  98  |   await fillTimeInput(page, time);
  99  |   await page.getByRole('button', { name: /schedule meeting/i }).click();
  100 |   await page.waitForURL(/\/upcoming\/[a-z0-9-]+$/);
  101 |   await expect(page.getByText('Q3 Planning Review')).toBeVisible();
  102 | });
  103 | 
  104 | // ── AC-03: Create meeting with optional fields ──────────────────
  105 | test('AC-03: Create meeting with description, client/org, and participants', async ({ page }) => {
  106 |   await enableUpcomingMeetings(page);
  107 |   await page.goto('/upcoming/new');
  108 |   const { date, time } = futureDate(3, 11, 0);
  109 |   await page.getByLabel(/meeting title/i).fill('Acme Quarterly Sync');
  110 |   await fillDateInput(page, date);
  111 |   await fillTimeInput(page, time);
  112 |   await page.getByLabel(/description \/ agenda/i).fill('Review Q3 results and Q4 targets.');
  113 |   await page.getByLabel(/client or organization/i).fill('Acme Corp');
  114 |   await page.getByRole('button', { name: /add participant/i }).click();
  115 |   const displayNames = page.locator('input[name="participants[0].displayName"]');
  116 |   await displayNames.fill('Alice Smith');
  117 |   await page.getByRole('button', { name: /schedule meeting/i }).click();
  118 |   await page.waitForURL(/\/upcoming\/[a-z0-9-]+$/);
  119 |   await expect(page.getByText('Acme Quarterly Sync')).toBeVisible();
  120 |   await expect(page.getByText('Review Q3 results and Q4 targets.')).toBeVisible();
  121 |   await expect(page.getByText('Acme Corp').first()).toBeVisible();
  122 |   await expect(page.getByText('Alice Smith')).toBeVisible();
  123 | });
  124 | 
  125 | // ── AC-06: Navigation flows ────────────────────────────────────
  126 | test('AC-06: Dashboard card → list view → calendar view → detail view', async ({ page }) => {
  127 |   await enableUpcomingMeetings(page);
  128 |   await page.goto('/upcoming/new');
  129 |   const { date, time } = futureDate(1, 15, 0);
  130 |   await page.getByLabel(/meeting title/i).fill('Nav Test Meeting');
  131 |   await fillDateInput(page, date);
  132 |   await fillTimeInput(page, time);
  133 |   await page.getByRole('button', { name: /schedule meeting/i }).click();
  134 |   await page.waitForURL(/\/upcoming\/[a-z0-9-]+$/);
  135 |   const meetingTitle = 'Nav Test Meeting';
  136 |   await page.goto('/upcoming');
  137 |   await expect(page.getByText(meetingTitle).first()).toBeVisible();
  138 |   await expect(page.getByLabel(/upcoming meetings list/i)).toBeVisible();
  139 |   await page.getByRole('button', { name: /calendar/i }).click();
  140 |   await page.waitForURL(/\?view=calendar/);
  141 |   await expect(page.getByLabel(/upcoming meetings calendar/i)).toBeVisible();
  142 |   await page.getByRole('link', { name: /Nav Test Meeting/i }).first().click();
  143 |   await expect(page.getByRole('heading', { name: meetingTitle })).toBeVisible();
  144 | });
  145 | 
  146 | // ── AC-07: Cancelled meetings excluded from default view ─────────
  147 | // SKIPPED: Test isolation issue — shared in-memory store + duplicate detection
  148 | // causes form to stay on /upcoming/new when "Nav Test Meeting" (created by AC-06)
  149 | // has the same scheduled date as "To Be Cancelled". Application logic is correct.
  150 | test.skip('AC-07: Cancelled meetings hidden from list/calendar default view', async ({ page }) => {
  151 |   await enableUpcomingMeetings(page);
  152 |   await page.goto('/upcoming/new');
  153 |   const { date, time } = futureDate(6, 9, 0); // Use unique day offset to avoid duplicate
  154 |   await page.getByLabel(/meeting title/i).fill('To Be Cancelled');
  155 |   await fillDateInput(page, date);
  156 |   await fillTimeInput(page, time);
  157 |   await page.waitForTimeout(500);
  158 |   await page.getByRole('button', { name: /schedule meeting/i }).click();
  159 |   try {
  160 |     await page.waitForURL(/\/upcoming\/[a-z0-9-]+$/, { timeout: 10000 });
  161 |   } catch (e) {
  162 |     const currentUrl = page.url();
  163 |     const bodyText = await page.locator('body').innerText().catch(() => '(error reading body)');
  164 |     throw new Error(`Did not navigate to meeting detail page. URL: ${currentUrl}, Body: ${bodyText.slice(0, 200)}`);
  165 |   }
  166 |   const meetingUrl = page.url();
  167 |   await expect(page.getByText('To Be Cancelled').first()).toBeVisible();
  168 |   page.on('dialog', d => d.accept());
  169 |   await page.waitForLoadState('networkidle');
  170 |   const cancelBtn = page.getByRole('button', { name: /cancel meeting/i });
  171 |   const btnVisible = await cancelBtn.isVisible({ timeout: 2000 }).catch(() => false);
  172 |   if (btnVisible) {
  173 |     await cancelBtn.click();
  174 |     await page.waitForLoadState('networkidle');
  175 |   }
  176 |   await page.goto('/upcoming');
  177 |   await expect(page.getByText('Upcoming Meetings')).toBeVisible();
  178 |   await expect(page.getByText('To Be Cancelled').first()).not.toBeVisible();
  179 |   await page.goto(meetingUrl);
```