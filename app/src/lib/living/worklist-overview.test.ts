import { describe, expect, test } from 'vitest';

import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { Worklist } from './types.js';
import {
	formatHandoffCopy,
	instanceHasRemainingWork,
	instanceIsFinished,
	jobInstanceOverview,
	resolvePostCommitNavigation,
	resolveTaskDeepLink,
	resolveTaskExit,
	resolveUnmountableTaskTarget
} from './worklist-overview.js';

const taskChrome = [
	{ id: 'register', title: 'Registrazione', roles: ['administration'] },
	{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
	{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
];

/** Shared statuses; viewerAvailable encodes role-eligible Available. */
function worklistFor(viewer: 'admin' | 'logistic'): Worklist {
	const nir1Avail = viewer === 'logistic' ? ['palletWeight'] : [];
	const advanceAvail = viewer === 'admin' ? ['codiceFir'] : [];
	const bothAvail =
		viewer === 'admin'
			? ['register']
			: viewer === 'logistic'
				? ['palletWeight']
				: [];
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
							palletWeight: 'available'
						},
						viewerAvailable: nir1Avail
					},
					{
						id: 'NIR-done',
						label: 'Done truck',
						tasks: {
							register: 'done',
							codiceFir: 'done',
							palletWeight: 'done'
						},
						viewerAvailable: []
					},
					{
						id: 'NIR-advance',
						label: 'Advance',
						tasks: {
							register: 'done',
							codiceFir: 'available',
							palletWeight: 'locked'
						},
						viewerAvailable: advanceAvail
					},
					{
						id: 'NIR-both-available',
						label: 'Split available',
						tasks: {
							register: 'available',
							codiceFir: 'locked',
							palletWeight: 'available'
						},
						viewerAvailable: bothAvail
					}
				]
			}
		]
	};
}

const adminWorklist = worklistFor('admin');
const logisticWorklist = worklistFor('logistic');

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

describe('post-Commit navigation', () => {
	test('advance prefers nextTaskFormBundle', () => {
		expect(
			resolvePostCommitNavigation('intake', 'NIR-advance', adminWorklist, nextBundle)
		).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
	});

	test('handoff opens overview when remaining work is not role-eligible', () => {
		expect(
			resolvePostCommitNavigation('intake', 'NIR-1', adminWorklist, null)
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });
		expect(instanceHasRemainingWork(adminWorklist, 'intake', 'NIR-1')).toBe(true);
	});

	test('finished instance goes Home', () => {
		expect(
			resolvePostCommitNavigation('intake', 'NIR-done', adminWorklist, null)
		).toEqual({ kind: 'home' });
		expect(instanceIsFinished(adminWorklist, 'intake', 'NIR-done')).toBe(true);
	});
});

describe('job instance overview model', () => {
	test('lists statuses; linkable only for viewerAvailable', () => {
		const admin = jobInstanceOverview(adminWorklist, 'intake', 'NIR-1');
		expect(admin?.label).toBe('NIR-1');
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'done', false],
			['codiceFir', 'done', false],
			['palletWeight', 'available', false]
		]);
		expect(admin?.handoffRoles).toEqual(['logistic']);
		expect(formatHandoffCopy(admin!.handoffRoles!)).toContain('logistic');
		expect(admin?.finished).toBe(false);

		const logistic = jobInstanceOverview(logisticWorklist, 'intake', 'NIR-1');
		expect(logistic?.tasks.find((t) => t.taskId === 'palletWeight')?.linkable).toBe(
			true
		);
		expect(logistic?.handoffRoles).toBeNull();
	});

	test('handoff uses first available only; no banner when next Task is theirs', () => {
		const admin = jobInstanceOverview(adminWorklist, 'intake', 'NIR-both-available');
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'available', true],
			['codiceFir', 'locked', false],
			['palletWeight', 'available', false]
		]);
		expect(admin?.handoffRoles).toBeNull();

		const logistic = jobInstanceOverview(
			logisticWorklist,
			'intake',
			'NIR-both-available'
		);
		expect(logistic?.handoffRoles).toEqual(['administration']);
		expect(logistic?.tasks.find((t) => t.taskId === 'register')?.linkable).toBe(false);
		expect(logistic?.tasks.find((t) => t.taskId === 'palletWeight')?.linkable).toBe(
			true
		);
	});

	test('finished model marks all done', () => {
		const done = jobInstanceOverview(adminWorklist, 'intake', 'NIR-done');
		expect(done?.finished).toBe(true);
		expect(done?.handoffRoles).toBeNull();
	});

	test('unknown instance yields null', () => {
		expect(jobInstanceOverview(adminWorklist, 'intake', 'missing')).toBeNull();
		expect(jobInstanceOverview(adminWorklist, 'intake', 'new')).toBeNull();
	});
});

describe('Task deep link + exit', () => {
	test('unauthorized / locked Task URL → overview; unknown → Home', () => {
		expect(
			resolveTaskDeepLink('intake', 'NIR-1', 'palletWeight', false, adminWorklist)
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });

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

	test('task-form deny shares unmountable redirect with deep-link', () => {
		expect(
			resolveUnmountableTaskTarget('intake', 'NIR-1', false, adminWorklist)
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });
		expect(resolveUnmountableTaskTarget('intake', '', true, adminWorklist)).toEqual({
			kind: 'home'
		});
		expect(
			resolveUnmountableTaskTarget('intake', 'missing', false, adminWorklist)
		).toEqual({ kind: 'home' });
	});

	test('Task exit: provisional Home; existing overview', () => {
		expect(resolveTaskExit('intake', '', true)).toEqual({ kind: 'home' });
		expect(resolveTaskExit('intake', 'NIR-1', false)).toEqual({
			kind: 'overview',
			jobId: 'intake',
			jobInstanceId: 'NIR-1'
		});
	});
});
