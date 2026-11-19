import { describe, expect, test, vi } from 'vitest';

import { CommitDomainError } from './commit-port.js';
import { draftAddressForTask, draftKey } from './drafts.js';
import {
	orgDraftStorageKey,
	memoryStorage,
	readStoredDraft
} from './draft-storage.js';
import type { LoadDomainPort } from './load-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import {
	DEFAULT_DRAFT_DEBOUNCE_MS,
	OrgSession,
	type DraftPersistClock
} from './session.svelte.js';
import { emptyWorklist, type LoadResult, type Worklist } from './types.js';

function manualClock(): DraftPersistClock & { flush: () => void } {
	const pending = new Map<unknown, () => void>();
	let nextId = 1;
	return {
		setTimeout(fn) {
			const id = nextId++;
			pending.set(id, fn);
			return id;
		},
		clearTimeout(id) {
			pending.delete(id);
		},
		flush() {
			const runs = [...pending.values()];
			pending.clear();
			for (const run of runs) run();
		}
	};
}

const sample: LoadResult = {
	org: 'feragame',
	activeConfigVersion: '1',
	configId: 'feragame-config',
	userId: 'u1',
	displayName: 'Ada',
	roles: ['administration'],
	worklist: {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				startLabel: 'Nuovo camion',
				taskIds: ['register', 'palletWeight'],
				tasks: [
					{ id: 'register', title: 'Registrazione', roles: ['administration'] },
					{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
				],
				instances: []
			},
			{
				jobId: 'lot_processing',
				canStart: false,
				repeatable: true,
				startLabel: 'Apri lotto',
				taskIds: ['startLot'],
				tasks: [{ id: 'startLot', title: 'Apri lotto', roles: ['logistic'] }],
				instances: []
			}
		]
	},
	taskFormBundles: []
};

