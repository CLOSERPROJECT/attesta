import { expect, test } from '@playwright/test';

test('home route renders', async ({ page }) => {
	await page.goto('/app');
	await expect(page.getByRole('heading', { name: 'Attesta app' })).toBeVisible();
});

test('about route renders via basename', async ({ page }) => {
	await page.goto('/app/about');
	await expect(page.getByRole('heading', { name: 'About' })).toBeVisible();
});
