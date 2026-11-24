import { describe, expect, test, vi } from 'vitest';

import { runTaskFormBeforeLoad } from './task-form-before-load.js';
import { createTaskFormMount } from './task-form-mount.svelte.js';
import type { TaskFormPort } from './task-form-port.js';
import type { Worklist } from './types.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';

const sample: TaskFormBundle = {
	jobId: 'intake',
	taskId: 'register',
	jobInstanceId: '',
	title: 'Register',
	surface: 'live',
	schema: { type: 'object' },
	uiSchema: {},
	initialData: {},
	chrome: [],
	chromeFieldIds: [],
	sealFieldIds: [],
	bindings: { domain: {} }
};

const stream = 'recycling';
const truck = '261102-1';
const taskPath = `/${stream}/i/${truck}/intake/new/register`;
const homeRedirect = { kind: 'home' as const, stream, instance: truck };

function adminWorklist(): Worklist {
	return {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				startLabel: 'Nuovo camion',
				taskIds: ['register', 'codiceFir', 'palletWeight'],
				tasks: [
					{ id: 'register', title: 'Registrazione', roles: ['administration'] },
					{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
					{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
				],
				instances: []
			}
		]
	};
}

function sessionStub(worklist: Worklist | null) {
	const drafts = new Map<string, Record<string, unknown>>();
	const seededMeta = new Map<string, readonly string[]>();
	let openSeedByPath = new Map<string, Record<string, string>>();
	return {
		result: worklist ? { worklist } : null,
		getDraft: (path: string, key: string) => drafts.get(`${path}::${key}`) ?? null,
		seedDraft: (
			path: string,
			key: string,
			value: Record<string, unknown>,
			seededFields?: readonly string[]
		) => {
			drafts.set(`${path}::${key}`, value);
			if (seededFields) seededMeta.set(`${path}::${key}`, seededFields);
		},
		rememberOpenSeed: (path: string, seed: Record<string, string>) => {
			openSeedByPath.set(path, seed);
		},
		clearOpenSeed: (path: string) => {
			openSeedByPath.delete(path);
		},
		drafts,
		seededMeta,
		get openSeedByPath() {
			return openSeedByPath;
		}
	};
}

describe('runTaskFormBeforeLoad', () => {
	test('loads mount for allowed provisional address when Worklist present', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const navigateRedirect = vi.fn();

		await runTaskFormBeforeLoad(taskPath, {
			mount,
			session: sessionStub(adminWorklist()),
			navigateRedirect
		});

		expect(mount.status).toBe('ready');
		expect(mount.bundle).toEqual(sample);
		expect(navigateRedirect).not.toHaveBeenCalled();
		expect(fetch).toHaveBeenCalledOnce();
	});

	test('throws navigateRedirect on gate deny without fetching', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const deniedNav = Promise.resolve('nav');
		const navigateRedirect = vi.fn(() => deniedNav);
		const blocked: Worklist = {
			jobs: [{ ...adminWorklist().jobs[0]!, canStart: false }]
		};

		await expect(
			runTaskFormBeforeLoad(taskPath, {
				mount,
				session: sessionStub(blocked),
				navigateRedirect
			})
		).rejects.toBe(deniedNav);

		expect(mount.status).toBe('denied');
		expect(mount.redirect).toEqual(homeRedirect);
		expect(navigateRedirect).toHaveBeenCalledWith(homeRedirect);
		expect(fetch).not.toHaveBeenCalled();
	});

	test('starts fetch and returns when Worklist is not ready yet', async () => {
		let release: (bundle: TaskFormBundle) => void = () => {};
		const pendingFetch = new Promise<TaskFormBundle>((resolve) => {
			release = resolve;
		});
		const fetch = vi.fn(() => pendingFetch);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const navigateRedirect = vi.fn();

		await runTaskFormBeforeLoad(taskPath, {
			mount,
			session: sessionStub(null),
			navigateRedirect
		});

		expect(fetch).toHaveBeenCalledOnce();
		expect(mount.status).toBe('loading');
		expect(navigateRedirect).not.toHaveBeenCalled();

		release(sample);
		await vi.waitFor(() => expect(mount.status).toBe('ready'));
		expect(mount.bundle).toEqual(sample);
	});

	test('applies ?seed.* into isolated draft bag, records provenance, strips URL', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const session = sessionStub(adminWorklist());
		const replaceHref = vi.fn();
		const href = `https://plant.example/app${taskPath}?seed.lotId=L-wait&keep=1`;

		await runTaskFormBeforeLoad(taskPath, {
			mount,
			session,
			navigateRedirect: vi.fn(),
			href,
			replaceHref
		});

		const seedKey = 'seed:lotId=L-wait';
		expect(session.drafts.get(`intake/register::${seedKey}`)).toEqual({ lotId: 'L-wait' });
		expect(session.seededMeta.get(`intake/register::${seedKey}`)).toEqual(['lotId']);
		expect(session.openSeedByPath.get('intake/register')).toEqual({ lotId: 'L-wait' });
		expect(replaceHref).toHaveBeenCalledWith(`/app${taskPath}?keep=1`);
		expect(mount.status).toBe('ready');
	});

	test('unseeded provisional open clears prior open-seed lock pointer', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const session = sessionStub(adminWorklist());
		session.rememberOpenSeed('intake/register', { lotId: 'L-wait' });

		await runTaskFormBeforeLoad(taskPath, {
			mount,
			session,
			navigateRedirect: vi.fn(),
			href: `https://plant.example/app${taskPath}`,
			replaceHref: vi.fn()
		});

		expect(session.openSeedByPath.has('intake/register')).toBe(false);
		expect(mount.status).toBe('ready');
	});

	test('unavailable leaf with blockedReason stays without fetching', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const wl: Worklist = {
			jobs: [
				{
					...adminWorklist().jobs[0]!,
					canStart: false,
					instances: [
						{
							id: 'L1',
							label: 'L1',
							tasks: { register: 'done', codiceFir: 'done', palletWeight: 'unavailable' },
							blockedReasons: { palletWeight: 'Linea occupata' }
						}
					]
				}
			]
		};
		const path = `/${stream}/i/${truck}/intake/L1/palletWeight`;

		await runTaskFormBeforeLoad(path, {
			mount,
			session: sessionStub(wl),
			navigateRedirect: vi.fn()
		});

		expect(mount.status).toBe('blocked');
		expect(mount.blockedReason).toBe('Linea occupata');
		expect(fetch).not.toHaveBeenCalled();
	});
});