describe('OrgSession', () => {
	test('hydrate loads Worklist via LoadDomainPort', async () => {
		const load = vi.fn(async () => sample) satisfies LoadDomainPort['load'];
		const session = new OrgSession({
			load: { load },
			hydrate: false
		});
		expect(session.result).toBeNull();
		await session.hydrate();
		expect(load).toHaveBeenCalledOnce();
		expect(session.error).toBeNull();
		expect(session.result?.org).toBe('feragame');
		expect(session.operator()).toEqual({
			userId: 'u1',
			displayName: 'Ada',
			org: 'feragame',
			roles: ['administration']
		});
		expect(session.jobIds).toEqual(['intake', 'lot_processing']);
		expect(session.worklistJob('intake')?.startLabel).toBe('Nuovo camion');
		const register = session.worklistJob('intake')?.tasks.find((t) => t.id === 'register');
		expect(register?.title).toBe('Registrazione');
		expect(register?.roles).toEqual(['administration']);
		const pallet = session.worklistJob('intake')?.tasks.find((t) => t.id === 'palletWeight');
		expect(pallet?.title).toBeTruthy();
		expect(pallet?.roles).toEqual(['logistic']);
	});

	test('hydrate surfaces port errors without leaving stale result', async () => {
		const session = new OrgSession({
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

	test('commit success replaces Worklist and clears Draft', async () => {
		const nextWorklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['register', 'codiceFir'],
					tasks: [],
					instances: [
						{
							id: 'NIR-1',
							label: 'NIR-1',
							tasks: { register: 'done' as const, codiceFir: 'available' as const }
						}
					]
				},
				{
					jobId: 'lot_processing',
					canStart: false,
					repeatable: true,
					taskIds: ['startLot'],
					tasks: [],
					instances: []
				}
			]
		};
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: nextWorklist,
			navigation: 'overview' as const,
			nextTaskFormBundle: null
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.setDraft('intake/register', 'NIR-1', { client: 'NO' });
		const outcome = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: { client: 'NO', palletCount: 1 },
			mint: { kind: 'none' }
		});
		expect(outcome?.jobInstanceId).toBe('NIR-1');
		expect(outcome?.navigation).toBe('overview');
		expect(outcome?.nextTaskFormBundle).toBeNull();
		expect(session.result?.worklist).toEqual(nextWorklist);
		expect(session.result?.taskFormBundles).toEqual([]);
		expect(session.getDraft('intake/register', 'NIR-1')).toBeNull();
		expect(session.commitError).toBeNull();
	});

	test('commit ignores concurrent second call while committing', async () => {
		let release!: (value: {
			org: string;
			jobInstanceId: string;
			worklist: typeof sample.worklist;
			navigation: 'overview';
			nextTaskFormBundle: null;
		}) => void;
		const commit = vi.fn(
			() =>
				new Promise<Parameters<typeof release>[0]>((resolve) => {
					release = resolve;
				})
		);
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		const req = {
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: { client: 'NO' },
			mint: { kind: 'none' as const }
		};
		const first = session.commit(req);
		const second = session.commit(req);
		expect(commit).toHaveBeenCalledTimes(1);
		expect(await second).toBeNull();
		release({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: sample.worklist,
			navigation: 'overview',
			nextTaskFormBundle: null
		});
		expect(await first).toMatchObject({ jobInstanceId: 'NIR-1', navigation: 'overview' });
		expect(commit).toHaveBeenCalledTimes(1);
	});


	test('provisional startingNew clears _new draft and returns response id', async () => {
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: emptyWorklist(),
			navigation: 'task' as const,
			nextTaskFormBundle: null
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.setDraft('intake/register', '', { client: 'NO' });
		const outcome = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: '',
			value: { client: 'NO' },
			mint: { kind: 'startingNew' }
		});
		expect(outcome?.jobInstanceId).toBe('NIR-1');
		expect(commit).toHaveBeenCalledWith({
			taskPath: 'intake/register',
			jobInstanceId: '',
			value: { client: 'NO' },
			mint: { kind: 'startingNew' }
		});
		expect(session.getDraft('intake/register', '')).toBeNull();
		expect(session.getDraft('intake/register', '_new')).toBeNull();
	});

	test('commit outcome surfaces navigation + nextTaskFormBundle for auto-advance', async () => {
		const next: TaskFormBundle = {
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
			sealFieldIds: [],
			bindings: { domain: {} }
		};
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: emptyWorklist(),
			navigation: 'task' as const,
			nextTaskFormBundle: next
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		const outcome = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: { client: 'NO' },
			mint: { kind: 'none' }
		});
		expect(outcome?.navigation).toBe('task');
		expect(outcome?.nextTaskFormBundle).toEqual(next);
	});

	test('navigation stay keeps HttpLinkStrip; clearLinksStay clears it', async () => {
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'T1',
			worklist: emptyWorklist(),
			navigation: 'stay' as const,
			continueNavigation: 'overview' as const,
			nextTaskFormBundle: null,
			links: [
				{
					label: 'Labels',
					url: 'https://example.test/labels.pdf',
					action: 'download' as const,
					filename: 'labels.pdf'
				}
			]
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		const outcome = await session.commit({
			taskPath: 'intake/lots/lots-1/register',
			jobInstanceId: 'T1',
			value: { lotId: 'L1' },
			mint: { kind: 'none' }
		});
		expect(outcome?.navigation).toBe('stay');
		expect(outcome?.continueNavigation).toBe('overview');
		expect(session.lastHttpLinks).toHaveLength(1);
		session.clearLinksStay();
		expect(session.lastHttpLinks).toEqual([]);
	});

	test('links alone do not invent stay; navigation stay + continueNavigation parks', async () => {
		const withLinksOverview = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'L1',
			worklist: emptyWorklist(),
			navigation: 'overview' as const,
			nextTaskFormBundle: null,
			links: [
				{
					label: 'Digital Product Passport',
					url: 'https://example.test/01/09506000134352',
					action: 'link' as const
				}
			],
			httpErrors: ['warn']
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit: withLinksOverview },
			hydrate: false
		});
		session.result = structuredClone(sample);
		await session.commit({
			taskPath: 'lot_processing/lock',
			jobInstanceId: 'L1',
			value: { ack: true },
			mint: { kind: 'none' }
		});
		expect(session.lastHttpLinks).toHaveLength(1);
		expect(session.lastHttpErrors).toEqual(['warn']);

		const stayCommit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'L1',
			worklist: emptyWorklist(),
			navigation: 'stay' as const,
			continueNavigation: 'overview' as const,
			nextTaskFormBundle: null,
			links: [
				{
					label: 'Digital Product Passport',
					url: 'https://example.test/01/09506000134352',
					action: 'link' as const
				}
			],
			httpErrors: ['warn']
		}));
		const staySession = new OrgSession({
			load: { load: async () => sample },
			commit: { commit: stayCommit },
			hydrate: false
		});
		staySession.result = structuredClone(sample);
		const outcome = await staySession.commit({
			taskPath: 'lot_processing/lock',
			jobInstanceId: 'L1',
			value: { ack: true },
			mint: { kind: 'none' }
		});
		expect(outcome?.navigation).toBe('stay');
		expect(outcome?.continueNavigation).toBe('overview');
		expect(staySession.lastHttpLinks).toHaveLength(1);
		staySession.clearLinksStay();
		expect(staySession.lastHttpLinks).toEqual([]);
		expect(staySession.lastHttpErrors).toEqual([]);
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
					tasks: [],
					instances: []
				}
			]
		};
		const load = vi.fn(async () => reloaded);
		const commit = vi.fn(async () => {
			throw new CommitDomainError('forbidden', 403);
		});
		const session = new OrgSession({
			load: { load },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.setDraft('intake/register', '_new', { client: 'NO', notes: 'keep me' });

		const outcome = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: { client: 'NO' },
			mint: { kind: 'none' }
		});
		expect(outcome).toBeNull();
		expect(session.commitError).toBe('forbidden');
		expect(session.getDraft('intake/register', '_new')).toEqual({
			client: 'NO',
			notes: 'keep me'
		});
		expect(session.hasDirtyDraft('intake/register', '_new')).toBe(true);
		expect(load).toHaveBeenCalledOnce();
		expect(session.result?.worklist).toEqual(reloaded.worklist);
	});

	test('seedDraft is non-dirty; setDraft dirties; clearDraft clears both', () => {
		const session = new OrgSession({
			load: { load: async () => sample },
			hydrate: false
		});
		const path = 'intake/register';
		const key = 'NIR-1';

		session.seedDraft(path, key, { client: 'NO' });
		expect(session.getDraft(path, key)).toEqual({ client: 'NO' });
		expect(session.hasDirtyDraft(path, key)).toBe(false);

		session.setDraft(path, key, { client: 'NO', notes: 'typed' });
		expect(session.getDraft(path, key)).toEqual({ client: 'NO', notes: 'typed' });
		expect(session.hasDirtyDraft(path, key)).toBe(true);

		// Dirty sticks: later seed must not undirty.
		session.seedDraft(path, key, { client: 'RESET' });
		expect(session.getDraft(path, key)).toEqual({ client: 'RESET' });
		expect(session.hasDirtyDraft(path, key)).toBe(true);

		session.clearDraft(path, key);
		expect(session.getDraft(path, key)).toBeNull();
		expect(session.hasDirtyDraft(path, key)).toBe(false);
	});

	test('refreshKeepDrafts keeps seed and dirty Drafts in memory', async () => {
		const load = vi.fn(async () => sample);
		const session = new OrgSession({
			load: { load },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.seedDraft('intake/register', 'NIR-1', { client: 'seed' });
		session.setDraft('intake/register', '_new', { client: 'dirty' });

		await session.refreshKeepDrafts();

		expect(session.getDraft('intake/register', 'NIR-1')).toEqual({ client: 'seed' });
		expect(session.hasDirtyDraft('intake/register', 'NIR-1')).toBe(false);
		expect(session.getDraft('intake/register', '_new')).toEqual({ client: 'dirty' });
		expect(session.hasDirtyDraft('intake/register', '_new')).toBe(true);
		expect(load).toHaveBeenCalledOnce();
	});

	test('successful commit clears dirty Draft via clearDraft', async () => {
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: sample.worklist,
			navigation: 'overview' as const,
			nextTaskFormBundle: null
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		session.setDraft('intake/register', 'NIR-1', { client: 'NO' });
		expect(session.hasDirtyDraft('intake/register', 'NIR-1')).toBe(true);

		await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: { client: 'NO' },
			mint: { kind: 'none' }
		});

		expect(session.getDraft('intake/register', 'NIR-1')).toBeNull();
		expect(session.hasDirtyDraft('intake/register', 'NIR-1')).toBe(false);
	});

	test('dirty Drafts debounce to storage; seed never persists', async () => {
		const storage = memoryStorage();
		const clock = manualClock();
		const session = new OrgSession({
			load: { load: async () => sample },
			hydrate: false,
			storage,
			clock,
			debounceMs: DEFAULT_DRAFT_DEBOUNCE_MS,
			flushOnHide: false
		});
		await session.hydrate();
		const path = 'intake/register';
		const bagKey = 'NIR-1';
		const key = draftKey(path, bagKey);
		const storageKey = orgDraftStorageKey('feragame', 'u1', key);

		session.seedDraft(path, bagKey, { client: 'seed' });
		clock.flush();
		expect(storage.getItem(storageKey)).toBeNull();
		expect(session.hasDirtyDraft(path, bagKey)).toBe(false);

		session.setDraft(path, bagKey, { client: 'dirty' });
		expect(storage.getItem(storageKey)).toBeNull();
		clock.flush();
		expect(readStoredDraft(storage, storageKey)).toEqual({ client: 'dirty' });
		expect(session.hasDirtyDraft(path, bagKey)).toBe(true);
	});

	test('rehydrate on identity known; refreshKeepDrafts does not clobber memory', async () => {
		const bagKey = draftKey('intake/register', 'NIR-1');
		const storage = memoryStorage({
			[orgDraftStorageKey('feragame', 'u1', bagKey)]: JSON.stringify({
				client: 'from-storage'
			})
		});
		const load = vi.fn(async () => sample);
		const session = new OrgSession({
			load: { load },
			hydrate: false,
			storage,
			debounceMs: 0,
			flushOnHide: false
		});

		expect(session.getDraft('intake/register', 'NIR-1')).toBeNull();
		await session.hydrate();
		expect(session.getDraft('intake/register', 'NIR-1')).toEqual({ client: 'from-storage' });
		expect(session.hasDirtyDraft('intake/register', 'NIR-1')).toBe(true);

		session.setDraft('intake/register', 'NIR-1', { client: 'in-memory-wins' });
		await session.refreshKeepDrafts();
		expect(session.getDraft('intake/register', 'NIR-1')).toEqual({
			client: 'in-memory-wins'
		});
		expect(load).toHaveBeenCalledTimes(2);
	});

	test('flushPendingDrafts writes pending debounce; commit clears memory and storage', async () => {
		const storage = memoryStorage();
		const clock = manualClock();
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: sample.worklist,
			navigation: 'overview' as const,
			nextTaskFormBundle: null
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false,
			storage,
			clock,
			debounceMs: DEFAULT_DRAFT_DEBOUNCE_MS,
			flushOnHide: false
		});
		await session.hydrate();
		const path = 'intake/register';
		const bagKey = 'NIR-1';
		const storageKey = orgDraftStorageKey(
			'feragame',
			'u1',
			draftKey(path, bagKey)
		);

		session.setDraft(path, bagKey, { client: 'pending' });
		expect(storage.getItem(storageKey)).toBeNull();
		session.flushPendingDrafts();
		expect(readStoredDraft(storage, storageKey)).toEqual({ client: 'pending' });

		await session.commit({
			taskPath: path,
			jobInstanceId: bagKey,
			value: { client: 'pending' },
			mint: { kind: 'none' }
		});
		expect(session.getDraft(path, bagKey)).toBeNull();
		expect(storage.getItem(storageKey)).toBeNull();
	});

	test('identity switch clears memory bag; other users storage keys untouched', async () => {
		const storage = memoryStorage();
		const adaKey = orgDraftStorageKey(
			'feragame',
			'u1',
			draftKey('intake/register', 'NIR-1')
		);
		const otherKey = orgDraftStorageKey(
			'other-org',
			'u9',
			draftKey('intake/register', 'OTHER')
		);
		storage.setItem(otherKey, JSON.stringify({ client: 'untouched' }));

		let current = structuredClone(sample);
		const session = new OrgSession({
			load: { load: async () => current },
			hydrate: false,
			storage,
			debounceMs: 0,
			flushOnHide: false
		});
		await session.hydrate();
		session.setDraft('intake/register', 'NIR-1', { client: 'ada' });
		expect(session.hasDirtyDraft('intake/register', 'NIR-1')).toBe(true);
		expect(readStoredDraft(storage, adaKey)).toEqual({ client: 'ada' });

		current = {
			...structuredClone(sample),
			org: 'other-org',
			userId: 'u2',
			displayName: 'Bob'
		};
		await session.hydrate();
		expect(session.getDraft('intake/register', 'NIR-1')).toBeNull();
		expect(session.hasDirtyDraft('intake/register', 'NIR-1')).toBe(false);
		// Prior user's dirty key remains in storage; unrelated user key untouched.
		expect(readStoredDraft(storage, adaKey)).toEqual({ client: 'ada' });
		expect(readStoredDraft(storage, otherKey)).toEqual({ client: 'untouched' });
	});

	test('session has no Spawn/Close surface (incl. stay-after-Commit)', async () => {
		const absent = [
			'closeGroup',
			'closePort',
			'spawnGroup',
			'spawnPort',
			'canSpawn',
			'canClose'
		] as const;
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: sample.worklist,
			navigation: 'stay' as const,
			continueNavigation: 'overview' as const,
			nextTaskFormBundle: null,
			links: []
		}));
		const session = new OrgSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false,
			storage: null,
			flushOnHide: false
		});
		session.result = structuredClone(sample);
		for (const key of absent) {
			expect(session).not.toHaveProperty(key);
		}
		const outcome = await session.commit({
			taskPath: 'intake/register',
			jobInstanceId: 'NIR-1',
			value: {},
			mint: { kind: 'none' }
		});
		expect(outcome?.navigation).toBe('stay');
		expect(session).not.toHaveProperty('linksStayActive');
		for (const key of absent) {
			expect(session).not.toHaveProperty(key);
		}
		expect(JSON.stringify(session.result?.worklist)).not.toMatch(
			/canSpawn|canClose|"groups"|spawnGroup|closeGroup/
		);
	});

	test('draftAddressForTask mark binding matches hasDirtyDraft for flat and provisional Start', () => {
		const session = new OrgSession({
			load: { load: async () => sample },
			hydrate: false,
			storage: null,
			flushOnHide: false
		});
		const flat = draftAddressForTask({
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: 'NIR-1'
		});
		session.setDraft(flat.taskPath, flat.draftBagInstanceKey, { client: 'NO' });
		expect(session.hasDirtyDraft(flat.taskPath, flat.draftBagInstanceKey)).toBe(true);

		const provisional = draftAddressForTask({
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: ''
		});
		expect(provisional.taskPath).toBe('intake/register');
		expect(provisional.draftBagInstanceKey).toBe('');
		session.setDraft(provisional.taskPath, provisional.draftBagInstanceKey, {
			client: 'NO'
		});
		expect(
			session.hasDirtyDraft(provisional.taskPath, provisional.draftBagInstanceKey)
		).toBe(true);
	});
});
