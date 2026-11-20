import { describe, expect, test, vi } from 'vitest';

import type { TaskFormBundle } from '../form/task-form-bundle.js';
import { normalizeTaskAddress, taskAddressKey } from '../task-address.js';
import type { TaskFormPort } from '../task-form-port.js';
import { createBoardFormCache } from './board-form-cache.svelte.js';

const bundle = (taskId: string): TaskFormBundle => ({
	jobId: 'intake',
	taskId,
	jobInstanceId: 'NIR-1',
	title: taskId,
	surface: 'live',
	schema: { type: 'object' },
	uiSchema: {},
	initialData: {},
	chrome: [],
	chromeFieldIds: [],
	sealFieldIds: [],
	bindings: { domain: {} }
});

function addr(taskId: string) {
	return normalizeTaskAddress({
		stream: 'recycling',
		instance: 'NIR-1',
		jobId: 'intake',
		jobInstanceId: 'NIR-1',
		taskId
	});
}

describe('BoardFormCache', () => {
	test('preloads with concurrency cap and putReady skips fetch', async () => {
		const fetch = vi.fn(async (req: { taskId: string }) => {
			await new Promise((r) => setTimeout(r, 20));
			return bundle(req.taskId);
		});
		const cache = createBoardFormCache({
			taskForm: { fetch } satisfies TaskFormPort,
			concurrency: 2
		});
		cache.beginScope('recycling|NIR-1');
		cache.enqueue([addr('a'), addr('b'), addr('c'), addr('d')]);
		expect(fetch).toHaveBeenCalledTimes(2);

		const readyAddr = addr('e');
		cache.putReady(readyAddr, bundle('e'));
		expect(cache.get(taskAddressKey(readyAddr))?.status).toBe('ready');
		expect(fetch).toHaveBeenCalledTimes(2);

		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('ready');
			expect(cache.get(taskAddressKey(addr('d')))?.status).toBe('ready');
		});
		expect(fetch).toHaveBeenCalledTimes(4);
		cache.dispose();
	});

	test('scope change aborts queue and clears entries', async () => {
		let resolveFetch!: (b: TaskFormBundle) => void;
		const fetch = vi.fn(
			() =>
				new Promise<TaskFormBundle>((resolve) => {
					resolveFetch = resolve;
				})
		);
		const cache = createBoardFormCache({
			taskForm: { fetch } satisfies TaskFormPort,
			concurrency: 4
		});
		cache.beginScope('s1');
		cache.enqueue([addr('a')]);
		expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('loading');
		cache.beginScope('s2');
		expect(cache.get(taskAddressKey(addr('a')))).toBeUndefined();
		resolveFetch(bundle('a'));
		await Promise.resolve();
		expect(cache.get(taskAddressKey(addr('a')))).toBeUndefined();
		cache.dispose();
	});

	test('invalidate drops entry', () => {
		const cache = createBoardFormCache({
			taskForm: { fetch: async () => bundle('a') }
		});
		cache.beginScope('s');
		const a = addr('a');
		cache.putReady(a, bundle('a'));
		cache.invalidate(taskAddressKey(a));
		expect(cache.get(taskAddressKey(a))).toBeUndefined();
		cache.dispose();
	});
});
