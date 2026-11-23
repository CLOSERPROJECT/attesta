import { describe, expect, test, vi } from 'vitest';

import { normalizeTaskAddress, taskAddressKey } from '../task-address.js';
import { TaskRecapError, type TaskRecap, type TaskRecapPort } from '../task-recap-port.js';
import { createBoardRecapCache } from './board-recap-cache.svelte.js';

const recap = (taskId: string, n = 1): TaskRecap => ({
	jobId: 'intake',
	jobInstanceId: 'NIR-1',
	taskId,
	timestamp: '2024-01-01T00:00:00Z',
	userId: 'u1',
	facts: { a: n },
	links: []
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

describe('BoardRecapCache', () => {
	test('preloads with concurrency cap and putReady skips fetch', async () => {
		const fetch = vi.fn(async (req: { taskId: string }) => {
			await new Promise((r) => setTimeout(r, 20));
			return { items: [recap(req.taskId)] };
		});
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort,
			concurrency: 2
		});
		cache.beginScope('recycling|NIR-1');
		cache.enqueue([addr('a'), addr('b'), addr('c'), addr('d')]);
		expect(fetch).toHaveBeenCalledTimes(2);

		const readyAddr = addr('e');
		cache.putReady(readyAddr, [recap('e')]);
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
		let resolveFetch!: (list: { items: TaskRecap[] }) => void;
		const fetch = vi.fn(
			() =>
				new Promise<{ items: TaskRecap[] }>((resolve) => {
					resolveFetch = resolve;
				})
		);
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort,
			concurrency: 4
		});
		cache.beginScope('s1');
		cache.enqueue([addr('a')]);
		expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('loading');
		cache.beginScope('s2');
		expect(cache.get(taskAddressKey(addr('a')))).toBeUndefined();
		resolveFetch({ items: [recap('a')] });
		await Promise.resolve();
		expect(cache.get(taskAddressKey(addr('a')))).toBeUndefined();
		cache.dispose();
	});

	test('invalidate drops entry', () => {
		const cache = createBoardRecapCache({
			taskRecap: { fetch: async () => ({ items: [recap('a')] }) }
		});
		cache.beginScope('s');
		const a = addr('a');
		cache.putReady(a, [recap('a')]);
		cache.invalidate(taskAddressKey(a));
		expect(cache.get(taskAddressKey(a))).toBeUndefined();
		cache.dispose();
	});

	test('HTTP 404 maps to ready with empty items', async () => {
		const fetch = vi.fn(async () => {
			throw new TaskRecapError('missing', 404);
		});
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort
		});
		cache.beginScope('s');
		cache.enqueue([addr('a')]);
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('ready');
		});
		expect(cache.get(taskAddressKey(addr('a')))?.items).toEqual([]);
		cache.dispose();
	});

	test('empty list maps to ready with empty items', async () => {
		const fetch = vi.fn(async () => ({ items: [] }));
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort
		});
		cache.beginScope('s');
		cache.enqueue([addr('a')]);
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('ready');
		});
		expect(cache.get(taskAddressKey(addr('a')))?.items).toEqual([]);
		cache.dispose();
	});

	test('enqueue does not retry error without invalidate', async () => {
		const fetch = vi.fn(async () => {
			throw new TaskRecapError('bad request', 400);
		});
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort
		});
		cache.beginScope('s');
		const a = addr('a');
		cache.enqueue([a]);
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(a))?.status).toBe('error');
		});
		expect(fetch).toHaveBeenCalledTimes(1);

		cache.enqueue([a]);
		await Promise.resolve();
		expect(fetch).toHaveBeenCalledTimes(1);
		expect(cache.get(taskAddressKey(a))?.status).toBe('error');

		cache.invalidate(taskAddressKey(a));
		cache.enqueue([a]);
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(a))?.status).toBe('error');
		});
		expect(fetch).toHaveBeenCalledTimes(2);
		cache.dispose();
	});
});
