import { describe, expect, test, vi } from 'vitest';

import type { TaskFormBundle } from './form/task-form-bundle.js';
import { createTaskFormMount } from './task-form-mount.svelte.js';
import type { TaskFormPort } from './task-form-port.js';
import type { Worklist } from './types.js';

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
				instances: [
					{
						id: 'NIR-advance',
						label: 'Advance',
						tasks: {
							register: 'done',
							codiceFir: 'actionable',
							palletWeight: 'unavailable'
						}
					}
				]
			}
		]
	};
}

describe('TaskFormMount', () => {
	test('gate deny exposes redirect and skips fetch', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const blocked: Worklist = {
			jobs: [{ ...adminWorklist().jobs[0]!, canStart: false }]
		};

		const result = await mount.load(
			{
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: '',
				provisionalJob: true
			},
			blocked
		);

		expect(result).toEqual({ kind: 'denied', redirect: { kind: 'home' } });
		expect(mount.status).toBe('denied');
		expect(mount.redirect).toEqual({ kind: 'home' });
		expect(mount.bundle).toBeNull();
		expect(mount.addressKey).toBe('intake|new|register');
		expect(fetch).not.toHaveBeenCalled();
	});

	test('allow fetches via TaskFormPort and sets ready bundle', async () => {
		const fetch = vi.fn(async (req) => {
			expect(req).toEqual({
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: 'new'
			});
			return sample;
		});
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });

		const result = await mount.load(
			{
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: '',
				provisionalJob: true
			},
			adminWorklist()
		);

		expect(result).toEqual({ kind: 'ready' });
		expect(mount.status).toBe('ready');
		expect(mount.bundle).toEqual(sample);
		expect(mount.error).toBeNull();
		expect(mount.redirect).toBeNull();
		expect(mount.addressKey).toBe('intake|new|register');
		expect(fetch).toHaveBeenCalledOnce();
	});

	test('overlapping loads ignore stale generation', async () => {
		let releaseSlow: (bundle: TaskFormBundle) => void = () => {};
		const slow = new Promise<TaskFormBundle>((resolve) => {
			releaseSlow = resolve;
		});

		const fetch = vi
			.fn<TaskFormPort['fetch']>()
			.mockImplementationOnce(() => slow)
			.mockImplementationOnce(async () => ({
				...sample,
				taskId: 'codiceFir',
				jobInstanceId: 'NIR-advance',
				title: 'Codice FIR'
			}));

		const mount = createTaskFormMount({ taskForm: { fetch } });

		const first = mount.load(
			{
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: '',
				provisionalJob: true
			},
			adminWorklist()
		);

		const second = await mount.load(
			{
				jobId: 'intake',
				taskId: 'codiceFir',
				jobInstanceId: 'NIR-advance',
				provisionalJob: false
			},
			adminWorklist()
		);

		expect(second).toEqual({ kind: 'ready' });
		expect(mount.bundle?.taskId).toBe('codiceFir');
		expect(mount.addressKey).toBe('intake|NIR-advance|codiceFir');

		releaseSlow({ ...sample, title: 'stale-should-not-win' });
		const firstResult = await first;

		expect(firstResult).toEqual({ kind: 'aborted' });
		expect(mount.bundle?.taskId).toBe('codiceFir');
		expect(mount.bundle?.title).toBe('Codice FIR');
		expect(mount.status).toBe('ready');
		expect(fetch).toHaveBeenCalledTimes(2);
	});

	test('reload re-fetches last address and accepts fresh worklist', async () => {
		const actionable = {
			...sample,
			taskId: 'codiceFir',
			jobInstanceId: 'NIR-advance',
			title: 'Codice FIR'
		};
		const fetch = vi
			.fn<TaskFormPort['fetch']>()
			.mockResolvedValueOnce(actionable)
			.mockResolvedValueOnce({ ...actionable, title: 'Codice FIR again' });
		const mount = createTaskFormMount({ taskForm: { fetch } });

		await mount.load(
			{
				jobId: 'intake',
				taskId: 'codiceFir',
				jobInstanceId: 'NIR-advance',
				provisionalJob: false
			},
			adminWorklist()
		);

		const result = await mount.reload(adminWorklist());

		expect(result).toEqual({ kind: 'ready' });
		expect(mount.bundle?.title).toBe('Codice FIR again');
		expect(mount.addressKey).toBe('intake|NIR-advance|codiceFir');
		expect(fetch).toHaveBeenCalledTimes(2);
		expect(fetch).toHaveBeenLastCalledWith({
			jobId: 'intake',
			taskId: 'codiceFir',
			jobInstanceId: 'NIR-advance'
		});
	});
});
