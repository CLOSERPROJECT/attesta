import { describe, expect, test, vi } from 'vitest';

import { createHttpLoadDomainPort } from './http-load-port.js';
import { LoadDomainError } from './load-port.js';
import { emptyWorklist, type LoadResult } from './types.js';

const sample: LoadResult = {
	org: 'feragame',
	activeConfigVersion: '1',
	configId: 'truck-loop',
	worklist: emptyWorklist(),
	taskFormBundles: []
};

describe('HttpLoadDomainPort', () => {
	test('GET /api/living/domain returns LoadResult (no run id)', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe('/api/living/domain');
			expect(init?.method).toBe('GET');
			expect(init?.credentials).toBe('include');
			return new Response(JSON.stringify(sample), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const port = createHttpLoadDomainPort({ fetch: fetchFn, baseUrl: '' });
		const got = await port.load();
		expect(got.org).toBe('feragame');
		expect(got.activeConfigVersion).toBe('1');
		expect(got.configId).toBe('truck-loop');
		expect(got.worklist).toEqual(emptyWorklist());
		expect(got.taskFormBundles).toEqual([]);
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('rejects /api/runs-shaped URLs (adapter only hits living domain)', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			expect(url).not.toContain('/api/runs');
			expect(url).toBe('https://host.example/api/living/domain');
			return new Response(JSON.stringify(sample), { status: 200 });
		});
		const port = createHttpLoadDomainPort({
			fetch: fetchFn,
			baseUrl: 'https://host.example'
		});
		await port.load();
	});

	test('non-OK response becomes LoadDomainError', async () => {
		const fetchFn = vi.fn(
			async () => new Response('organization required', { status: 403 })
		);
		const port = createHttpLoadDomainPort({ fetch: fetchFn });
		await expect(port.load()).rejects.toBeInstanceOf(LoadDomainError);
		await expect(port.load()).rejects.toMatchObject({ status: 403 });
	});
});
