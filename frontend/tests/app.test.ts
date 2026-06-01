import { test, expect } from '@playwright/test';

test('homepage renders ContextPilot heading', async ({ page }) => {
	await page.goto('/');
	await expect(page.locator('h1')).toContainText('ContextPilot');
});