import { describe, expect, test, vi } from 'vitest';

import { CommitDomainError } from './commit-port.js';
import { draftAddressForTask, draftKey } from './drafts.js';
import {
	livingDraftStorageKey,
	memoryStorage,
	readStoredDraft
} from './draft-storage.js';
import type { LoadDomainPort } from './load-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import {
	DEFAULT_DRAFT_DEBOUNCE_MS,
	LivingSession,
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
				jobId: 'line',
				canStart: false,
				repeatable: true,
				startLabel: 'Nuovo turno',
				taskIds: ['shift'],
				tasks: [{ id: 'shift', title: 'Turno', roles: ['line'] }],
				instances: []
			}
		]
	},
	taskFormBundles: []
};

describe('LivingSession', () => {
	test('hydrate loads Worklist via LoadDomainPort', async () => {
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
		expect(session.operator()).toEqual({
			userId: 'u1',
			displayName: 'Ada',
			org: 'feragame',
			roles: ['administration']
		});
		expect(session.jobIds).toEqual(['intake', 'line']);
		expect(session.worklistJob('intake')?.startLabel).toBe('Nuovo camion');
		const register = session.worklistJob('intake')?.tasks.find((t) => t.id === 'register');
		expect(register?.title).toBe('Registrazione');
		expect(register?.roles).toEqual(['administration']);
		const pallet = session.worklistJob('intake')?.tasks.find((t) => t.id === 'palletWeight');
		expect(pallet?.title).toBeTruthy();
		expect(pallet?.roles).toEqual(['logistic']);
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
				{ jobId: 'line', canStart: false, repeatable: true, taskIds: ['shift'], tasks: [], instances: [] }
			]
		};
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: nextWorklist,
			next: 'overview' as const,
			nextTaskFormBundle: null
		}));
		const session = new LivingSession({
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
		expect(outcome?.next).toBe('overview');
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
			next: 'overview';
			nextTaskFormBundle: null;
		}) => void;
		const commit = vi.fn(
			() =>
				new Promise<Parameters<typeof release>[0]>((resolve) => {
					release = resolve;
				})
		);
		const session = new LivingSession({
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
			next: 'overview',
			nextTaskFormBundle: null
		});
		expect(await first).toMatchObject({ jobInstanceId: 'NIR-1', next: 'overview' });
		expect(commit).toHaveBeenCalledTimes(1);
	});

	test('spawningNew clears nested provisional draft and returns overview next', async () => {
		const nextWorklist: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['lots'],
					tasks: [{ id: 'lots', title: 'Lotti', roles: ['administration'] }],
					instances: [
						{
							id: 'T1',
							label: 'T1',
							tasks: { lots: 'available' },
							groups: {
								lots: {
									groupId: 'lots',
									canSpawn: true,
									canClose: true,
									repeatable: true,
									childTaskIds: ['register'],
									instances: [
										{
											id: 'lots-1',
											tasks: { register: 'done' }
										}
									]
								}
							}
						}
					]
				}
			]
		};
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'T1',
			worklist: nextWorklist,
			next: 'overview' as const,
			nextTaskFormBundle: null
		}));
		const session = new LivingSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		const bagKey = 'T1::lots::';
		session.setDraft('intake/lots/new/register', bagKey, { lotId: 'L1' });
		const outcome = await session.commit({
			taskPath: 'intake/lots/new/register',
			jobInstanceId: 'T1',
			value: { lotId: 'L1' },
			mint: { kind: 'spawningNew' },
			draftBagInstanceKey: bagKey
		});
		expect(outcome?.next).toBe('overview');
		expect(outcome?.jobInstanceId).toBe('T1');
		expect(session.result?.worklist).toEqual(nextWorklist);
		expect(session.getDraft('intake/lots/new/register', bagKey)).toBeNull();
		expect(commit).toHaveBeenCalledWith({
			taskPath: 'intake/lots/new/register',
			jobInstanceId: 'T1',
			value: { lotId: 'L1' },
			mint: { kind: 'spawningNew' }
		});
	});

	test('provisional startingNew clears _new draft and returns response id', async () => {
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'NIR-1',
			worklist: emptyWorklist(),
			next: 'task' as const,
			nextTaskFormBundle: null
		}));
		const session = new LivingSession({
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

	test('commit outcome surfaces next + nextTaskFormBundle for auto-advance', async () => {
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
			next: 'task' as const,
			nextTaskFormBundle: next
		}));
		const session = new LivingSession({
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
		expect(outcome?.next).toBe('task');
		expect(outcome?.nextTaskFormBundle).toEqual(next);
		expect(session.linksStayActive).toBe(false);
	});

	test('commit with download links sets linksStayActive; clearLinksStay resets', async () => {
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'T1',
			worklist: emptyWorklist(),
			next: 'overview' as const,
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
		const session = new LivingSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		await session.commit({
			taskPath: 'intake/lots/lots-1/register',
			jobInstanceId: 'T1',
			value: { lotId: 'L1' },
			mint: { kind: 'none' }
		});
		expect(session.linksStayActive).toBe(true);
		expect(session.lastHttpLinks).toHaveLength(1);
		session.clearLinksStay();
		expect(session.linksStayActive).toBe(false);
		expect(session.lastHttpLinks).toEqual([]);
	});

	test('link-only commit leaves stay inactive; clearLinksStay drops strip state', async () => {
		const commit = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'L1',
			worklist: emptyWorklist(),
			next: 'overview' as const,
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
		const session = new LivingSession({
			load: { load: async () => sample },
			commit: { commit },
			hydrate: false
		});
		session.result = structuredClone(sample);
		await session.commit({
			taskPath: 'passport/lock',
			jobInstanceId: 'L1',
			value: { ack: true },
			mint: { kind: 'none' }
		});
		expect(session.linksStayActive).toBe(false);
		expect(session.lastHttpLinks).toHaveLength(1);
		expect(session.lastHttpErrors).toEqual(['warn']);
		session.clearLinksStay();
		expect(session.lastHttpLinks).toEqual([]);
		expect(session.lastHttpErrors).toEqual([]);
		expect(session.linksStayActive).toBe(false);
	});

	test('fetchTaskForm delegates to TaskFormPort without touching home bulk', async () => {
		const bundle: TaskFormBundle = {
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
		const fetch = vi.fn(async () => bundle);
		const session = new LivingSession({
			load: { load: async () => sample },
			taskForm: { fetch },
			hydrate: false
		});
		session.result = structuredClone(sample);
		const priorBundles = session.result.taskFormBundles;
		const got = await session.fetchTaskForm({ jobId: 'intake', taskId: 'register' });
		expect(got).toEqual(bundle);
		expect(fetch).toHaveBeenCalledWith({ jobId: 'intake', taskId: 'register' });
		expect(session.result?.taskFormBundles).toBe(priorBundles);
	});

	test('closeGroup success replaces Worklist and returns next for advance', async () => {
		const afterClose: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['lots', 'weightCheck'],
					tasks: [
						{ id: 'lots', title: 'Lotti', roles: ['administration'] },
						{ id: 'weightCheck', title: 'Peso', roles: ['logistic'] }
					],
					instances: [
						{
							id: 'T1',
							label: 'T1',
							tasks: { lots: 'done', weightCheck: 'actionable' },
							groups: {
								lots: {
									groupId: 'lots',
									canSpawn: false,
									canClose: false,
									repeatable: true,
									childTaskIds: ['register'],
									instances: [
										{
											id: 'lots-1',
											tasks: { register: 'done' }
										}
									]
								}
							}
						}
					]
				}
			]
		};
		const nextBundle: TaskFormBundle = {
			jobId: 'intake',
			taskId: 'weightCheck',
			jobInstanceId: 'T1',
			title: 'Peso',
			surface: 'live',
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: [],
			bindings: { domain: {} }
		};
		const close = vi.fn(async () => ({
			org: 'feragame',
			jobInstanceId: 'T1',
			worklist: afterClose,
			next: 'task' as const,
			nextTaskFormBundle: nextBundle
		}));
		const session = new LivingSession({
			load: { load: async () => sample },
			close: { close },
			hydrate: false
		});
		session.result = structuredClone(sample);
		const outcome = await session.closeGroup({
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots'
		});
		expect(outcome?.next).toBe('task');
		expect(outcome?.nextTaskFormBundle?.taskId).toBe('weightCheck');
		expect(session.result?.worklist).toEqual(afterClose);
		expect(session.result?.worklist.jobs[0]?.instances[0]?.tasks.weightCheck).toBe(
			'actionable'
		);
		expect(session.commitError).toBeNull();
		expect(close).toHaveBeenCalledWith({
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots'
		});
	});

	test('closeGroup failure keeps board, re-Loads, and preserves stay links', async () => {
		const reloaded = structuredClone(sample);
		const load = vi.fn(async () => reloaded);
		const close = vi.fn(async () => {
			throw new CommitDomainError('cannot close', 400);
		});
		const session = new LivingSession({
			load: { load },
			close: { close },
			hydrate: false
		});
		session.result = structuredClone(sample);
		const stayLinks = [
			{
				label: 'Labels',
				url: 'https://example.test/labels.pdf',
				action: 'download' as const,
				filename: 'labels.pdf'
			}
		];
		session.lastHttpLinks = stayLinks;
		session.lastHttpErrors = [];
		session.linksStayActive = true;
		const outcome = await session.closeGroup({
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots'
		});
		expect(outcome).toBeNull();
		expect(session.commitError).toBe('cannot close');
		expect(session.linksStayActive).toBe(true);
		expect(session.lastHttpLinks).toEqual(stayLinks);
		expect(load).toHaveBeenCalledOnce();
		expect(session.result?.worklist).toEqual(reloaded.worklist);
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
		const session = new LivingSession({
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
		const session = new LivingSession({
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
		const session = new LivingSession({
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
			next: 'overview' as const,
			nextTaskFormBundle: null
		}));
		const session = new LivingSession({
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
		const session = new LivingSession({
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
		const storageKey = livingDraftStorageKey('feragame', 'u1', key);

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
			[livingDraftStorageKey('feragame', 'u1', bagKey)]: JSON.stringify({
				client: 'from-storage'
			})
		});
		const load = vi.fn(async () => sample);
		const session = new LivingSession({
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
			next: 'overview' as const,
			nextTaskFormBundle: null
		}));
		const session = new LivingSession({
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
		const storageKey = livingDraftStorageKey(
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
		const adaKey = livingDraftStorageKey(
			'feragame',
			'u1',
			draftKey('intake/register', 'NIR-1')
		);
		const otherKey = livingDraftStorageKey(
			'other-org',
			'u9',
			draftKey('intake/register', 'OTHER')
		);
		storage.setItem(otherKey, JSON.stringify({ client: 'untouched' }));

		let current = structuredClone(sample);
		const session = new LivingSession({
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

	test('draftAddressForTask mark binding matches hasDirtyDraft for flat and provisional', () => {
		const session = new LivingSession({
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
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'new'
		});
		expect(provisional.taskPath).toBe('intake/lots/new/register');
		expect(provisional.draftBagInstanceKey).toBe('T1::lots::');
		session.setDraft(
			provisional.taskPath,
			provisional.draftBagInstanceKey,
			{ lotId: 'L1' }
		);
		expect(
			session.hasDirtyDraft(provisional.taskPath, provisional.draftBagInstanceKey)
		).toBe(true);
	});
});
