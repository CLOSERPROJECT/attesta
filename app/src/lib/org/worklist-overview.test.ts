import { describe, expect, test } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import type { Worklist } from './types.js';
import {
	formatFactValue,
	formatHandoffCopy,
	formatSubmissionCount,
	jobInstanceOverview,
	manySubmitHistoryVisible,
	taskStatusLabel
} from './worklist-overview.js';

const taskChrome = [
	{ id: 'register', title: 'Registrazione', roles: ['administration'] },
	{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
	{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
];

function worklistFor(viewer: 'admin' | 'logistic'): Worklist {
	const palletMine = viewer === 'logistic' ? 'actionable' : 'available';
	const registerMine = viewer === 'admin' ? 'actionable' : 'available';
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
							palletWeight: palletMine
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
						id: 'NIR-both-available',
						label: 'Split available',
						tasks: {
							register: registerMine,
							codiceFir: 'locked',
							palletWeight: palletMine
						}
					}
				]
			}
		]
	};
}

const adminWorklist = worklistFor('admin');
const logisticWorklist = worklistFor('logistic');

describe('job instance overview model', () => {
	test('taskStatusLabel maps five-value status via Paraglide', () => {
		expect(taskStatusLabel('locked')).toBe(m.status_locked());
		expect(taskStatusLabel('unavailable')).toBe(m.status_unavailable());
		expect(taskStatusLabel('available')).toBe(m.status_available());
		expect(taskStatusLabel('actionable')).toBe(m.status_actionable());
		expect(taskStatusLabel('done')).toBe(m.status_done());
	});

	test('formatFactValue stringifies recap values', () => {
		expect(formatFactValue(null)).toBe('—');
		expect(formatFactValue('x')).toBe('x');
		expect(formatFactValue(42)).toBe('42');
		expect(formatFactValue({ a: 1 })).toBe('{"a":1}');
	});

	test('lists statuses; linkable only for actionable', () => {
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

	test('handoff uses first open only; no banner when next Task is theirs', () => {
		const admin = jobInstanceOverview(adminWorklist, 'intake', 'NIR-both-available');
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'actionable', true],
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

	test('finished model marks all done and remains projectable for overview', () => {
		const done = jobInstanceOverview(adminWorklist, 'intake', 'NIR-done');
		// Completed band deep-links need a finished overview model (not null).
		expect(done).not.toBeNull();
		expect(done?.finished).toBe(true);
		expect(done?.handoffRoles).toBeNull();
		expect(done?.tasks.every((t) => t.status === 'done')).toBe(true);
	});

	test('finished ignores submit: many leaves', () => {
		const flat: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['truckArrive', 'register', 'weightCheck'],
					tasks: [
						{
							id: 'truckArrive',
							title: 'Arrive',
							roles: ['administration'],
							submit: 'once'
						},
						{
							id: 'register',
							title: 'Register',
							roles: ['administration'],
							submit: 'many'
						},
						{
							id: 'weightCheck',
							title: 'Weight',
							roles: ['administration'],
							submit: 'once'
						}
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: {
								truckArrive: 'done',
								register: 'actionable',
								weightCheck: 'done'
							}
						}
					]
				}
			]
		};
		const model = jobInstanceOverview(flat, 'intake', 'T1');
		expect(model?.finished).toBe(true);
		const register = model?.tasks.find((t) => t.taskId === 'register');
		expect(register).toMatchObject({
			status: 'actionable',
			linkable: true,
			submit: 'many',
			submissionCount: 0
		});
		expect(model?.tasks.find((t) => t.taskId === 'truckArrive')?.submit).toBeUndefined();
	});

	test('many-submit leaf carries submissionCounts; history gate needs N > 0', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['register'],
					tasks: [
						{
							id: 'register',
							title: 'Register',
							roles: ['administration'],
							submit: 'many'
						}
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { register: 'actionable' },
							submissionCounts: { register: 2 }
						}
					]
				}
			]
		};
		const register = jobInstanceOverview(wl, 'intake', 'T1')?.tasks[0];
		expect(register).toMatchObject({
			submit: 'many',
			submissionCount: 2,
			linkable: true
		});
		expect(manySubmitHistoryVisible(register?.submit, register?.submissionCount)).toBe(
			true
		);
		expect(manySubmitHistoryVisible('many', 0)).toBe(false);
		expect(manySubmitHistoryVisible(undefined, 3)).toBe(false);
		expect(formatSubmissionCount(1)).toBe(m.submission_count_one({ count: 1 }));
		expect(formatSubmissionCount(2)).toBe(m.submission_count_many({ count: 2 }));
	});

	test('unknown instance yields null', () => {
		expect(jobInstanceOverview(adminWorklist, 'intake', 'missing')).toBeNull();
		expect(jobInstanceOverview(adminWorklist, 'intake', 'new')).toBeNull();
	});

});
