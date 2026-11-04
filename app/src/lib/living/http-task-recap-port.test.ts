import { describe, expect, test, vi } from 'vitest';

import { createHttpTaskRecapPort } from './http-task-recap-port.js';
import { TaskRecapError } from './task-recap-port.js';

const sample = {
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
		expect(got.taskId).toBe('register');
		expect(got.facts.client).toBe('NO');
		expect(got.titles).toEqual({ client: 'Cliente', numeroFir: 'Numero FIR' });
		expect(got.links).toHaveLength(2);
		expect(got.links[1]?.filename).toBe('reg.csv');
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('flat address omits group params; normalizes missing links', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			expect(String(input)).toBe(
				'/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T1'
			);
			return new Response(
				JSON.stringify({
					jobId: 'intake',
					jobInstanceId: 'T1',
					taskId: 'truckArrive',
					timestamp: '2026-10-29T15:00:00Z',
					userId: 'ada',
					facts: { date: '2026-10-19' }
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
		expect(got.links).toEqual([]);
		expect(got.facts.date).toBe('2026-10-19');
		expect(got.titles).toBeUndefined();
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
