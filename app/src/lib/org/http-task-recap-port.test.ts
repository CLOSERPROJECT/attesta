import { describe, expect, test, vi } from 'vitest';

import { createHttpTaskRecapPort } from './http-task-recap-port.js';
import { TaskRecapError } from './task-recap-port.js';

const sampleRow = {
	key: 'lot-1',
	cells: { client: 'NO' },
	submissionId: '2026-10-29T15:00:00Z',
	timestamp: '2026-10-29T15:00:00Z',
	userId: 'ada',
	userName: 'Ada',
	links: [{ label: 'Stampa', url: 'blob:labels', action: 'link' }]
};

const sampleTable = {
	mode: 'entity' as const,
	columns: [
		{ key: 'client', title: 'Cliente' },
		{ key: 'numeroFir', title: 'Numero FIR' }
	],
	rows: [sampleRow]
};

describe('HttpTaskRecapPort', () => {
	test('sends stream + instance with flat address; normalizes missing links', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL) => {
			expect(String(input)).toBe(
				'/api/org/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T1&stream=recycling&instance=261102-1'
			);
			return new Response(
				JSON.stringify({
					mode: 'submission',
					columns: [{ key: 'date', title: 'Date' }],
					rows: [
						{
							key: '2026-10-29T15:00:00Z',
							cells: { date: '2026-10-19' },
							submissionId: '2026-10-29T15:00:00Z',
							timestamp: '2026-10-29T15:00:00Z',
							userId: 'ada',
							userName: 'Ada'
						}
					]
				}),
				{ status: 200 }
			);
		});
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		const got = await port.fetch({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			taskId: 'truckArrive',
			jobInstanceId: 'T1'
		});
		expect(got.mode).toBe('submission');
		expect(got.rows).toHaveLength(1);
		expect(got.rows[0]?.links).toBeUndefined();
		expect(got.rows[0]?.cells.date).toBe('2026-10-19');
		expect(got.columns[0]?.title).toBe('Date');
	});

	test('parses entity table with links', async () => {
		const fetchFn = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						mode: 'entity',
						columns: sampleTable.columns,
						rows: [
							{
								...sampleRow,
								key: 'lot-a',
								cells: { client: 'CO' },
								links: [{ label: 'A', url: 'blob:a', action: 'link' }]
							},
							{
								...sampleRow,
								key: 'lot-b',
								cells: { client: 'DA' },
								submissionId: '2026-10-29T16:00:00Z',
								timestamp: '2026-10-29T16:00:00Z',
								links: [{ label: 'B', url: 'blob:b', action: 'link' }]
							}
						]
					}),
					{ status: 200 }
				)
		);
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		const got = await port.fetch({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: 'T1'
		});
		expect(got.mode).toBe('entity');
		expect(got.rows).toHaveLength(2);
		expect(got.rows[0]?.cells.client).toBe('CO');
		expect(got.rows[1]?.cells.client).toBe('DA');
		expect(got.rows[0]?.links?.[0]?.url).toBe('blob:a');
		expect(got.rows[1]?.links?.[0]?.url).toBe('blob:b');
	});

	test('rejects body without mode/rows', async () => {
		const fetchFn = vi.fn(
			async () =>
				new Response(JSON.stringify({ items: [sampleRow] }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		);
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		await expect(
			port.fetch({
				stream: 'recycling',
				instance: '261102-1',
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: 'T1'
			})
		).rejects.toMatchObject({ name: 'TaskRecapError', status: 500 } satisfies Partial<TaskRecapError>);
	});

	test('maps non-OK to TaskRecapError', async () => {
		const fetchFn = vi.fn(
			async () => new Response('task recap not found', { status: 404 })
		);
		const port = createHttpTaskRecapPort({ fetch: fetchFn });
		await expect(
			port.fetch({
				stream: 'recycling',
				instance: '261102-1',
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: 'T1'
			})
		).rejects.toMatchObject({ name: 'TaskRecapError', status: 404 } satisfies Partial<TaskRecapError>);
	});
});
