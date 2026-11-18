import { describe, expect, test, vi } from 'vitest';

import {
	awaitOrgSessionWorklist,
	runTaskFormBeforeLoad
} from './task-form-before-load.js';
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

describe('awaitOrgSessionWorklist', () => {
	test('returns existing worklist without waiting', async () => {
		const worklist = adminWorklist();
		const session = { result: { worklist }, loading: false };
		await expect(awaitOrgSessionWorklist(session, { settleMs: 0 })).resolves.toBe(
			worklist
		);
	});

	test('awaits until loading flips false', async () => {
		const worklist = adminWorklist();
		const session = {
			result: null as { worklist: Worklist } | null,
			loading: true
		};

		const pending = awaitOrgSessionWorklist(session, { pollMs: 5, maxMs: 1000, settleMs: 0 });
		await new Promise((r) => setTimeout(r, 15));
		session.result = { worklist };
		session.loading = false;

		await expect(pending).resolves.toBe(worklist);
	});

	test('settles briefly then proceeds with null when hydrate never starts', async () => {
		const session = { result: null, loading: false };
		await expect(
			awaitOrgSessionWorklist(session, { settleMs: 20, pollMs: 5, maxMs: 50 })
		).resolves.toBeUndefined();
	});
});

describe('runTaskFormBeforeLoad', () => {
	test('loads mount for allowed provisional address', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const navigateRedirect = vi.fn();

		await runTaskFormBeforeLoad('/intake/new/register', {
			mount,
			session: { result: { worklist: adminWorklist() }, loading: false },
			navigateRedirect,
			awaitWorklist: async () => adminWorklist()
		});

		expect(mount.status).toBe('ready');
		expect(mount.bundle).toEqual(sample);
		expect(navigateRedirect).not.toHaveBeenCalled();
		expect(fetch).toHaveBeenCalledOnce();
	});

	test('throws navigateRedirect on gate deny', async () => {
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
				session: { result: { worklist: blocked }, loading: false },
				navigateRedirect,
				awaitWorklist: async () => blocked
			})
		).rejects.toBe(deniedNav);

		expect(mount.status).toBe('denied');
		expect(mount.redirect).toEqual({ kind: 'home' });
		expect(navigateRedirect).toHaveBeenCalledWith({ kind: 'home' });
		expect(fetch).not.toHaveBeenCalled();
	});
});
