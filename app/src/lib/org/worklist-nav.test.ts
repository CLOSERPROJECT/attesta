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

const stream = 'pv-recovery';
const instance = 'NIR-1';

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
	test('task next uses nextTaskFormBundle address under stream instance', () => {
		expect(
			resolvePostCommitNavigation(
				'task',
				stream,
				'NIR-advance',
				'intake',
				'NIR-advance',
				nextBundle
			)
		).toEqual({
			kind: 'task',
			stream,
			instance: 'NIR-advance',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
	});

	test('task next without usable bundle falls back to instance home', () => {
		expect(
			resolvePostCommitNavigation('task', stream, instance, 'intake', 'NIR-advance', null)
		).toEqual({
			kind: 'home',
			stream,
			instance
		});
	});

	test('legacy overview wire maps to instance home (never overview/archive)', () => {
		const target = resolvePostCommitNavigation(
			'overview',
			stream,
			instance,
			'intake',
			'NIR-1',
			null
		);
		expect(target).toEqual({ kind: 'home', stream, instance });
		expect(target.kind).not.toBe('overview');
		expect(target.kind).not.toBe('archive');
	});

	test('home next goes instance home; Play Commit uses minted instance', () => {
		expect(
			resolvePostCommitNavigation('home', stream, instance, 'intake', 'NIR-done', null)
		).toEqual({
			kind: 'home',
			stream,
			instance
		});
		expect(resolvePostCommitNavigation('home', stream, '', 'intake', 'T-new', null)).toEqual({
			kind: 'home',
			stream,
			instance: 'T-new'
		});
	});
});

describe('Task deep link + exit', () => {
	test('unauthorized Task URL → instance home; unavailable stays for blockedReason', () => {
		expect(
			resolveTaskDeepLink(
				stream,
				instance,
				'intake',
				'NIR-1',
				'palletWeight',
				false,
				adminWorklist
			)
		).toEqual({ kind: 'home', stream, instance });

		const withBlocked: typeof adminWorklist = {
			jobs: [
				{
					...adminWorklist.jobs[0]!,
					instances: [
						{
							id: 'NIR-1',
							label: 'NIR-1',
							tasks: { register: 'done', palletWeight: 'unavailable' },
							blockedReasons: { palletWeight: 'Linea occupata' }
						}
					]
				}
			]
		};
		expect(
			resolveTaskDeepLink(
				stream,
				instance,
				'intake',
				'NIR-1',
				'palletWeight',
				false,
				withBlocked
			)
		).toEqual({ kind: 'task' });

		expect(
			resolveTaskDeepLink(
				stream,
				'NIR-advance',
				'intake',
				'NIR-advance',
				'codiceFir',
				false,
				adminWorklist
			)
		).toEqual({ kind: 'task' });

		expect(
			resolveTaskDeepLink(stream, instance, 'intake', 'new', 'register', true, adminWorklist)
		).toEqual({ kind: 'task' });
	});

	test('provisional Start mounts only when canStart and first Task', () => {
		const blocked: Worklist = {
			jobs: [{ ...adminWorklist.jobs[0]!, canStart: false }]
		};
		expect(
			resolveTaskDeepLink(stream, instance, 'intake', 'new', 'register', true, blocked)
		).toEqual({ kind: 'home', stream, instance });
		expect(
			resolveTaskDeepLink(stream, instance, 'intake', 'new', 'codiceFir', true, adminWorklist)
		).toEqual({ kind: 'home', stream, instance });
	});

	test('Play provisional Start skips Worklist canStart (task-form is the gate)', () => {
		expect(
			resolveTaskDeepLink(stream, '', 'intake', 'new', 'truckArrive', true, null, {
				play: true
			})
		).toEqual({ kind: 'task' });
	});

	test('task-form deny shares unmountable redirect with deep-link', () => {
		expect(
			resolveUnmountableTaskTarget(stream, instance, 'intake', 'NIR-1', false, adminWorklist)
		).toEqual({ kind: 'home', stream, instance });
		expect(
			resolveUnmountableTaskTarget(stream, '', 'intake', '', true, adminWorklist)
		).toEqual({
			kind: 'myStream',
			stream
		});
	});

	test('Task exit: instance home; Play → my stream', () => {
		expect(resolveTaskExit(stream, instance, false)).toEqual({
			kind: 'home',
			stream,
			instance
		});
		expect(resolveTaskExit(stream, '', true)).toEqual({
			kind: 'myStream',
			stream
		});
	});

	test('navigation stay parks; other destinations follow ADR law', () => {
		expect(resolveCommitFollowUp('stay', stream, instance, 'intake', 'T1', null)).toEqual({
			kind: 'stay'
		});
		expect(resolveCommitFollowUp('overview', stream, instance, 'intake', 'T1', null)).toEqual({
			kind: 'home',
			stream,
			instance
		});
		expect(
			resolveCommitFollowUp(
				'task',
				stream,
				'NIR-advance',
				'intake',
				'NIR-advance',
				nextBundle
			)
		).toEqual({
			kind: 'task',
			stream,
			instance: 'NIR-advance',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
	});

	test('Continue uses continueNavigation only', () => {
		expect(
			resolveContinueNavigation('overview', stream, instance, 'intake', 'T1', null)
		).toEqual({
			kind: 'home',
			stream,
			instance
		});
		expect(
			resolveContinueNavigation(
				'task',
				stream,
				'NIR-advance',
				'intake',
				'NIR-advance',
				nextBundle
			)
		).toEqual({
			kind: 'task',
			stream,
			instance: 'NIR-advance',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
		expect(resolveContinueNavigation(null, stream, instance, 'intake', 'T1', null)).toEqual({
			kind: 'home',
			stream,
			instance
		});
	});
});
