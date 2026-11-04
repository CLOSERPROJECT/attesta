import { describe, expect, test, vi } from 'vitest';

import { CommitDomainError } from './commit-port.js';
import { createHttpCloseGroupPort } from './http-group-close-port.js';
import { emptyWorklist } from './types.js';

const closedBoard = {
	org: 'grouporg',
	jobInstanceId: 'T1',
	next: 'task' as const,
	worklist: {
		jobs: [
			{
				jobId: 'intake',
				canStart: false,
				repeatable: true,
				taskIds: ['lots', 'weightCheck'],
				tasks: [],
				instances: [
					{
						id: 'T1',
						label: 'T1',
						tasks: { lots: 'done' as const, weightCheck: 'actionable' as const },
						groups: {
							lots: {
								groupId: 'lots',
								closeLabel: 'Chiudi scarico',
								repeatable: true,
								canSpawn: false,
								canClose: false,
								childTaskIds: ['register'],
								instances: [{ id: 'lots-1', tasks: { register: 'done' as const } }]
							}
						}
					}
				]
			}
		]
	},
	nextTaskFormBundle: {
		jobId: 'intake',
		taskId: 'weightCheck',
		jobInstanceId: 'T1',
		title: 'Peso',
		surface: 'live' as const,
		schema: { type: 'object' },
		uiSchema: {},
		initialData: {},
		chrome: [],
		chromeFieldIds: [],
		sealFieldIds: [],
		bindings: { domain: {} }
	}
};

describe('HttpGroupClosePort', () => {
	test('POST /api/living/close with jobId + jobInstanceId + groupId', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe('/api/living/close');
			expect(init?.method).toBe('POST');
			expect(JSON.parse(String(init?.body))).toEqual({
				jobId: 'intake',
				jobInstanceId: 'T1',
				groupId: 'lots'
			});
			return new Response(JSON.stringify(closedBoard), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});
		const port = createHttpCloseGroupPort({ fetch: fetchFn });
		const got = await port.close({
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots'
		});
		expect(got.next).toBe('task');
		expect(got.nextTaskFormBundle?.taskId).toBe('weightCheck');
		expect(got.worklist.jobs[0]?.instances[0]?.tasks.weightCheck).toBe('actionable');
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('surfaces non-OK close as CommitDomainError', async () => {
		const fetchFn = vi.fn(async () => new Response('cannot close', { status: 400 }));
		const port = createHttpCloseGroupPort({ fetch: fetchFn });
		await expect(
			port.close({ jobId: 'intake', jobInstanceId: 'T1', groupId: 'lots' })
		).rejects.toBeInstanceOf(CommitDomainError);
	});

	test('normalizes missing worklist/next on close', async () => {
		const fetchFn = vi.fn(
			async () =>
				new Response(JSON.stringify({ org: 'grouporg', jobInstanceId: 'T1' }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		);
		const port = createHttpCloseGroupPort({ fetch: fetchFn });
		const got = await port.close({
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots'
		});
		expect(got.org).toBe('grouporg');
		expect(got.jobInstanceId).toBe('T1');
		expect(got.next).toBe('home');
		expect(got.worklist).toEqual(emptyWorklist());
		expect(got.nextTaskFormBundle).toBeNull();
		expect(got.links).toEqual([]);
		expect(got.httpErrors).toEqual([]);
	});
});
