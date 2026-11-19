import { describe, expect, test, vi } from 'vitest';

import { createHttpTaskFormPort } from './http-task-form-port.js';
import { TaskFormError } from './task-form-port.js';
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

const scoped = { stream: 'recycling', instance: 'T-1' };
const play = { stream: 'recycling' };

describe('HttpTaskFormPort', () => {
	test('GET /api/org/task-form sends stream + instance with job/task', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe(
				'/api/org/task-form?jobId=intake&taskId=register&stream=recycling&instance=T-1'
			);
			expect(init?.method).toBe('GET');
			expect(init?.credentials).toBe('include');
			return new Response(JSON.stringify(sample), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const port = createHttpTaskFormPort({ fetch: fetchFn, baseUrl: '' });
		const got = await port.fetch({ ...scoped, jobId: 'intake', taskId: 'register' });
		expect(got.jobId).toBe('intake');
		expect(got.taskId).toBe('register');
		expect(got.jobInstanceId).toBe('');
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('Play omits instance on the wire', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			expect(String(input)).toBe(
				'/api/org/task-form?jobId=intake&taskId=truckArrive&stream=recycling'
			);
			return new Response(JSON.stringify(sample), { status: 200 });
		});
		const port = createHttpTaskFormPort({ fetch: fetchFn });
		await port.fetch({ ...play, jobId: 'intake', taskId: 'truckArrive' });
	});

	test('sends empty jobInstanceId and new for provisional wire shapes', async () => {
		const urls: string[] = [];
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			urls.push(String(input));
			return new Response(JSON.stringify(sample), { status: 200 });
		});
		const port = createHttpTaskFormPort({ fetch: fetchFn });

		await port.fetch({
			...scoped,
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: ''
		});
		await port.fetch({
			...scoped,
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: 'new'
		});

		expect(urls).toEqual([
			'/api/org/task-form?jobId=intake&taskId=register&jobInstanceId=&stream=recycling&instance=T-1',
			'/api/org/task-form?jobId=intake&taskId=register&jobInstanceId=new&stream=recycling&instance=T-1'
		]);
	});

	test('sends real jobInstanceId for available Task mount', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			expect(String(input)).toBe(
				'/api/org/task-form?jobId=intake&taskId=codiceFir&jobInstanceId=NIR-261019-NO-1&stream=recycling&instance=T-1'
			);
			return new Response(
				JSON.stringify({
					...sample,
					taskId: 'codiceFir',
					jobInstanceId: 'NIR-261019-NO-1',
					title: 'Codice FIR',
					sealFieldIds: ['codiceFir']
				}),
				{ status: 200 }
			);
		});
		const port = createHttpTaskFormPort({ fetch: fetchFn });
		const got = await port.fetch({
			...scoped,
			jobId: 'intake',
			taskId: 'codiceFir',
			jobInstanceId: 'NIR-261019-NO-1'
		});
		expect(got.taskId).toBe('codiceFir');
		expect(got.jobInstanceId).toBe('NIR-261019-NO-1');
	});

	test('rejects /api/runs-shaped URLs', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			expect(url).not.toContain('/api/runs');
			expect(url).toBe(
				'https://host.example/api/org/task-form?jobId=intake&taskId=register&stream=recycling&instance=T-1'
			);
			return new Response(JSON.stringify(sample), { status: 200 });
		});
		const port = createHttpTaskFormPort({
			fetch: fetchFn,
			baseUrl: 'https://host.example'
		});
		await port.fetch({ ...scoped, jobId: 'intake', taskId: 'register' });
	});

	test('non-OK response becomes TaskFormError', async () => {
		const fetchFn = vi.fn(async () => new Response('forbidden', { status: 403 }));
		const port = createHttpTaskFormPort({ fetch: fetchFn });
		await expect(
			port.fetch({
				...scoped,
				jobId: 'intake',
				taskId: 'palletWeight',
				jobInstanceId: 'x'
			})
		).rejects.toBeInstanceOf(TaskFormError);
		await expect(
			port.fetch({
				...scoped,
				jobId: 'intake',
				taskId: 'palletWeight',
				jobInstanceId: 'x'
			})
		).rejects.toMatchObject({ status: 403, message: 'forbidden' });
	});
});
