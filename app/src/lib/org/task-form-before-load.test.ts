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

describe('runTaskFormBeforeLoad', () => {
	test('loads mount for allowed provisional address when Worklist present', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const navigateRedirect = vi.fn();

		await runTaskFormBeforeLoad('/intake/new/register', {
			mount,
			session: { result: { worklist: adminWorklist() } },
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
			runTaskFormBeforeLoad('/intake/new/register', {
				mount,
				session: { result: { worklist: blocked } },
				navigateRedirect
			})
		).rejects.toBe(deniedNav);

		expect(mount.status).toBe('denied');
		expect(mount.redirect).toEqual({ kind: 'home' });
		expect(navigateRedirect).toHaveBeenCalledWith({ kind: 'home' });
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

		await runTaskFormBeforeLoad('/intake/new/register', {
			mount,
			session: { result: null },
			navigateRedirect
		});

		expect(fetch).toHaveBeenCalledOnce();
		expect(mount.status).toBe('loading');
		expect(navigateRedirect).not.toHaveBeenCalled();

		release(sample);
		await vi.waitFor(() => expect(mount.status).toBe('ready'));
		expect(mount.bundle).toEqual(sample);
	});
});
