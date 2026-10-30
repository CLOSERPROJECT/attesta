import { describe, expect, test, vi } from 'vitest';

import { CommitDomainError } from './commit-port.js';
import { LivingSession } from './session.svelte.js';
import type { LoadDomainPort } from './load-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import { emptyWorklist, type LoadResult } from './types.js';

const sample: LoadResult = {
	org: 'feragame',
	activeConfigVersion: '1',
	configId: 'truck-loop',
	worklist: {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				startLabel: 'Nuovo camion',
				taskIds: ['register'],
				instances: []
			},
			{
				jobId: 'line',
				canStart: false,
				repeatable: true,
				startLabel: 'Nuovo turno',
				taskIds: ['shift'],
				instances: []
			}
		]
	},
	taskFormBundles: []
};

describe('LivingSession', () => {
	test('hydrate loads Worklist + bundles via LoadDomainPort', async () => {
		const load = vi.fn(async () => sample) satisfies LoadDomainPort['load'];
		const session = new LivingSession({
			load: { load },
			hydrate: false
		});
		expect(session.result).toBeNull();
		await session.hydrate();
		expect(load).toHaveBeenCalledOnce();
		expect(session.error).toBeNull();
		expect(session.result?.org).toBe('feragame');
		expect(session.jobIds).toEqual(['intake', 'line']);
		expect(session.worklistJob('intake')?.startLabel).toBe('Nuovo camion');
	});

	test('hydrate surfaces port errors without leaving stale result', async () => {
		const session = new LivingSession({
			load: {
				load: async () => {
					throw new Error('boom');
				}
			},
			hydrate: false
		});
		session.result = sample;
		await session.hydrate();
		expect(session.result).toBeNull();
		expect(session.error).toBe('boom');
	});

	test('commit success replaces Worklist + bundles and clears Draft', async () => {
		const nextWorklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['register', 'codiceFir'],
					instances: [
						{
							id: 'NIR-1',
							label: 'NIR-1',
							tasks: { register: 'done' as const, codiceFir: 'available' as const }
						}
					]
				},
				{ jobId: 'line', canStart: false, repeatable: true, taskIds: ['shift'], instances: [] }
			]
		};
		const nextBundles: TaskFormBundle[] = [
			{
				jobId: 'intake',
				taskId: 'codiceFir',
				jobInstanceId: 'NIR-1',
				title: 'Codice FIR',
				surface: 'live',
				schema: { type: 'object' },
				uiSchema: {},
				initialData: {},
				chrome: [],
				chromeFieldIds: [],
				sealFieldIds: ['codiceFir'],
				bindings: { domain: {} }
			}
		];
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: nextWorklist,
			taskFormBundles: nextBundles
		}));
		const session = new LivingSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.setDraft('intake/register', 'NIR-1', { client: 'NO' });
		const minted = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: { client: 'NO', palletCount: 1 },
			mint: { kind: 'none' }
		});
		expect(minted).toBe('NIR-1');
		expect(session.result?.worklist).toEqual(nextWorklist);
		expect(session.result?.taskFormBundles).toEqual(nextBundles);
		expect(session.getDraft('intake/register', 'NIR-1')).toBeNull();
		expect(session.commitError).toBeNull();
	});

	test('provisional startingNew clears _new draft and returns response id', async () => {
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: emptyWorklist(),
			taskFormBundles: []
		}));
		const session = new LivingSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.setDraft('intake/register', '', { client: 'NO' });
		const minted = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: '',
			value: { client: 'NO' },
			mint: { kind: 'startingNew' }
		});
		expect(minted).toBe('NIR-1');
		expect(commit).toHaveBeenCalledWith({
			taskPath: 'intake/register',
			jobInstanceId: '',
			value: { client: 'NO' },
			mint: { kind: 'startingNew' }
		});
		expect(session.getDraft('intake/register', '')).toBeNull();
		expect(session.getDraft('intake/register', '_new')).toBeNull();
	});

	test('commit denial keeps Draft and re-Loads Worklist', async () => {
		const reloaded = structuredClone(sample);
		reloaded.worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['register'],
					instances: []
				}
			]
		};
		const load = vi.fn(async () => reloaded);
		const commit = vi.fn(async () => {
			throw new CommitDomainError('forbidden', 403);
		});
		const session = new LivingSession({
			load: { load },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.setDraft('intake/register', '_new', { client: 'NO', notes: 'keep me' });

		const minted = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: { client: 'NO' },
			mint: { kind: 'none' }
		});
		expect(minted).toBeNull();
		expect(session.commitError).toBe('forbidden');
		expect(session.getDraft('intake/register', '_new')).toEqual({
			client: 'NO',
			notes: 'keep me'
		});
		expect(load).toHaveBeenCalledOnce();
		expect(session.result?.worklist).toEqual(reloaded.worklist);
	});
});
