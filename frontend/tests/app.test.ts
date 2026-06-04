import { test, expect } from '@playwright/test';

test('homepage renders the expected root heading for the active app-shell mode', async ({ page }) => {
	await page.goto('/');
	await expect(page.locator('h1').first()).toHaveText(
		/^(Meeting context, when you need it most|Dashboard)$/,
	);
});