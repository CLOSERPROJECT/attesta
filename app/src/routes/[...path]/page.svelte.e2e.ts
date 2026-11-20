import { expect, test, type Page } from '@playwright/test';

const stream = 'recycling';
const instance = 'NIR-1';
const homePath = `/app/${stream}/i/${instance}`;

const timelineStub = {
	stream,
	instance,
	label: 'Camion A',
	status: 'active',
	steps: [
		{
			stepId: '1',
			title: 'Ingresso',
			substepCount: 2,
			expanded: true,
			substeps: [
				{
					substepId: 'intake.register',
					title: 'Registrazione',
					status: 'available',
					statusLabel: 'available',
					orgActionable: true,
					href: `${homePath}/intake/${instance}/register`
				},
				{
					substepId: 'intake.palletWeight',
					title: 'Peso pallet',
					status: 'locked',
					statusLabel: 'locked',
					blockedReason: 'Linea occupata: lotto L2 (camion T-B)'
				}
			]
		}
	]
};

const homeLoad = {
	org: 'feragame',
	activeConfigVersion: '1',
	configId: 'feragame-config',
	userId: 'u1',
	displayName: 'Ada',
	roles: ['administration'],
	worklist: {
		jobs: [
			{
				jobId: 'intake',
				// Instance-scoped wire suppresses anchor canStart (#361).
				canStart: false,
				repeatable: true,
				title: 'Ingresso',
				startLabel: 'Nuovo camion',
				taskIds: ['register', 'palletWeight'],
				tasks: [
					{ id: 'register', title: 'Registrazione', roles: ['administration'] },
					{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
				],
				instances: [
					{
						id: instance,
						label: 'Camion A',
						tasks: { register: 'actionable', palletWeight: 'unavailable' },
						blockedReasons: {
							palletWeight: 'Linea occupata: lotto L2 (camion T-B)'
						}
					}
				]
			},
			{
				jobId: 'lot_processing',
				canStart: true,
				repeatable: true,
				title: 'Lavorazione',
				startLabel: 'Apri lotto',
				taskIds: ['startLot'],
				tasks: [{ id: 'startLot', title: 'Apri lotto', roles: ['logistic'] }],
				instances: []
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

async function stubOrgApis(page: Page) {
	await page.route('**/api/org/streams', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify([
				{
					key: stream,
					title: 'Riciclo',
					play: { jobId: 'intake', taskId: 'register' }
				}
			])
		});
	});
	await page.route('**/api/org/domain**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(homeLoad)
		});
	});
	await page.route('**/api/org/timeline**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(timelineStub)
		});
	});
	await page.route('**/api/org/task-form**', async (route) => {
		const url = new URL(route.request().url());
		const jobId = url.searchParams.get('jobId') ?? 'intake';
		const taskId = url.searchParams.get('taskId') ?? 'register';
		const jobInstanceId = url.searchParams.get('jobInstanceId') ?? '';
		const provisional = jobInstanceId === '' || jobInstanceId === 'new';
		const rolesOk =
			taskId === 'register' ||
			taskId === 'startLot' ||
			(taskId === 'palletWeight' && homeLoad.roles.includes('logistic'));
		if (!provisional && !rolesOk && taskId === 'palletWeight') {
			await route.fulfill({ status: 403, body: 'forbidden' });
			return;
		}
		const title =
			taskId === 'register'
				? 'Registrazione'
				: taskId === 'startLot'
					? 'Apri lotto'
					: taskId;
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				...registerBundle,
				jobId,
				taskId,
				jobInstanceId: provisional ? '' : jobInstanceId,
				title
			})
		});
	});
}

test('instance route renders SPA timeline home', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(homePath);
	await expect(page).toHaveURL(homePath);
	await expect(page.getByTestId('org-timeline')).toBeVisible();
	await expect(page.getByTestId('instance-chrome-label')).toHaveText('Camion A');
	await expect(page.locator('.js-show-all-filter')).toBeVisible();
	await expect(page.locator('.substep-card-link')).toBeVisible();
});

test('Can start (each Job) → nav-back → timeline', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`${homePath}/lot_processing/new/startLot`);
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-startLot')).toBeVisible();
	await expect(page.getByTestId('nav-back')).toBeVisible();
	await page.getByTestId('nav-back').click();
	await expect(page).toHaveURL(homePath);
	await expect(page.getByTestId('org-timeline')).toBeVisible();
});

test('act-now task → nav-back → timeline', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`${homePath}/intake/${instance}/register`);
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-register')).toBeVisible();
	await expect(page.getByTestId('nav-back')).toBeVisible();
	await page.getByTestId('nav-back').click();
	await expect(page).toHaveURL(homePath);
	await expect(page.getByTestId('org-timeline')).toBeVisible();
});

test('unavailable Task shows blockedReason', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`${homePath}/intake/${instance}/palletWeight`);
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-blocked-reason')).toHaveText(
		'Linea occupata: lotto L2 (camion T-B)'
	);
});

test('unauthorized Task URL redirects to instance timeline', async ({ page }) => {
	await stubOrgApis(page);
	// available (other role) → not unavailable → redirect home → SPA timeline
	const loadOtherRole = {
		...homeLoad,
		worklist: {
			jobs: [
				{
					...homeLoad.worklist.jobs[0]!,
					instances: [
						{
							id: instance,
							label: 'Camion A',
							tasks: { register: 'done', palletWeight: 'available' }
						}
					]
				}
			]
		}
	};
	await page.unroute('**/api/org/domain**');
	await page.route('**/api/org/domain**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(loadOtherRole)
		});
	});
	await page.goto(`${homePath}/intake/${instance}/palletWeight`);
	await expect(page).toHaveURL(homePath);
	await expect(page.getByTestId('org-timeline')).toBeVisible();
});

test('unknown Task instance redirects to instance timeline', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`${homePath}/intake/missing/register`);
	await expect(page).toHaveURL(homePath);
	await expect(page.getByTestId('org-timeline')).toBeVisible();
});

test('Play route shows stream chrome', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`/app/${stream}/new`);
	await expect(page.getByTestId('play-route')).toBeVisible();
	await expect(page.getByTestId('play-chrome')).toBeVisible();
	await expect(page.getByTestId('play-chrome-stream')).toHaveText('Riciclo');
	await expect(page.getByTestId('play-chrome-dashboard')).toHaveText(
		'Torna alla dashboard'
	);
	const dashboard = page.getByTestId('play-chrome-dashboard');
	await expect(dashboard).toHaveAttribute(
		'href',
		`/my/streams/org-${stream}/`
	);
	await expect(dashboard).toHaveAttribute('rel', 'external');
	await expect(dashboard).toHaveAttribute('target', '_self');
	await expect(page.getByTestId('task-register')).toBeVisible();
});

test('start Task seed.* is stripped from the URL', async ({ page }) => {
	await stubOrgApis(page);
	// Provisional start needs Worklist canStart (intake is suppressed on instance scope).
	await page.goto(
		`${homePath}/lot_processing/new/startLot?seed.lotId=L-wait&keep=1`
	);
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page).toHaveURL(
		new RegExp(
			`/app/${stream}/i/${instance}/lot_processing/new/startLot\\?keep=1$`
		)
	);
});
