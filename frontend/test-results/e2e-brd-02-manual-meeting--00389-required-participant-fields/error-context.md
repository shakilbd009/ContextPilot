# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: e2e/brd-02-manual-meeting-import.spec.ts >> AC-07b: Save succeeds with only required participant fields
- Location: tests/e2e/brd-02-manual-meeting-import.spec.ts:267:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('Minimal Participant Meeting')
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for getByText('Minimal Participant Meeting')

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
  - heading "Import Meeting" [level=1]
  - paragraph: Record a completed meeting from memory
  - alert: An unexpected error occurred. Please try again.
  - text: Meeting title
  - textbox "Meeting title":
    - /placeholder: Q3 planning review
    - text: Minimal Participant Meeting
  - text: Completed date
  - textbox "Completed date":
    - /placeholder: ""
    - text: 2026-06-02
  - text: Completed time (optional)
  - textbox "Completed time (optional)":
    - /placeholder: ""
    - text: 11:00
  - group "Participants":
    - text: Participants Display name
    - textbox "Display name":
      - /placeholder: Alice Smith
      - text: Bob
    - button "Add participant"
    - button "Show advanced participant details"
  - group "Meeting content":
    - text: Meeting content
    - paragraph: At least one of transcript or notes is required.
    - text: Transcript
    - textbox "Transcript":
      - /placeholder: Paste or type the meeting transcript…
      - text: Meeting transcript for minimal test
    - text: Notes
    - textbox "Notes":
      - /placeholder: Or enter meeting notes…
    - text: 35 / 50,000 characters
  - button "Cancel"
  - button "Save meeting"
