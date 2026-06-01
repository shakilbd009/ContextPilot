# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: app.test.ts >> homepage renders ContextPilot heading
- Location: tests/app.test.ts:3:1

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: locator('h1')
Expected substring: "ContextPilot"
Received string:    "Meeting context, when you need it most"
Timeout: 5000ms

Call log:
  - Expect "toContainText" with timeout 5000ms
  - waiting for locator('h1')
    9 × locator resolved to <h1 id="landing-heading" class="landing__title svelte-1uha8ag">Meeting context, when you need it most</h1>
      - unexpected value "Meeting context, when you need it most"

```

```yaml
- heading "Meeting context, when you need it most" [level=1]
```

# Test source

```ts
  1 | import { test, expect } from '@playwright/test';
  2 | 
  3 | test('homepage renders ContextPilot heading', async ({ page }) => {
  4 | 	await page.goto('/');
> 5 | 	await expect(page.locator('h1')).toContainText('ContextPilot');
    |                                   ^ Error: expect(locator).toContainText(expected) failed
  6 | });
```