import { expect, test, type Page } from '@playwright/test';

const stream = 'recycling';
const instance = 'NIR-1';
const homePath = `/app/${stream}/i/${instance}`;

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

test('instance home renders act-now Worklist + chrome (Italian default)', async ({
	page
}) => {
	await stubOrgApis(page);
	await page.goto(homePath);
	await expect(page.locator('html')).toHaveAttribute('lang', 'it');
	await expect(page.getByTestId('org-shell')).toBeVisible();
	await expect(page.getByTestId('instance-chrome')).toBeVisible();
	await expect(page.getByTestId('instance-chrome-stream')).toHaveText('Riciclo');
	await expect(page.getByTestId('instance-chrome-label')).toHaveText('Camion A');
	await expect(page.getByTestId('instance-chrome-timeline')).toHaveText(
		'Torna alla timeline'
	);
	await expect(page.getByTestId('instance-chrome-timeline')).toHaveAttribute(
		'href',
		`/my/streams/org-${stream}/instance/${instance}`
	);
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
	await expect(page.getByTestId('nav-back')).toHaveCount(0);
	await expect(page.getByTestId('nav-role-chip')).toHaveText('administration');
	await expect(page.getByTestId('worklist-act-now')).toBeVisible();
	await expect(page.getByTestId('available-row')).toBeVisible();
	await expect(page.getByTestId('can-start-row')).toBeVisible();
	await expect(page.getByTestId('can-start-row')).toContainText('Apri lotto');
	await expect(page.getByTestId('can-start-row')).not.toContainText('Nuovo camion');
	const localeCookie = (await page.context().cookies()).find(
		(c) => c.name === 'PARAGLIDE_LOCALE'
	);
	expect(localeCookie?.value ?? 'it').toBe('it');
});

test('Home → Can start (each Job) → nav-back → Home', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(homePath);
	await expect(page.getByTestId('worklist-act-now')).toBeVisible();
	await page.getByTestId('can-start-row').click();
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-startLot')).toBeVisible();
	await expect(page.getByTestId('nav-back')).toBeVisible();
	await page.getByTestId('nav-back').click();
	await expect(page.getByTestId('org-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
	await expect(page.getByTestId('worklist-act-now')).toBeVisible();
});

test('Home → act-now task row → nav-back → Home', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(homePath);
	await expect(page.getByTestId('available-row')).toBeVisible();
	await page.getByTestId('available-row').click();
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-register')).toBeVisible();
	await expect(page.getByTestId('nav-back')).toBeVisible();
	await page.getByTestId('nav-back').click();
	await expect(page.getByTestId('org-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
	await expect(page.getByTestId('worklist-act-now')).toBeVisible();
});

test('unavailable Task shows blockedReason', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`${homePath}/intake/${instance}/palletWeight`);
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page.getByTestId('task-blocked-reason')).toHaveText(
		'Linea occupata: lotto L2 (camion T-B)'
	);
});

test('unauthorized Task URL redirects to instance home', async ({ page }) => {
	await stubOrgApis(page);
	// available (other role) → not unavailable → redirect home
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
	await expect(page.getByTestId('org-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
});

test('unknown Task instance redirects to instance home', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`${homePath}/intake/missing/register`);
	await expect(page.getByTestId('org-shell')).toBeVisible();
	await expect(page.getByTestId('nav-place-primary')).toHaveText('Home');
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
	await expect(page.getByTestId('play-chrome-dashboard')).toHaveAttribute(
		'href',
		`/my/streams/org-${stream}/`
	);
	await expect(page.getByTestId('task-register')).toBeVisible();
});

test('start Task seed.* is stripped from the URL', async ({ page }) => {
	await stubOrgApis(page);
	await page.goto(`${homePath}/intake/new/register?seed.lotId=L-wait&keep=1`);
	await expect(page.getByTestId('task-route')).toBeVisible();
	await expect(page).toHaveURL(
		new RegExp(`/app/${stream}/i/${instance}/intake/new/register\\?keep=1$`)
	);
});