```

# Test source

```ts
  181 |     title: 'Valid Meeting Title',
  182 |     participantName: 'Alice',
  183 |     transcript: 'Valid transcript content here',
  184 |   });
  185 |   // Clear date
  186 |   const dateInput = page.locator('input[type="date"]');
  187 |   await dateInput.evaluate((el: HTMLInputElement) => { el.value = ''; });
  188 |   await submitForm(page);
  189 |   const errorMsg = page.getByText(/completed/i).first();
  190 |   await expect(errorMsg).toBeVisible();
  191 |   await expect(page).toHaveURL(/\/meetings\/new/);
  192 | });
  193 | 
  194 | // ── AC-05 ──────────────────────────────────────────────────────────
  195 | 
  196 | test('AC-05: Save blocked when no participant is present', async ({ page }) => {
  197 |   await signIn(page);
  198 |   await page.goto('/meetings/new');
  199 |   const { date, time } = futureDate(1, 10, 0);
  200 |   await fillForm(page, {
  201 |     title: 'Valid Meeting Title',
  202 |     date,
  203 |     time,
  204 |     transcript: 'Valid transcript content here',
  205 |   });
  206 |   // Remove the default participant
  207 |   const removeBtn = page.locator('.remove-participant-btn');
  208 |   if (await removeBtn.isVisible()) {
  209 |     await removeBtn.click();
  210 |     await page.waitForTimeout(300);
  211 |   }
  212 |   await submitForm(page);
  213 |   const errorMsg = page.getByText(/participant/i).first();
  214 |   await expect(errorMsg).toBeVisible();
  215 |   await expect(page).toHaveURL(/\/meetings\/new/);
  216 | });
  217 | 
  218 | // ── AC-06 ──────────────────────────────────────────────────────────
  219 | 
  220 | test('AC-06: Save blocked when participant display name is empty', async ({ page }) => {
  221 |   await signIn(page);
  222 |   await page.goto('/meetings/new');
  223 |   const { date, time } = futureDate(1, 10, 0);
  224 |   await fillForm(page, {
  225 |     title: 'Valid Meeting Title',
  226 |     date,
  227 |     time,
  228 |     transcript: 'Valid transcript content here',
  229 |     participantName: '',
  230 |   });
  231 |   await submitForm(page);
  232 |   const errorMsg = page.getByText(/display name/i).first();
  233 |   await expect(errorMsg).toBeVisible();
  234 |   await expect(page).toHaveURL(/\/meetings\/new/);
  235 | });
  236 | 
  237 | // ── AC-07 / AC-07b ────────────────────────────────────────────────
  238 | 
  239 | test('AC-07: Save succeeds with all optional participant fields populated', async ({ page }) => {
  240 |   await signIn(page);
  241 |   await page.goto('/meetings/new');
  242 |   const { date, time } = futureDate(1, 10, 0);
  243 |   await fillForm(page, {
  244 |     title: 'Full Participant Details Meeting',
  245 |     date,
  246 |     time,
  247 |     transcript: 'Meeting transcript content',
  248 |     participantName: 'Alice Smith',
  249 |   });
  250 |   // Expand advanced section only if not already visible
  251 |   const showAdvancedBtn = page.getByText(/show advanced/i);
  252 |   const hideAdvancedBtn = page.getByText(/hide advanced/i);
  253 |   const isAdvancedVisible = await hideAdvancedBtn.isVisible().catch(() => false);
  254 |   if (!isAdvancedVisible) {
  255 |     await showAdvancedBtn.click();
  256 |     await page.waitForTimeout(200);
  257 |   }
  258 |   await page.locator('input[name="participants[0].email"]').fill('alice@example.com');
  259 |   await page.locator('input[name="participants[0].organization"]').fill('Acme Corp');
  260 |   await page.locator('input[name="participants[0].role"]').fill('Engineer');
  261 |   await submitForm(page);
  262 |   await waitForRedirect(page, /\/meetings\/[a-z0-9-]+$/);
  263 |   await expect(page).not.toHaveURL(/\?/);
  264 |   await expect(page.getByText('Full Participant Details Meeting')).toBeVisible();
  265 | });
  266 | 
  267 | test('AC-07b: Save succeeds with only required participant fields', async ({ page }) => {
  268 |   await signIn(page);
  269 |   await page.goto('/meetings/new');
  270 |   const { date, time } = futureDate(1, 11, 0);
  271 |   await fillForm(page, {
  272 |     title: 'Minimal Participant Meeting',
  273 |     date,
  274 |     time,
  275 |     transcript: 'Meeting transcript for minimal test',
  276 |     participantName: 'Bob',
  277 |   });
  278 |   await submitForm(page);
  279 |   await waitForRedirect(page, /\/meetings\/[a-z0-9-]+$/);
  280 |   await expect(page).not.toHaveURL(/\?/);
> 281 |   await expect(page.getByText('Minimal Participant Meeting')).toBeVisible();
      |                                                               ^ Error: expect(locator).toBeVisible() failed
  282 | });
  283 | 
  284 | // ── AC-08 ──────────────────────────────────────────────────────────
  285 | 
  286 | test('AC-08: Save blocked when both transcript and notes are empty', async ({ page }) => {
  287 |   await signIn(page);
  288 |   await page.goto('/meetings/new');
  289 |   const { date, time } = futureDate(1, 10, 0);
  290 |   await fillForm(page, {
  291 |     title: 'Meeting Without Content',
  292 |     date,
  293 |     time,
  294 |     participantName: 'Alice',
  295 |     transcript: '',
  296 |     notes: '',
  297 |   });
  298 |   await submitForm(page);
  299 |   // Should see error about content requirement
  300 |   const errorMsg = page.getByText(/transcript|notes|content/i).first();
  301 |   await expect(errorMsg).toBeVisible();
  302 |   await expect(page).toHaveURL(/\/meetings\/new/);
  303 | });
  304 | 
  305 | // ── AC-09 ──────────────────────────────────────────────────────────
  306 | 
  307 | test('AC-09: Save blocked when combined transcript+notes exceeds 50,000 chars', async ({ page }) => {
  308 |   await signIn(page);
  309 |   await page.goto('/meetings/new');
  310 |   const { date, time } = futureDate(1, 10, 0);
  311 |   const largeTranscript = generateLargeText(50_001);
  312 |   await fillForm(page, {
  313 |     title: 'Large Content Meeting',
  314 |     date,
  315 |     time,
  316 |     participantName: 'Alice',
  317 |     transcript: largeTranscript,
  318 |     notes: '',
  319 |   });
  320 |   await submitForm(page);
  321 |   // Should show error about exceeding limit
  322 |   const errorMsg = page.getByText(/50,000|limit|exceeded/i).first();
  323 |   await expect(errorMsg).toBeVisible();
  324 |   await expect(page).toHaveURL(/\/meetings\/new/);
  325 | });
  326 | 
  327 | // ── AC-10 ──────────────────────────────────────────────────────────
  328 | 
  329 | test('AC-10: Happy path save with transcript', async ({ page }) => {
  330 |   await signIn(page);
  331 |   await page.goto('/meetings/new');
  332 |   const { date, time } = futureDate(2, 14, 30);
  333 |   await fillForm(page, {
  334 |     title: 'Q3 Planning Session',
  335 |     date,
  336 |     time,
  337 |     transcript: 'The team discussed quarterly goals and resource allocation.',
  338 |     participantName: 'Alice',
  339 |   });
  340 |   await submitForm(page);
  341 |   await waitForRedirect(page, /\/meetings\/[a-z0-9-]+$/);
  342 |   // No query string
  343 |   await expect(page).not.toHaveURL(/\?/);
  344 |   await expect(page.getByText('Q3 Planning Session')).toBeVisible();
  345 |   // Also visible in meetings list
  346 |   await page.goto('/meetings');
  347 |   await expect(page.getByText('Q3 Planning Session')).toBeVisible();
  348 | });
  349 | 
  350 | // ── AC-11 ──────────────────────────────────────────────────────────
  351 | 
  352 | test('AC-11: Save succeeds with notes only (no transcript)', async ({ page }) => {
  353 |   await signIn(page);
  354 |   await page.goto('/meetings/new');
  355 |   const { date, time } = futureDate(3, 15, 0);
  356 |   await fillForm(page, {
  357 |     title: 'Notes Only Meeting',
  358 |     date,
  359 |     time,
  360 |     transcript: '',
  361 |     notes: 'Discussed Q3 priorities and resource allocation.',
  362 |     participantName: 'Bob',
  363 |   });
  364 |   await submitForm(page);
  365 |   await waitForRedirect(page, /\/meetings\/[a-z0-9-]+$/);
  366 |   await expect(page).not.toHaveURL(/\?/);
  367 |   await expect(page.getByText('Notes Only Meeting')).toBeVisible();
  368 |   await page.goto('/meetings');
  369 |   await expect(page.getByText('Notes Only Meeting')).toBeVisible();
  370 | });
  371 | 
  372 | // ── AC-14 ──────────────────────────────────────────────────────────
  373 | 
  374 | test('AC-14: Redirect has no query params', async ({ page }) => {
  375 |   await signIn(page);
  376 |   await page.goto('/meetings/new');
  377 |   const { date, time } = futureDate(4, 9, 0);
  378 |   await fillForm(page, {
  379 |     title: 'No Query Params Meeting',
  380 |     date,
  381 |     time,
```