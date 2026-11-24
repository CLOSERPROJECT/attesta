import { describe, expect, test, vi } from 'vitest';

import type { TaskFormBundle } from '../form/task-form-bundle.js';
import { normalizeTaskAddress, taskAddressKey } from '../task-address.js';
import type { TimelineLoadQuery, TimelinePort } from '../timeline/timeline-port.js';
import type { TimelineView } from '../timeline/types.js';
import { createBoardFormCache } from './board-form-cache.svelte.js';
import { createBoardRecapCache } from './board-recap-cache.svelte.js';
import { boardScopeKey, createBoardSession, type BoardScope } from './board-session.svelte.js';
import { BoardTimelineModel } from './board-timeline-model.svelte.js';
import type { BoardTaskFocus } from './board-url.js';

function bootView(stream = 'recycling'): TimelineView {
	return {
		stream,
		instance: '',
		mode: 'boot',
		label: stream,
		status: 'active',
		steps: [
			{
				stepId: '1',
				title: 'Intake',
				substepCount: 1,
				expanded: true,
				substeps: [
					{
						substepId: 'intake.truckArrive',
						title: 'Truck arrive',
						status: 'available',
						orgActionable: true,
						href: `/app/${stream}/new/intake/new/truckArrive`
					}
				]
			}
		]
	};
}

function instanceView(stream = 'recycling', instance = '261102-1'): TimelineView {
	return {
		stream,
		instance,
		mode: 'instance',
		label: 'Camion',
		status: 'active',
		steps: [
			{
				stepId: '1',
				title: 'Intake',
				substepCount: 1,
				expanded: true,
				substeps: [
					{
						substepId: 'intake.register',
						title: 'Register',
						status: 'available',
						orgActionable: true,
						href: `/app/${stream}/i/${instance}/intake/${instance}/register`
					}
				]
			}
		]
	};
}

function fakeTimeline(
	impl: (query: TimelineLoadQuery) => Promise<TimelineView>
): TimelinePort & { loads: TimelineLoadQuery[] } {
	const loads: TimelineLoadQuery[] = [];
	return {
		loads,
		async load(query) {
			loads.push(query);
			return impl(query);
		}
	};
}

function harness(
	scope: BoardScope,
	timeline: TimelinePort,
	extras: {
		formFetch?: ReturnType<typeof vi.fn>;
		dropSeededDraft?: ReturnType<typeof vi.fn>;
	} = {}
) {
	const formFetch =
		extras.formFetch ??
		vi.fn(async () => {
			throw new Error('unexpected form fetch');
		});
	const formCache = createBoardFormCache({
		taskForm: { fetch: formFetch }
	});
	const recapCache = createBoardRecapCache({
		taskRecap: { fetch: vi.fn(async () => ({ items: [] })) }
	});
	const model = new BoardTimelineModel({ syncUrl: () => {} });
	const navigateToInstance = vi.fn(async () => {});
	const hydrate = vi.fn();
	const dropSeededDraft = extras.dropSeededDraft ?? vi.fn();
	const session = createBoardSession({
		scope,
		timeline,
		model,
		formCache,
		recapCache,
		navigateToInstance,
		hydrate,
		dropSeededDraft
	});
	return {
		session,
		model,
		formCache,
		recapCache,
		navigateToInstance,
		hydrate,
		dropSeededDraft,
		formFetch
	};
}

describe('boardScopeKey', () => {
	test('encodes boot and instance distinctly', () => {
		expect(boardScopeKey({ kind: 'boot', stream: 'recycling' })).toBe('recycling||boot');
		expect(
			boardScopeKey({
				kind: 'instance',
				stream: 'recycling',
				instance: '261102-1'
			})
		).toBe('recycling|261102-1|instance');
	});
});

