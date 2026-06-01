import { test, expect } from '@playwright/test';

test('homepage renders the landing hero heading', async ({ page }) => {
	await page.goto('/');
	await expect(page.locator('h1#landing-heading')).toContainText(
		'Meeting context, when you need it most',
	);
});