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

const stream = 'recycling';
const truck = '261102-1';
const homeRedirect = { kind: 'home' as const, stream, instance: truck };

const provisionalAddr = {
	stream,
	instance: truck,
	jobId: 'intake',
	taskId: 'register',
	jobInstanceId: '',
	provisionalJob: true
} as const;

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
	test('applyGate deny binds address, exposes redirect, and skips fetch', () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });
		const blocked: Worklist = {
			jobs: [{ ...adminWorklist().jobs[0]!, canStart: false }]
		};

		const result = mount.applyGate(blocked, provisionalAddr);

		expect(result).toEqual({ kind: 'denied', redirect: homeRedirect });
		expect(mount.status).toBe('denied');
		expect(mount.redirect).toEqual(homeRedirect);
		expect(mount.bundle).toBeNull();
		expect(mount.addressKey).toBe('recycling|261102-1|intake|new|register');
		expect(fetch).not.toHaveBeenCalled();
	});

	test('load fetches via TaskFormPort without worklist and sets ready bundle', async () => {
		const fetch = vi.fn(async (req) => {
			expect(req).toEqual({
				stream,
				instance: truck,
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: 'new'
			});
			return sample;
		});
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });

		const result = await mount.load(provisionalAddr);

		expect(result).toEqual({ kind: 'ready' });
		expect(mount.status).toBe('ready');
		expect(mount.bundle).toEqual(sample);
		expect(mount.error).toBeNull();
		expect(mount.redirect).toBeNull();
		expect(mount.addressKey).toBe('recycling|261102-1|intake|new|register');
		expect(fetch).toHaveBeenCalledOnce();
	});

	test('applyGate allow after load does not re-fetch', async () => {
		const fetch = vi.fn(async () => sample);
		const mount = createTaskFormMount({ taskForm: { fetch } satisfies TaskFormPort });

		await mount.load(provisionalAddr);
		expect(fetch).toHaveBeenCalledOnce();

		expect(mount.applyGate(adminWorklist())).toEqual({ kind: 'allow' });
		expect(mount.status).toBe('ready');
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

		const first = mount.load(provisionalAddr);

		const second = await mount.load({
			stream,
			instance: truck,
			jobId: 'intake',
			taskId: 'codiceFir',
			jobInstanceId: 'NIR-advance',
			provisionalJob: false
		});

		expect(second).toEqual({ kind: 'ready' });
		expect(mount.bundle?.taskId).toBe('codiceFir');
		expect(mount.addressKey).toBe('recycling|261102-1|intake|NIR-advance|codiceFir');

		releaseSlow({ ...sample, title: 'stale-should-not-win' });
		const firstResult = await first;

		expect(firstResult).toEqual({ kind: 'aborted' });
		expect(mount.bundle?.taskId).toBe('codiceFir');
		expect(mount.bundle?.title).toBe('Codice FIR');
		expect(mount.status).toBe('ready');
		expect(fetch).toHaveBeenCalledTimes(2);
	});

	test('applyGate deny aborts in-flight fetch without a second fetch', async () => {
		let releaseSlow: (bundle: TaskFormBundle) => void = () => {};
		const slow = new Promise<TaskFormBundle>((resolve) => {
			releaseSlow = resolve;
		});
		const fetch = vi.fn<TaskFormPort['fetch']>().mockImplementationOnce(() => slow);
		const mount = createTaskFormMount({ taskForm: { fetch } });
		const blocked: Worklist = {
			jobs: [{ ...adminWorklist().jobs[0]!, canStart: false }]
		};

		const pending = mount.load(provisionalAddr);
		const gate = mount.applyGate(blocked);

		expect(gate).toEqual({ kind: 'denied', redirect: homeRedirect });
		expect(mount.status).toBe('denied');
		expect(fetch).toHaveBeenCalledOnce();

		releaseSlow(sample);
		await expect(pending).resolves.toEqual({ kind: 'aborted' });
		expect(mount.status).toBe('denied');
		expect(mount.bundle).toBeNull();
		expect(fetch).toHaveBeenCalledOnce();
	});

	test('reload re-fetches last address without worklist', async () => {
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

		await mount.load({
			stream,
			instance: truck,
			jobId: 'intake',
			taskId: 'codiceFir',
			jobInstanceId: 'NIR-advance',
			provisionalJob: false
		});

		const result = await mount.reload();

		expect(result).toEqual({ kind: 'ready' });
		expect(mount.bundle?.title).toBe('Codice FIR again');
		expect(mount.addressKey).toBe('recycling|261102-1|intake|NIR-advance|codiceFir');
		expect(fetch).toHaveBeenCalledTimes(2);
		expect(fetch).toHaveBeenLastCalledWith({
			stream,
			instance: truck,
			jobId: 'intake',
			taskId: 'codiceFir',
			jobInstanceId: 'NIR-advance'
		});
	});
});