describe('BoardSession', () => {
	test('load boot uses TimelineLoadQuery { stream } only', async () => {
		const view = bootView();
		const timeline = fakeTimeline(async (q) => {
			expect(q).toEqual({ stream: 'recycling' });
			expect('instance' in q).toBe(false);
			return view;
		});
		const { session, model } = harness({ kind: 'boot', stream: 'recycling' }, timeline);

		await session.load();

		expect(timeline.loads).toEqual([{ stream: 'recycling' }]);
		expect(session.view).toEqual(view);
		expect(session.loading).toBe(false);
		expect(session.error).toBe('');
		expect(session.scopeKey).toBe('recycling||boot');
		expect(model.view).toEqual(view);
	});

	test('load instance uses TimelineLoadQuery { stream, instance }', async () => {
		const view = instanceView();
		const timeline = fakeTimeline(async () => view);
		const { session, hydrate } = harness(
			{ kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline
		);

		await session.load();

		expect(timeline.loads).toEqual([{ stream: 'recycling', instance: '261102-1' }]);
		expect(session.view).toEqual(view);
		expect(hydrate).toHaveBeenCalledWith({
			stream: 'recycling',
			instance: '261102-1'
		});
	});

	test('reload refreshes with the same query and keeps view on failure', async () => {
		const first = instanceView();
		const second = { ...first, label: 'Updated' };
		let n = 0;
		const timeline = fakeTimeline(async () => {
			n += 1;
			if (n === 1) return first;
			if (n === 2) return second;
			throw new Error('boom');
		});
		const { session, formCache } = harness(
			{ kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline
		);

		await session.load();
		const readyAddr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		formCache.putReady(readyAddr, {
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: '261102-1',
			title: 'Register',
			surface: 'live',
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: [],
			bindings: { domain: {} }
		});

		await session.reload();
		expect(session.view?.label).toBe('Updated');
		expect(formCache.get(taskAddressKey(readyAddr))?.status).toBe('ready');

		await session.reload();
		expect(session.view?.label).toBe('Updated');
		expect(timeline.loads).toHaveLength(3);
	});

	test('adoptInstance stages putReady and soft-navigates to instance', async () => {
		const timeline = fakeTimeline(async () => bootView());
		const { session, formCache, navigateToInstance } = harness(
			{ kind: 'boot', stream: 'recycling' },
			timeline
		);
		await session.load();

		const nextAddr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		const bundle: TaskFormBundle = {
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: '261102-1',
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
		const focus: BoardTaskFocus = {
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		};

		await session.adoptInstance('261102-1', {
			focus,
			bundleAddr: nextAddr,
			bundle
		});

		expect(formCache.get(taskAddressKey(nextAddr))?.status).toBe('ready');
		expect(navigateToInstance).toHaveBeenCalledWith({
			stream: 'recycling',
			instance: '261102-1',
			focus
		});
		expect(session.scope).toEqual({
			kind: 'instance',
			stream: 'recycling',
			instance: '261102-1'
		});
		expect(session.scopeKey).toBe('recycling|261102-1|instance');
	});

	test('load surfaces port errors without a view', async () => {
		const timeline = fakeTimeline(async () => {
			throw new Error('timeline down');
		});
		const { session } = harness({ kind: 'boot', stream: 'recycling' }, timeline);

		await session.load();

		expect(session.view).toBeNull();
		expect(session.loading).toBe(false);
		expect(session.error).toBe('timeline down');
	});

	test('domainChanged stay reloads timeline and hydrates worklist', async () => {
		const first = instanceView();
		const second = { ...first, label: 'After commit' };
		let n = 0;
		const timeline = fakeTimeline(async () => {
			n += 1;
			return n === 1 ? first : second;
		});
		const { session, hydrate } = harness(
			{ kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline
		);
		await session.load();
		hydrate.mockClear();

		const addr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		await session.domainChanged({
			origin: 'own',
			revision: 2,
			addr,
			follow: 'stay',
			submitMany: false
		});

		expect(session.view?.label).toBe('After commit');
		expect(timeline.loads).toHaveLength(2);
		expect(hydrate).toHaveBeenCalledWith({
			stream: 'recycling',
			instance: '261102-1'
		});
	});

	test('domainChanged stay submitMany refetches ready form bundles', async () => {
		const formFetch = vi.fn(async () => ({
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: '261102-1',
			title: 'Register refreshed',
			surface: 'live' as const,
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: [],
			bindings: { domain: { x: 1 } }
		}));
		const timeline = fakeTimeline(async () => instanceView());
		const formCache = createBoardFormCache({ taskForm: { fetch: formFetch } });
		const recapCache = createBoardRecapCache({
			taskRecap: { fetch: vi.fn(async () => []) }
		});
		const model = new BoardTimelineModel({ syncUrl: () => {} });
		const session = createBoardSession({
			scope: { kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline,
			model,
			formCache,
			recapCache,
			navigateToInstance: vi.fn(async () => {})
		});
		await session.load();

		const readyAddr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		formCache.putReady(readyAddr, {
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: '261102-1',
			title: 'Register',
			surface: 'live',
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: [],
			bindings: { domain: {} }
		});

		await session.domainChanged({
			origin: 'own',
			revision: 2,
			addr: readyAddr,
			follow: 'stay',
			submitMany: true
		});
		await vi.waitFor(() => {
			expect(formCache.get(taskAddressKey(readyAddr))?.status).toBe('ready');
			expect(formCache.get(taskAddressKey(readyAddr))?.bundle?.title).toBe('Register refreshed');
		});
		expect(formFetch).toHaveBeenCalled();
	});

	test('domainChanged stay without submitMany leaves form bundles ready', async () => {
		const timeline = fakeTimeline(async () => instanceView());
		const { session, formCache } = harness(
			{ kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline
		);
		await session.load();
		const readyAddr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		formCache.putReady(readyAddr, {
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: '261102-1',
			title: 'Register',
			surface: 'live',
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: [],
			bindings: { domain: {} }
		});

		await session.domainChanged({
			origin: 'own',
			revision: 2,
			addr: readyAddr,
			follow: 'stay',
			submitMany: false
		});

		expect(formCache.get(taskAddressKey(readyAddr))?.status).toBe('ready');
		expect(formCache.get(taskAddressKey(readyAddr))?.bundle?.title).toBe('Register');
	});

	test('domainChanged advance invalidates and re-enqueues that recap', async () => {
		const recap = (n: number) => ({
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register',
			timestamp: `2026-01-0${n}T00:00:00Z`,
			userId: 'u1',
			facts: { n },
			links: []
		});
		const recapFetch = vi.fn(async () => ({ items: [recap(2)] }));
		const timeline = fakeTimeline(async () => instanceView());
		const formCache = createBoardFormCache({
			taskForm: {
				fetch: vi.fn(async () => {
					throw new Error('unexpected form fetch');
				})
			}
		});
		const recapCache = createBoardRecapCache({
			taskRecap: { fetch: recapFetch }
		});
		const model = new BoardTimelineModel({ syncUrl: () => {} });
		const session = createBoardSession({
			scope: { kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline,
			model,
			formCache,
			recapCache,
			navigateToInstance: vi.fn(async () => {})
		});
		await session.load();

		const addr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		recapCache.putReady(addr, [recap(1)]);
		expect(recapCache.get(taskAddressKey(addr))?.items[0]?.facts.n).toBe(1);

		await session.domainChanged({
			origin: 'own',
			revision: 2,
			addr,
			follow: 'advance',
			submitMany: false
		});
		await vi.waitFor(() => {
			expect(recapCache.get(taskAddressKey(addr))?.status).toBe('ready');
			expect(recapCache.get(taskAddressKey(addr))?.items[0]?.facts.n).toBe(2);
		});
		expect(recapFetch).toHaveBeenCalled();
	});

	test('domainChanged leaving refreshes only the recap', async () => {
		const recap = (n: number) => ({
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register',
			timestamp: `2026-01-0${n}T00:00:00Z`,
			userId: 'u1',
			facts: { n },
			links: []
		});
		const recapFetch = vi.fn(async () => ({ items: [recap(2)] }));
		const timeline = fakeTimeline(async () => instanceView());
		const formCache = createBoardFormCache({
			taskForm: {
				fetch: vi.fn(async () => {
					throw new Error('unexpected form fetch');
				})
			}
		});
		const recapCache = createBoardRecapCache({
			taskRecap: { fetch: recapFetch }
		});
		const model = new BoardTimelineModel({ syncUrl: () => {} });
		const hydrate = vi.fn();
		const session = createBoardSession({
			scope: { kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline,
			model,
			formCache,
			recapCache,
			navigateToInstance: vi.fn(async () => {}),
			hydrate
		});
		await session.load();
		hydrate.mockClear();

		const addr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		recapCache.putReady(addr, [recap(1)]);

		await session.domainChanged({
			origin: 'own',
			revision: 2,
			addr,
			follow: 'leaving',
			submitMany: false
		});

		await vi.waitFor(() => {
			expect(recapCache.get(taskAddressKey(addr))?.status).toBe('ready');
			expect(recapCache.get(taskAddressKey(addr))?.items[0]?.facts.n).toBe(2);
		});
		expect(recapFetch).toHaveBeenCalled();
		expect(timeline.loads).toHaveLength(1);
		expect(hydrate).not.toHaveBeenCalled();
	});

	test('domainChanged advance without a view hydrates on instance scope', async () => {
		const timeline = fakeTimeline(async () => instanceView());
		const { session, hydrate } = harness(
			{ kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline
		);
		const addr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});

		await session.domainChanged({
			origin: 'own',
			revision: 2,
			addr,
			follow: 'advance',
			submitMany: false
		});

		expect(timeline.loads).toHaveLength(0);
		expect(hydrate).toHaveBeenCalledWith({
			stream: 'recycling',
			instance: '261102-1'
		});
	});

	test('domainChanged advance without a view does nothing on boot scope', async () => {
		const timeline = fakeTimeline(async () => bootView());
		const { session, hydrate } = harness({ kind: 'boot', stream: 'recycling' }, timeline);
		const addr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});

		await session.domainChanged({
			origin: 'own',
			revision: 2,
			addr,
			follow: 'advance',
			submitMany: false
		});

		expect(timeline.loads).toHaveLength(0);
		expect(hydrate).not.toHaveBeenCalled();
	});

	test('domainChanged foreign on instance revalidates open forms as stale', async () => {
		const staleBundle: TaskFormBundle = {
			jobId: 'intake',
			taskId: 'register',
			jobInstanceId: '261102-1',
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
		const freshBundle: TaskFormBundle = {
			...staleBundle,
			title: 'Register foreign'
		};
		let resolveForm!: (b: TaskFormBundle) => void;
		const formFetch = vi.fn(
			() =>
				new Promise<TaskFormBundle>((resolve) => {
					resolveForm = resolve;
				})
		);
		const first = instanceView();
		const second = { ...first, label: 'After foreign' };
		let n = 0;
		const timeline = fakeTimeline(async () => {
			n += 1;
			return n === 1 ? first : second;
		});
		const dropSeededDraft = vi.fn();
		const { session, model, formCache, hydrate } = harness(
			{ kind: 'instance', stream: 'recycling', instance: '261102-1' },
			timeline,
			{ formFetch, dropSeededDraft }
		);
		await session.load();
		hydrate.mockClear();

		const addr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'intake',
			jobInstanceId: '261102-1',
			taskId: 'register'
		});
		formCache.putReady(addr, staleBundle);
		model.setBodyOpen(addr, true);

		const changed = session.domainChanged({ origin: 'foreign', revision: 2 });
		expect(formCache.get(taskAddressKey(addr))?.status).toBe('ready');
		expect(formCache.get(taskAddressKey(addr))?.bundle).toBe(staleBundle);

		resolveForm(freshBundle);
		await changed;
		await vi.waitFor(() => {
			expect(formCache.get(taskAddressKey(addr))?.stale).toBe(true);
		});
		expect(formCache.get(taskAddressKey(addr))?.bundle).toBe(staleBundle);
		expect(session.view?.label).toBe('After foreign');
		expect(hydrate).toHaveBeenCalledWith({
			stream: 'recycling',
			instance: '261102-1',
			background: true
		});

		session.acceptStale(addr);
		expect(dropSeededDraft).toHaveBeenCalledWith(addr);
		expect(formCache.get(taskAddressKey(addr))?.bundle).toBe(freshBundle);
		expect(formCache.get(taskAddressKey(addr))?.stale).toBe(false);
	});

	test('domainChanged foreign on boot only reloads the timeline', async () => {
		const first = bootView();
		const second = { ...first, label: 'Boot after foreign' };
		let n = 0;
		const timeline = fakeTimeline(async () => {
			n += 1;
			return n === 1 ? first : second;
		});
		const formFetch = vi.fn(async () => {
			throw new Error('unexpected form fetch');
		});
		const { session, formCache, hydrate } = harness(
			{ kind: 'boot', stream: 'recycling' },
			timeline,
			{ formFetch }
		);
		await session.load();
		hydrate.mockClear();

		const readyAddr = normalizeTaskAddress({
			stream: 'recycling',
			instance: '',
			jobId: 'intake',
			jobInstanceId: '',
			taskId: 'truckArrive'
		});
		formCache.putReady(readyAddr, {
			jobId: 'intake',
			taskId: 'truckArrive',
			jobInstanceId: '',
			title: 'Truck',
			surface: 'live',
			schema: { type: 'object' },
			uiSchema: {},
			initialData: {},
			chrome: [],
			chromeFieldIds: [],
			sealFieldIds: [],
			bindings: { domain: {} }
		});

		await session.domainChanged({ origin: 'foreign', revision: 2 });

		expect(session.view?.label).toBe('Boot after foreign');
		expect(timeline.loads).toHaveLength(2);
		expect(hydrate).not.toHaveBeenCalled();
		expect(formFetch).not.toHaveBeenCalled();
		expect(formCache.get(taskAddressKey(readyAddr))?.status).toBe('ready');
		expect(formCache.get(taskAddressKey(readyAddr))?.bundle?.title).toBe('Truck');
	});
});
