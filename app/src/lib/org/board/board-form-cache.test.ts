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

	test('beginFreshDraft advances draftGeneration and keeps ready bundle', () => {
		const cache = createBoardFormCache({
			taskForm: { fetch: async () => bundle('a') }
		});
		cache.beginScope('s');
		const a = addr('a');
		const ready = bundle('a');
		cache.putReady(a, ready);
		const before = cache.get(taskAddressKey(a));
		expect(before?.status).toBe('ready');
		expect(before?.draftGeneration).toBe(0);

		cache.beginFreshDraft(a);

		const after = cache.get(taskAddressKey(a));
		expect(after?.status).toBe('ready');
		expect(after?.bundle).toBe(ready);
		expect(after?.draftGeneration).toBe(1);

		cache.beginFreshDraft(a);
		expect(cache.get(taskAddressKey(a))?.draftGeneration).toBe(2);
		cache.dispose();
	});

	test('beginFreshDraft is a no-op when address has no entry', () => {
		const cache = createBoardFormCache({
			taskForm: { fetch: async () => bundle('a') }
		});
		cache.beginScope('s');
		cache.beginFreshDraft(addr('missing'));
		expect(cache.get(taskAddressKey(addr('missing')))).toBeUndefined();
		cache.dispose();
	});

	test('refreshAllAfterDomainCommit shows loading then ready with new bundle and bumps draftGeneration', async () => {
		const stale = bundle('a');
		stale.bindings = { domain: { lot: { 'LOT-1': {} } } };
		const fresh = bundle('a');
		fresh.bindings = { domain: { lot: { 'LOT-1': {}, 'LOT-2': {} } } };

		let resolveFetch!: (b: TaskFormBundle) => void;
		const fetch = vi.fn(
			() =>
				new Promise<TaskFormBundle>((resolve) => {
					resolveFetch = resolve;
				})
		);
		const cache = createBoardFormCache({
			taskForm: { fetch } satisfies TaskFormPort
		});
		cache.beginScope('recycling|NIR-1');
		const a = addr('a');
		cache.putReady(a, stale);
		expect(cache.get(taskAddressKey(a))?.draftGeneration).toBe(0);

		cache.refreshAllAfterDomainCommit();

		const loading = cache.get(taskAddressKey(a));
		expect(loading?.status).toBe('loading');
		expect(loading?.bundle).toBeNull();
		expect(loading?.draftGeneration).toBe(0);
		expect(fetch).toHaveBeenCalledTimes(1);

		resolveFetch(fresh);
		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(a))?.status).toBe('ready');
		});
		const ready = cache.get(taskAddressKey(a));
		expect(ready?.bundle).toBe(fresh);
		expect(ready?.draftGeneration).toBe(1);
		cache.dispose();
	});

	test('refreshAllAfterDomainCommit refetches every ready entry in scope', async () => {
		const fetch = vi.fn(async (req: { taskId: string }) => bundle(req.taskId));
		const cache = createBoardFormCache({
			taskForm: { fetch } satisfies TaskFormPort,
			concurrency: 4
		});
		cache.beginScope('recycling|NIR-1');
		cache.putReady(addr('a'), bundle('a'));
		cache.putReady(addr('b'), bundle('b'));
		cache.putReady(addr('c'), bundle('c'));
		expect(fetch).toHaveBeenCalledTimes(0);

		cache.refreshAllAfterDomainCommit();

		expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('loading');
		expect(cache.get(taskAddressKey(addr('b')))?.status).toBe('loading');
		expect(cache.get(taskAddressKey(addr('c')))?.status).toBe('loading');

		await vi.waitFor(() => {
			expect(cache.get(taskAddressKey(addr('a')))?.status).toBe('ready');
			expect(cache.get(taskAddressKey(addr('b')))?.status).toBe('ready');
			expect(cache.get(taskAddressKey(addr('c')))?.status).toBe('ready');
		});
		expect(fetch).toHaveBeenCalledTimes(3);
		expect(cache.get(taskAddressKey(addr('a')))?.draftGeneration).toBe(1);
		expect(cache.get(taskAddressKey(addr('b')))?.draftGeneration).toBe(1);
		expect(cache.get(taskAddressKey(addr('c')))?.draftGeneration).toBe(1);
		cache.dispose();
	});
});
