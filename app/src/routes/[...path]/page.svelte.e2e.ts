import { expect, test } from '@playwright/test';

test('home route renders Worklist Home bands', async ({ page }) => {
	await page.goto('/app');
	await expect(page.getByTestId('living-shell')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Home' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Home' })).toBeVisible();
	await expect(page.getByTestId('living-reload')).toBeVisible();
});

test('Home → Task → back', async ({ page }) => {
	await page.goto('/app/intake/new/register');
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-back-home')).toBeVisible();
	await page.getByTestId('task-back-home').click();
	await expect(page.getByTestId('living-shell')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Home' })).toBeVisible();
});
