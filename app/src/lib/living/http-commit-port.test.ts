import { describe, expect, test, vi } from 'vitest';

import { CommitDomainError } from './commit-port.js';
import { createHttpCommitDomainPort } from './http-commit-port.js';
import { emptyDomain } from './types.js';

const sampleResult = {
	org: 'feragame',
	jobInstanceId: 'NIR-261019-NO-1',
	domain: {
		...emptyDomain(),
		load: {
			'NIR-261019-NO-1': { client: 'NO', palletCount: 1 }
		},
		jobInstances: [{ jobId: 'intake', id: 'NIR-261019-NO-1', label: 'NIR-261019-NO-1' }],
		submissions: [
			{
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: 'NIR-261019-NO-1',
				timestamp: '2026-10-29T15:00:00Z',
				userId: 'ada',
				facts: {},
				idsTouched: ['NIR-261019-NO-1']
			}
		]
	}
};

describe('HttpCommitDomainPort', () => {
	test('POST /api/living/commit with taskPath + jobInstanceId + value', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe('/api/living/commit');
			expect(init?.method).toBe('POST');
			expect(init?.credentials).toBe('include');
			expect(JSON.parse(String(init?.body))).toEqual({
				taskPath: 'intake/register',
				jobInstanceId: 'NIR-261019-NO-1',
				value: { client: 'NO', palletCount: 1 }
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
			value: { client: 'NO', palletCount: 1 }
		});
		expect(got.org).toBe('feragame');
		expect(got.jobInstanceId).toBe('NIR-261019-NO-1');
		expect(got.domain.load['NIR-261019-NO-1']?.client).toBe('NO');
		expect(fetchFn).toHaveBeenCalledOnce();
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
