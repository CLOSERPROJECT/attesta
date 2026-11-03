import { expect, test, type Page } from '@playwright/test';

const homeLoad = {
	org: 'feragame',
	activeConfigVersion: '1',
	configId: 'feragame-config',
	roles: ['administration'],
	worklist: {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				startLabel: 'Nuovo camion',
				taskIds: ['register', 'palletWeight'],
				tasks: [
					{ id: 'register', title: 'Registrazione', roles: ['administration'] },
					{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
				],
				instances: [
					{
						id: 'NIR-1',
						label: 'NIR-1',
						tasks: { register: 'done', palletWeight: 'available' },
						viewerAvailable: []
					},
					{
						id: 'NIR-2',
						label: 'Camion B',
						tasks: { register: 'available', palletWeight: 'locked' },
						viewerAvailable: ['register']
					},
					{
						id: 'NIR-done',
						label: 'Finished',
						tasks: { register: 'done', palletWeight: 'done' },
						viewerAvailable: []
					}
				]
			}
		]
	},
	taskFormBundles: []
};

const registerBundle = {
	jobId: 'intake',
	taskId: 'register',
	jobInstanceId: '',
	title: 'Registrazione',
	surface: 'live',
	schema: { type: 'object', properties: {} },
	uiSchema: {},
	initialData: {},
	chrome: [],
	chromeFieldIds: [],
	sealFieldIds: [],
	bindings: { domain: { submissions: [], jobInstances: [] } }
};

async function stubLivingApis(page: Page) {
	await page.route('**/api/living/domain', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(homeLoad)
		});
	});
	await page.route('**/api/living/task-form**', async (route) => {
		const url = new URL(route.request().url());
		const taskId = url.searchParams.get('taskId') ?? 'register';
		const jobInstanceId = url.searchParams.get('jobInstanceId') ?? '';
		const provisional = jobInstanceId === '' || jobInstanceId === 'new';
		const rolesOk =
			taskId === 'register' ||
			(taskId === 'palletWeight' && homeLoad.roles.includes('logistic'));
		if (!provisional && !rolesOk && taskId === 'palletWeight') {
			await route.fulfill({ status: 403, body: 'forbidden' });
			return;
		}
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				...registerBundle,
				taskId,
				jobInstanceId: provisional ? '' : jobInstanceId,
				title: taskId === 'register' ? 'Registrazione' : taskId
			})
		});
	});
}

test('home route renders Worklist Home bands', async ({ page }) => {
	await stubLivingApis(page);
	await page.goto('/app');
	await expect(page.getByTestId('living-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
	await expect(page.getByTestId('nav-back')).toHaveCount(0);
	await expect(page.getByTestId('nav-role-chip')).toHaveText('administration');
	await expect(page.getByTestId('worklist-available')).toBeVisible();
	await expect(page.getByTestId('worklist-can-start')).toBeVisible();
	await expect(page.getByTestId('available-row')).toBeVisible();
	await expect(page.getByTestId('can-start-row')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Available' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Can start' })).toBeVisible();
});

test('Home → Can start → nav-back → Home', async ({ page }) => {
	await stubLivingApis(page);
	await page.goto('/app');
	await expect(page.getByTestId('worklist-can-start')).toBeVisible();
	await page.getByTestId('can-start-row').click();
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-register')).toBeVisible();
	await expect(page.getByTestId('nav-back')).toBeVisible();
	await page.getByTestId('nav-back').click();
	await expect(page.getByTestId('living-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
	await expect(page.getByTestId('worklist-available')).toBeVisible();
	await expect(page.getByTestId('worklist-can-start')).toBeVisible();
});

test('Home → Available row → nav-back → overview', async ({ page }) => {
	await stubLivingApis(page);
	await page.goto('/app');
	await expect(page.getByTestId('available-row')).toBeVisible();
	await page.getByTestId('available-row').click();
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-register')).toBeVisible();
	await expect(page.getByTestId('nav-back')).toBeVisible();
	await page.getByTestId('nav-back').click();
	await expect(page.getByTestId('overview-route')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Camion B');
	await expect(page.getByTestId('overview-tasks')).toBeVisible();
});

test('overview shows handoff copy for other-role available Task', async ({ page }) => {
	await stubLivingApis(page);
	await page.goto('/app/intake/NIR-1');
	await expect(page.getByTestId('overview-route')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('NIR-1');
	await expect(page.getByTestId('overview-handoff')).toContainText('logistic');
	await expect(page.getByTestId('overview-task-status')).toHaveCount(2);
	await expect(page.getByTestId('overview-task-link')).toHaveCount(0);
});

test('unauthorized Task URL redirects to overview', async ({ page }) => {
	await stubLivingApis(page);
	await page.goto('/app/intake/NIR-1/palletWeight');
	await expect(page.getByTestId('overview-route')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('NIR-1');
});

test('unknown Task instance redirects Home', async ({ page }) => {
	await stubLivingApis(page);
	await page.goto('/app/intake/missing/register');
	await expect(page.getByTestId('living-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
});

test('finished overview URL redirects Home', async ({ page }) => {
	await stubLivingApis(page);
	await page.goto('/app/intake/NIR-done');
	await expect(page.getByTestId('living-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
});
