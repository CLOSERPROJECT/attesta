import { describe, expect, test, vi } from 'vitest';

import type { TaskFormBundle } from '../form/task-form-bundle.js';
import { normalizeTaskAddress, taskAddressKey } from '../task-address.js';
import type { TimelineLoadQuery, TimelinePort } from '../timeline/timeline-port.js';
import type { TimelineView } from '../timeline/types.js';
import { createBoardFormCache } from './board-form-cache.svelte.js';
import { createBoardRecapCache } from './board-recap-cache.svelte.js';
import {
	boardScopeKey,
	createBoardSession,
	type BoardScope
} from './board-session.svelte.js';
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

function instanceView(
	stream = 'recycling',
	instance = '261102-1'
): TimelineView {
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

function harness(scope: BoardScope, timeline: TimelinePort) {
	const formCache = createBoardFormCache({
		taskForm: { fetch: vi.fn(async () => {
			throw new Error('unexpected form fetch');
		}) }
	});
	const recapCache = createBoardRecapCache({
		taskRecap: { fetch: vi.fn(async () => []) }
	});
	const model = new BoardTimelineModel({ syncUrl: () => {} });
	const navigateToInstance = vi.fn(async () => {});
	const hydrate = vi.fn();
	const session = createBoardSession({
		scope,
		timeline,
		model,
		formCache,
		recapCache,
		navigateToInstance,
		hydrate
	});
	return { session, model, formCache, recapCache, navigateToInstance, hydrate };
}

describe('boardScopeKey', () => {
	test('encodes boot and instance distinctly', () => {
		expect(boardScopeKey({ kind: 'boot', stream: 'recycling' })).toBe(
			'recycling||boot'
		);
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
		const { session, model } = harness(
			{ kind: 'boot', stream: 'recycling' },
			timeline
		);

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

		expect(timeline.loads).toEqual([
			{ stream: 'recycling', instance: '261102-1' }
		]);
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
		const { session } = harness(
			{ kind: 'boot', stream: 'recycling' },
			timeline
		);

		await session.load();

		expect(session.view).toBeNull();
		expect(session.loading).toBe(false);
		expect(session.error).toBe('timeline down');
	});
});
