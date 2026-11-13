import { describe, expect, test } from 'vitest';

import {
	emptyWorklist,
	orderedTaskIds,
	taskById,
	worklistJobFor,
	type Worklist
} from './types.js';

describe('worklist helpers', () => {
	test('emptyWorklist has no jobs', () => {
		expect(emptyWorklist()).toEqual({ jobs: [] });
	});

	test('orderedTaskIds and taskById', () => {
		const tasks = [{ id: 'a' }, { id: 'b' }];
		expect(orderedTaskIds(tasks)).toEqual(['a', 'b']);
		expect(orderedTaskIds(undefined)).toEqual([]);
		expect(taskById(tasks, 'b')?.id).toBe('b');
		expect(taskById(tasks, 'missing')).toBeUndefined();
	});

	test('worklistJobFor finds by jobId', () => {
		const worklist: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['register'],
					tasks: [{ id: 'register', title: 'Registrazione', roles: ['administration'] }],
					instances: []
				},
				{
					jobId: 'line',
					canStart: false,
					repeatable: true,
					taskIds: ['shift', 'lineLoad'],
					tasks: [
						{ id: 'shift', title: 'Turno', roles: ['line'] },
						{ id: 'lineLoad', title: 'Carico', roles: ['line'] }
					],
					instances: [
						{
							id: '1st',
							label: 'Turno: 1st',
							tasks: { shift: 'done', lineLoad: 'available' }
						}
					]
				}
			]
		};
		expect(worklistJobFor(worklist, 'line')?.canStart).toBe(false);
		expect(worklistJobFor(worklist, 'line')?.instances[0]?.tasks.lineLoad).toBe('available');
		expect(worklistJobFor(worklist, 'missing')).toBeNull();
		expect(worklistJobFor(null, 'intake')).toBeNull();
	});

	test('Load fixture Worklist chrome carries Task title + roles', () => {
		const worklist: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['register', 'palletWeight'],
					tasks: [
						{ id: 'register', title: 'Registrazione', roles: ['administration'] },
						{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
					],
					instances: []
				}
			]
		};
		const intake = worklistJobFor(worklist, 'intake');
		expect(intake?.tasks).toHaveLength(2);
		expect(intake?.tasks[0]).toEqual({
			id: 'register',
			title: 'Registrazione',
			roles: ['administration']
		});
		expect(intake?.tasks[1]?.title).toBeTruthy();
		expect(intake?.tasks[1]?.roles).toEqual(['logistic']);
	});

	test('WorklistInstance submissionCounts mirrors Go wire (0 omit / 1 / N)', () => {
		const emptyCounts: Worklist = {
			jobs: [
				{
					jobId: 'session',
					canStart: false,
					repeatable: true,
					taskIds: ['repeat'],
					tasks: [{ id: 'repeat', title: 'Repeat', roles: ['operator'], submit: 'many' }],
					instances: [{ id: 'S1', label: 'Session', tasks: { repeat: 'actionable' } }]
				}
			]
		};
		expect(emptyCounts.jobs[0].instances[0].submissionCounts).toBeUndefined();

		const one: Worklist = {
			jobs: [
				{
					jobId: 'session',
					canStart: false,
					repeatable: true,
					taskIds: ['start', 'repeat'],
					tasks: [
						{ id: 'start', title: 'Start', roles: ['operator'] },
						{ id: 'repeat', title: 'Repeat', roles: ['operator'], submit: 'many' }
					],
					instances: [
						{
							id: 'S1',
							label: 'Session',
							tasks: { start: 'done', repeat: 'actionable' },
							submissionCounts: { start: 1 }
						}
					]
				}
			]
		};
		expect(one.jobs[0].instances[0].submissionCounts).toEqual({ start: 1 });

		const many: Worklist = {
			jobs: [
				{
					jobId: 'session',
					canStart: false,
					repeatable: true,
					taskIds: ['repeat'],
					tasks: [{ id: 'repeat', title: 'Repeat', roles: ['operator'], submit: 'many' }],
					instances: [
						{
							id: 'S1',
							label: 'Session',
							tasks: { repeat: 'actionable' },
							submissionCounts: { repeat: 3 }
						}
					]
				}
			]
		};
		expect(many.jobs[0].instances[0].submissionCounts?.repeat).toBe(3);
	});
});
