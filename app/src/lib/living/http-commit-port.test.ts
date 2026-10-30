import { describe, expect, test, vi } from 'vitest';

import { CommitDomainError } from './commit-port.js';
import { createHttpCommitDomainPort } from './http-commit-port.js';
import { emptyWorklist } from './types.js';

const sampleResult = {
	org: 'feragame',
	jobInstanceId: 'NIR-261019-NO-1',
	worklist: {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				taskIds: ['register', 'codiceFir'],
				instances: [
					{
						id: 'NIR-261019-NO-1',
						label: 'NIR-261019-NO-1',
						tasks: { register: 'done' as const, codiceFir: 'available' as const }
					}
				]
			},
			{
				jobId: 'line',
				canStart: false,
				repeatable: true,
				taskIds: ['shift'],
				instances: []
			}
		]
	},
	taskFormBundles: [
		{
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: '',
			title: 'Register',
			surface: 'live' as const,
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: [],
			bindings: { domain: {} }
		},
		{
			jobId: 'intake',
			taskId: 'codiceFir',
			jobInstanceId: 'NIR-261019-NO-1',
			title: 'Codice FIR',
			surface: 'live' as const,
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: ['codiceFir'],
			bindings: { domain: {} }
		}
	],
	links: [
		{ label: 'Stampa etichette', url: 'blob:labels', action: 'link' as const },
		{
			label: 'Scarica CSV',
			url: 'blob:csv',
			action: 'download' as const,
			filename: 'reg.csv'
		}
	],
	httpErrors: [] as string[]
};

describe('HttpCommitDomainPort', () => {
	test('POST /api/living/commit with taskPath + jobInstanceId + value + mint', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe('/api/living/commit');
			expect(init?.method).toBe('POST');
			expect(init?.credentials).toBe('include');
			expect(JSON.parse(String(init?.body))).toEqual({
				taskPath: 'intake/register',
				jobInstanceId: 'NIR-261019-NO-1',
				value: { client: 'NO', palletCount: 1 },
				mint: { kind: 'none' }
			});
			return new Response(JSON.stringify(sampleResult), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const port = createHttpCommitDomainPort({ fetch: fetchFn, baseUrl: '' });
		const got = await port.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-261019-NO-1',
			value: { client: 'NO', palletCount: 1 },
			mint: { kind: 'none' }
		});
		expect(got.org).toBe('feragame');
		expect(got.jobInstanceId).toBe('NIR-261019-NO-1');
		expect(got.worklist).toEqual(sampleResult.worklist);
		expect(got.taskFormBundles).toEqual(sampleResult.taskFormBundles);
		expect(got.links).toEqual(sampleResult.links);
		expect(got.httpErrors).toEqual([]);
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('normalizes missing links/httpErrors/worklist/bundles', async () => {
		const { links: _l, httpErrors: _e, worklist: _w, taskFormBundles: _b, ...withoutHttp } =
			sampleResult;
		const fetchFn = vi.fn(
			async () =>
				new Response(JSON.stringify(withoutHttp), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		);
		const port = createHttpCommitDomainPort({ fetch: fetchFn });
		const got = await port.commit({
			taskPath: 'intake/codiceFir',
			jobInstanceId: 'NIR-261019-NO-1',
			value: { codiceFir: 'FIR-1' },
			mint: { kind: 'none' }
		});
		expect(got.links).toEqual([]);
		expect(got.httpErrors).toEqual([]);
		expect(got.worklist).toEqual(emptyWorklist());
		expect(got.taskFormBundles).toEqual([]);
	});

	test('provisional startingNew omits real id on wire (empty jobInstanceId)', async () => {
		const fetchFn = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			expect(JSON.parse(String(init?.body))).toEqual({
				taskPath: 'intake/register',
				jobInstanceId: '',
				value: { client: 'NO' },
				mint: { kind: 'startingNew' }
			});
			return new Response(JSON.stringify(sampleResult), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});
		const port = createHttpCommitDomainPort({ fetch: fetchFn });
		await port.commit({
			taskPath: 'intake/register',
			jobInstanceId: '',
			value: { client: 'NO' },
			mint: { kind: 'startingNew' }
		});
	});

	test('rejects /api/runs-shaped URLs', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			expect(url).not.toContain('/api/runs');
			expect(url).toBe('https://host.example/api/living/commit');
			return new Response(JSON.stringify(sampleResult), { status: 200 });
		});
		const port = createHttpCommitDomainPort({
			fetch: fetchFn,
			baseUrl: 'https://host.example'
		});
		await port.commit({
			taskPath: 'intake/codiceFir',
			jobInstanceId: 'NIR-261019-NO-1',
			value: { codiceFir: 'FIR-1' }
		});
	});

	test('403 becomes CommitDomainError without parsing a Domain body', async () => {
		const fetchFn = vi.fn(async () => new Response('forbidden', { status: 403 }));
		const port = createHttpCommitDomainPort({ fetch: fetchFn });
		await expect(
			port.commit({
				taskPath: 'intake/register',
				jobInstanceId: 'x',
				value: {}
			})
		).rejects.toBeInstanceOf(CommitDomainError);
		await expect(
			port.commit({
				taskPath: 'intake/register',
				jobInstanceId: 'x',
				value: {}
			})
		).rejects.toMatchObject({ status: 403, message: 'forbidden' });
	});
});
