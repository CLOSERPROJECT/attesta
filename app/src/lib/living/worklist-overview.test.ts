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

const worklist: Worklist = {
	jobs: [
		{
			jobId: 'intake',
			canStart: true,
			repeatable: true,
			startLabel: 'Nuovo camion',
			taskIds: ['register', 'codiceFir', 'palletWeight'],
			tasks: [
				{ id: 'register', title: 'Registrazione', roles: ['administration'] },
				{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
				{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
			],
			instances: [
				{
					id: 'NIR-1',
					label: 'NIR-1',
					tasks: {
						register: 'done',
						codiceFir: 'done',
						palletWeight: 'available'
					}
				},
				{
					id: 'NIR-done',
					label: 'Done truck',
					tasks: {
						register: 'done',
						codiceFir: 'done',
						palletWeight: 'done'
					}
				},
				{
					id: 'NIR-advance',
					label: 'Advance',
					tasks: {
						register: 'done',
						codiceFir: 'available',
						palletWeight: 'locked'
					}
				},
				{
					id: 'NIR-both-available',
					label: 'Split available',
					tasks: {
						register: 'available',
						codiceFir: 'locked',
						palletWeight: 'available'
					}
				}
			]
		}
	]
};

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
			resolvePostCommitNavigation('intake', 'NIR-advance', worklist, nextBundle)
		).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
	});

	test('handoff opens overview when remaining work is not role-eligible', () => {
		expect(
			resolvePostCommitNavigation('intake', 'NIR-1', worklist, null)
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });
		expect(instanceHasRemainingWork(worklist, 'intake', 'NIR-1')).toBe(true);
	});

	test('finished instance goes Home', () => {
		expect(
			resolvePostCommitNavigation('intake', 'NIR-done', worklist, null)
		).toEqual({ kind: 'home' });
		expect(instanceIsFinished(worklist, 'intake', 'NIR-done')).toBe(true);
	});
});

describe('job instance overview model', () => {
	test('lists statuses; linkable only for role-eligible available', () => {
		const admin = jobInstanceOverview(worklist, 'intake', 'NIR-1', ['administration']);
		expect(admin?.label).toBe('NIR-1');
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'done', false],
			['codiceFir', 'done', false],
			['palletWeight', 'available', false]
		]);
		expect(admin?.handoffRoles).toEqual(['logistic']);
		expect(formatHandoffCopy(admin!.handoffRoles!)).toContain('logistic');
		expect(admin?.finished).toBe(false);

		const logistic = jobInstanceOverview(worklist, 'intake', 'NIR-1', ['logistic']);
		expect(logistic?.tasks.find((t) => t.taskId === 'palletWeight')?.linkable).toBe(
			true
		);
		expect(logistic?.handoffRoles).toBeNull();
	});

	test('handoff uses first available only; no banner when next Task is theirs', () => {
		const admin = jobInstanceOverview(worklist, 'intake', 'NIR-both-available', [
			'administration'
		]);
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'available', true],
			['codiceFir', 'locked', false],
			['palletWeight', 'available', false]
		]);
		expect(admin?.handoffRoles).toBeNull();

		const logistic = jobInstanceOverview(worklist, 'intake', 'NIR-both-available', [
			'logistic'
		]);
		expect(logistic?.handoffRoles).toEqual(['administration']);
		expect(logistic?.tasks.find((t) => t.taskId === 'register')?.linkable).toBe(false);
		expect(logistic?.tasks.find((t) => t.taskId === 'palletWeight')?.linkable).toBe(
			true
		);
	});

	test('finished model marks all done', () => {
		const done = jobInstanceOverview(worklist, 'intake', 'NIR-done', ['administration']);
		expect(done?.finished).toBe(true);
		expect(done?.handoffRoles).toBeNull();
	});

	test('unknown instance yields null', () => {
		expect(jobInstanceOverview(worklist, 'intake', 'missing', ['administration'])).toBeNull();
		expect(jobInstanceOverview(worklist, 'intake', 'new', ['administration'])).toBeNull();
	});
});

describe('Task deep link + exit', () => {
	test('unauthorized / locked Task URL → overview; unknown → Home', () => {
		expect(
			resolveTaskDeepLink('intake', 'NIR-1', 'palletWeight', false, worklist, [
				'administration'
			])
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });

		expect(
			resolveTaskDeepLink('intake', 'NIR-advance', 'codiceFir', false, worklist, [
				'administration'
			])
		).toEqual({ kind: 'task' });

		expect(
			resolveTaskDeepLink('intake', 'missing', 'register', false, worklist, [
				'administration'
			])
		).toEqual({ kind: 'home' });

		expect(
			resolveTaskDeepLink('intake', 'new', 'register', true, worklist, ['administration'])
		).toEqual({ kind: 'task' });
	});

	test('task-form deny shares unmountable redirect with deep-link', () => {
		expect(
			resolveUnmountableTaskTarget('intake', 'NIR-1', false, worklist)
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });
		expect(resolveUnmountableTaskTarget('intake', '', true, worklist)).toEqual({
			kind: 'home'
		});
		expect(
			resolveUnmountableTaskTarget('intake', 'missing', false, worklist)
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
