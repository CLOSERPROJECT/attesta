import { describe, expect, test, vi } from 'vitest';

import { normalizeTaskAddress, taskAddressKey } from '../task-address.js';
import {
	EMPTY_TASK_RECAP_TABLE,
	TaskRecapError,
	type TaskRecapPort,
	type TaskRecapTable
} from '../task-recap-port.js';
import { createBoardRecapCache } from './board-recap-cache.svelte.js';

const recap = (taskId: string, n = 1): TaskRecapTable => ({
	mode: 'submission',
	columns: [{ key: 'a', title: 'A' }],
	rows: [
		{
			key: `${taskId}-${n}`,
			cells: { a: n },
			submissionId: '2024-01-01T00:00:00Z',
			timestamp: '2024-01-01T00:00:00Z',
			userId: 'u1',
			userName: 'U1'
		}
	]
});

function addr(taskId: string) {
	return normalizeTaskAddress({
		stream: 'pv-recovery',
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
			return recap(req.taskId);
		});
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort,
			concurrency: 2
		});
		cache.beginScope('pv-recovery|NIR-1');
		cache.enqueue([addr('a'), addr('b'), addr('c'), addr('d')]);
		expect(fetch).toHaveBeenCalledTimes(2);

		const readyAddr = addr('e');
		cache.putReady(readyAddr, recap('e'));
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
		let resolveFetch!: (table: TaskRecapTable) => void;
		const fetch = vi.fn(
			() =>
				new Promise<TaskRecapTable>((resolve) => {
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
		resolveFetch(recap('a'));
		await Promise.resolve();
		expect(cache.get(taskAddressKey(addr('a')))).toBeUndefined();
		cache.dispose();
	});

	test('invalidate drops entry', () => {
		const cache = createBoardRecapCache({
			taskRecap: { fetch: async () => recap('a') }
		});
		cache.beginScope('s');
		const a = addr('a');
		cache.putReady(a, recap('a'));
		cache.invalidate(taskAddressKey(a));
		expect(cache.get(taskAddressKey(a))).toBeUndefined();
		cache.dispose();
	});

	test('HTTP 404 maps to ready with empty table', async () => {
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
		expect(cache.get(taskAddressKey(addr('a')))?.table).toEqual(EMPTY_TASK_RECAP_TABLE);
		cache.dispose();
	});

	test('empty table maps to ready with empty rows', async () => {
		const fetch = vi.fn(async () => EMPTY_TASK_RECAP_TABLE);
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort
		});
		cache.beginScope('s');
		cache.enqueue([addr('a')]);
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('ready');
		});
		expect(cache.get(taskAddressKey(addr('a')))?.table.rows).toEqual([]);
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

	test('revalidateAll keeps old table until the new table lands', async () => {
		let resolveFetch!: (table: TaskRecapTable) => void;
		const fetch = vi.fn(
			() =>
				new Promise<TaskRecapTable>((resolve) => {
					resolveFetch = resolve;
				})
		);
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort
		});
		cache.beginScope('s');
		const a = addr('a');
		const oldTable = recap('a', 1);
		cache.putReady(a, oldTable);

		cache.revalidateAll();
		expect(cache.get(taskAddressKey(a))?.status).toBe('ready');
		expect(cache.get(taskAddressKey(a))?.table).toBe(oldTable);

		const nextTable = recap('a', 2);
		resolveFetch(nextTable);
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(a))?.table).toEqual(nextTable);
		});
		expect(cache.get(taskAddressKey(a))?.status).toBe('ready');
		cache.dispose();
	});

	test('revalidateAll keeps old table when refetch fails', async () => {
		const fetch = vi.fn(async () => {
			throw new TaskRecapError('boom', 500);
		});
		const cache = createBoardRecapCache({
			taskRecap: { fetch } satisfies TaskRecapPort
		});
		cache.beginScope('s');
		const a = addr('a');
		const oldTable = recap('a', 1);
		cache.putReady(a, oldTable);

		cache.revalidateAll();
		await vi.waitFor(() => {
			expect(fetch).toHaveBeenCalled();
		});
		await Promise.resolve();
		expect(cache.get(taskAddressKey(a))?.status).toBe('ready');
		expect(cache.get(taskAddressKey(a))?.table).toBe(oldTable);
		cache.dispose();
	});

	test('revalidateAll re-enqueues error entries', async () => {
		const fetch = vi.fn(async (): Promise<TaskRecapTable> => {
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

		fetch.mockImplementation(async () => recap('a', 2));
		cache.revalidateAll();
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(a))?.status).toBe('ready');
		});
		expect(cache.get(taskAddressKey(a))?.table.rows[0]?.cells.a).toBe(2);
		expect(fetch).toHaveBeenCalledTimes(2);
		cache.dispose();
	});
});
