import { expect, test } from '@playwright/test';

test('home route renders living Jobs shell', async ({ page }) => {
	await page.goto('/app');
	await expect(page.getByTestId('living-shell')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Jobs' })).toBeVisible();
	await expect(page.getByTestId('living-reload')).toBeVisible();
});

test('about route renders via basename', async ({ page }) => {
	await page.goto('/app/about');
	await expect(page.getByRole('heading', { name: 'About' })).toBeVisible();
});
