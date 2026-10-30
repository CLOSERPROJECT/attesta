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
					instances: []
				},
				{
					jobId: 'line',
					canStart: false,
					repeatable: true,
					taskIds: ['shift', 'lineLoad'],
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
});
