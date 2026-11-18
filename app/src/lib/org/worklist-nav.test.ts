import { describe, expect, test } from 'vitest';

import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { Worklist } from './types.js';
import {
	resolveCommitFollowUp,
	resolveContinueNavigation,
	resolvePostCommitNavigation,
	resolveTaskDeepLink,
	resolveTaskExit,
	resolveUnmountableTaskTarget
} from './worklist-nav.js';

const taskChrome = [
	{ id: 'register', title: 'Registrazione', roles: ['administration'] },
	{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
	{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
];

function worklistFor(viewer: 'admin' | 'logistic'): Worklist {
	const pallet = viewer === 'logistic' ? 'actionable' : 'available';
	const codice = viewer === 'admin' ? 'actionable' : 'available';
	return {
		jobs: [
			{
				jobId: 'intake',
				canStart: viewer === 'admin',
				repeatable: true,
				startLabel: 'Nuovo camion',
				taskIds: ['register', 'codiceFir', 'palletWeight'],
				tasks: taskChrome,
				instances: [
					{
						id: 'NIR-1',
						label: 'NIR-1',
						tasks: {
							register: 'done',
							codiceFir: 'done',
							palletWeight: pallet
						}
					},
					{
						id: 'NIR-advance',
						label: 'Advance',
						tasks: {
							register: 'done',
							codiceFir: codice,
							palletWeight: 'unavailable'
						}
					}
				]
			}
		]
	};
}

const adminWorklist = worklistFor('admin');

const nextBundle: TaskFormBundle = {
	jobId: 'intake',
	taskId: 'codiceFir',
	jobInstanceId: 'NIR-advance',
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

describe('post-Commit navigation adapter', () => {
	test('task next uses nextTaskFormBundle address', () => {
		expect(
			resolvePostCommitNavigation('task', 'intake', 'NIR-advance', nextBundle)
		).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
	});

	test('task next without usable bundle falls back to home', () => {
		expect(resolvePostCommitNavigation('task', 'intake', 'NIR-advance', null)).toEqual({
			kind: 'home'
		});
	});

	test('legacy overview wire maps to home (never Archive overview)', () => {
		expect(resolvePostCommitNavigation('overview', 'intake', 'NIR-1', null)).toEqual({
			kind: 'home'
		});
		expect(resolvePostCommitNavigation('overview', 'intake', 'NIR-1', null).kind).not.toBe(
			'overview'
		);
		expect(resolvePostCommitNavigation('overview', 'intake', 'NIR-1', null).kind).not.toBe(
			'archive'
		);
	});

	test('home next goes Home', () => {
		expect(resolvePostCommitNavigation('home', 'intake', 'NIR-done', null)).toEqual({
			kind: 'home'
		});
	});
});

describe('Task deep link + exit', () => {
	test('unauthorized / unavailable Task URL → home; unknown → home', () => {
		expect(
			resolveTaskDeepLink('intake', 'NIR-1', 'palletWeight', false, adminWorklist)
		).toEqual({ kind: 'home' });

		expect(
			resolveTaskDeepLink('intake', 'NIR-advance', 'codiceFir', false, adminWorklist)
		).toEqual({ kind: 'task' });

		expect(
			resolveTaskDeepLink('intake', 'missing', 'register', false, adminWorklist)
		).toEqual({ kind: 'home' });

		expect(
			resolveTaskDeepLink('intake', 'new', 'register', true, adminWorklist)
		).toEqual({ kind: 'task' });
	});

	test('provisional Start mounts only when canStart and first Task (server ViewerMayMountTask)', () => {
		const blocked: Worklist = {
			jobs: [
				{
					...adminWorklist.jobs[0]!,
					canStart: false
				}
			]
		};
		// After truckArrive Commit: worklist updates before navigate; must not fetch /new.
		expect(
			resolveTaskDeepLink('intake', 'new', 'register', true, blocked)
		).toEqual({ kind: 'home' });
		expect(
			resolveTaskDeepLink('intake', '', 'register', true, blocked)
		).toEqual({ kind: 'home' });
		// Wrong leaf on a startable Job (CollectActionableTaskRefs is first Task only).
		expect(
			resolveTaskDeepLink('intake', 'new', 'codiceFir', true, adminWorklist)
		).toEqual({ kind: 'home' });
	});

	test('task-form deny shares unmountable redirect with deep-link → home', () => {
		expect(
			resolveUnmountableTaskTarget('intake', 'NIR-1', false, adminWorklist)
		).toEqual({ kind: 'home' });
		expect(resolveUnmountableTaskTarget('intake', '', true, adminWorklist)).toEqual({
			kind: 'home'
		});
		expect(
			resolveUnmountableTaskTarget('intake', 'missing', false, adminWorklist)
		).toEqual({ kind: 'home' });
	});

	test('Task exit: always home (minted and provisional)', () => {
		expect(resolveTaskExit('intake', '', true)).toEqual({ kind: 'home' });
		expect(resolveTaskExit('intake', 'NIR-1', false)).toEqual({ kind: 'home' });
	});

	test('navigation stay parks; other destinations follow ADR law', () => {
		expect(resolveCommitFollowUp('stay', 'intake', 'T1', null)).toEqual({
			kind: 'stay'
		});
		expect(resolveCommitFollowUp('overview', 'intake', 'T1', null)).toEqual({
			kind: 'home'
		});
		expect(
			resolveCommitFollowUp('task', 'intake', 'NIR-advance', nextBundle)
		).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
		expect(resolveCommitFollowUp('home', 'passport', 'L1', null)).toEqual({
			kind: 'home'
		});
	});

	test('Continue uses continueNavigation only', () => {
		expect(
			resolveContinueNavigation('overview', 'intake', 'T1', null)
		).toEqual({
			kind: 'home'
		});
		expect(
			resolveContinueNavigation('task', 'intake', 'NIR-advance', nextBundle)
		).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
		expect(resolveContinueNavigation(null, 'intake', 'T1', null)).toEqual({
			kind: 'home'
		});
	});
});
