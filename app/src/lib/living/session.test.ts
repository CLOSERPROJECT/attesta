import { describe, expect, test, vi } from 'vitest';

import { CommitDomainError } from './commit-port.js';
import { LivingSession } from './session.svelte.js';
import type { LoadDomainPort } from './load-port.js';
import { emptyDomain, type LoadResult } from './types.js';

const sample: LoadResult = {
	org: 'feragame',
	activeConfigVersion: '1',
	domain: emptyDomain(),
	config: {
		id: 'truck-loop',
		org: 'feragame',
		version: '1',
		jobs: {
			intake: {
				repeatable: true,
				label: '$load.id',
				startLabel: 'Nuovo camion',
				tasks: [{ id: 'register' }]
			},
			line: {
				repeatable: true,
				label: 'line',
				startLabel: 'Nuovo turno',
				tasks: [{ id: 'shift' }]
			}
		}
	}
};

describe('LivingSession', () => {
	test('hydrate loads Domain + Active Config via LoadDomainPort', async () => {
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
		expect(session.job('intake')?.startLabel).toBe('Nuovo camion');
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

	test('commit success replaces Domain and clears Draft', async () => {
		const nextDomain = {
			...emptyDomain(),
			load: { 'NIR-1': { client: 'NO' } },
			jobInstances: [{ jobId: 'intake', id: 'NIR-1', label: 'NIR-1' }],
			submissions: [
				{
					jobId: 'intake',
					taskId: 'register',
					jobInstanceId: 'NIR-1',
					timestamp: 't',
					userId: 'ada',
					facts: {},
					idsTouched: ['NIR-1']
				}
			]
		};
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			domain: nextDomain
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
		expect(session.result?.domain.load['NIR-1']?.client).toBe('NO');
		expect(session.getDraft('intake/register', 'NIR-1')).toBeNull();
		expect(session.commitError).toBeNull();
	});

	test('provisional startingNew clears _new draft and returns response id', async () => {
		const nextDomain = {
			...emptyDomain(),
			load: { 'NIR-1': { client: 'NO' } },
			jobInstances: [{ jobId: 'intake', id: 'NIR-1', label: 'NIR-1' }],
			submissions: []
		};
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			domain: nextDomain
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

	test('commit denial keeps Domain + Draft and re-Loads', async () => {
		const load = vi.fn(async () => sample);
		const commit = vi.fn(async () => {
			throw new CommitDomainError('forbidden', 403);
		});
		const session = new LivingSession({
			load: { load },
			commit: { commit },
			hydrate: false
		});
		const seeded = structuredClone(sample);
		seeded.domain.load = { existing: { client: 'SU' } };
		session.result = seeded;
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
		// Re-Load returned empty Domain — still no corruption from failed Commit body.
		expect(session.result?.domain.load).toEqual({});
	});
});
