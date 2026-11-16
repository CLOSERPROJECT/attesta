import { describe, expect, test, vi } from 'vitest';

import { createHttpTaskRecapPort } from './http-task-recap-port.js';
import { TaskRecapError } from './task-recap-port.js';

const sampleItem = {
	jobId: 'intake',
	jobInstanceId: 'T1',
	taskId: 'register',
	groupId: 'lots',
	groupInstanceId: 'lots-1',
	timestamp: '2026-10-29T15:00:00Z',
	userId: 'ada',
	facts: { client: 'NO' },
	titles: { client: 'Cliente', numeroFir: 'Numero FIR' },
	links: [
		{ label: 'Stampa', url: 'blob:labels', action: 'link' },
		{ label: 'CSV', url: 'blob:csv', action: 'download', filename: 'reg.csv' }
	]
};

const sample = { items: [sampleItem] };

describe('HttpTaskRecapPort', () => {
	test('GET /api/living/task-recap with address + nested group', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe(
				'/api/living/task-recap?jobId=intake&taskId=register&jobInstanceId=T1&groupId=lots&groupInstanceId=lots-1'
			);
			expect(init?.method).toBe('GET');
			expect(init?.credentials).toBe('include');
			return new Response(JSON.stringify(sample), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const port = createHttpTaskRecapPort({ fetch: fetchFn, baseUrl: '' });
		const got = await port.fetch({
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'lots-1'
		});
		expect(got.items).toHaveLength(1);
		expect(got.items[0]?.taskId).toBe('register');
		expect(got.items[0]?.facts.client).toBe('NO');
		expect(got.items[0]?.titles).toEqual({ client: 'Cliente', numeroFir: 'Numero FIR' });
		expect(got.items[0]?.links).toHaveLength(2);
		expect(got.items[0]?.links[1]?.filename).toBe('reg.csv');
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('flat address omits group params; normalizes missing links', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			expect(String(input)).toBe(
				'/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T1'
			);
			return new Response(
				JSON.stringify({
					items: [
						{
							jobId: 'intake',
							jobInstanceId: 'T1',
							taskId: 'truckArrive',
							timestamp: '2026-10-29T15:00:00Z',
							userId: 'ada',
							facts: { date: '2026-10-19' }
						}
					]
				}),
				{ status: 200 }
			);
		});
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		const got = await port.fetch({
			jobId: 'intake',
			taskId: 'truckArrive',
			jobInstanceId: 'T1'
		});
		expect(got.items).toHaveLength(1);
		expect(got.items[0]?.links).toEqual([]);
		expect(got.items[0]?.facts.date).toBe('2026-10-19');
		expect(got.items[0]?.titles).toBeUndefined();
	});

	test('parses many items oldest-first wire', async () => {
		const fetchFn = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						items: [
							{
								...sampleItem,
								timestamp: '2026-10-29T14:00:00Z',
								facts: { client: 'CO' },
								links: [{ label: 'A', url: 'blob:a', action: 'link' }]
							},
							{
								...sampleItem,
								timestamp: '2026-10-29T15:00:00Z',
								facts: { client: 'DA' },
								links: [{ label: 'B', url: 'blob:b', action: 'link' }]
							}
						]
					}),
					{ status: 200 }
				)
		);
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		const got = await port.fetch({
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: 'T1'
		});
		expect(got.items).toHaveLength(2);
		expect(got.items[0]?.facts.client).toBe('CO');
		expect(got.items[1]?.facts.client).toBe('DA');
		expect(got.items[0]?.links[0]?.url).toBe('blob:a');
		expect(got.items[1]?.links[0]?.url).toBe('blob:b');
	});

	test('rejects single-object body without items', async () => {
		const fetchFn = vi.fn(
			async () =>
				new Response(JSON.stringify(sampleItem), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		);
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		await expect(
			port.fetch({ jobId: 'intake', taskId: 'register', jobInstanceId: 'T1' })
		).rejects.toMatchObject({ name: 'TaskRecapError', status: 500 } satisfies Partial<TaskRecapError>);
	});

	test('maps non-OK to TaskRecapError', async () => {
		const fetchFn = vi.fn(
			async () => new Response('task recap not found', { status: 404 })
		);
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		await expect(
			port.fetch({ jobId: 'intake', taskId: 'register', jobInstanceId: 'T1' })
		).rejects.toMatchObject({ name: 'TaskRecapError', status: 404 } satisfies Partial<TaskRecapError>);
	});
});
