import { describe, expect, test } from 'vitest';

import type { Submission, TaskConfig } from './types.js';
import { isTaskUnlocked, orderedTaskIds, taskMayBeUnlockGate } from './unlock.js';

const order = ['register', 'codiceFir', 'palletWeight', 'weightCheck'];
const tasks: TaskConfig[] = [
	{ id: 'register' },
	{ id: 'codiceFir' },
	{
		id: 'palletWeight',
		dynamicProperties: { weights: { read: 'leftovers', completeness: 'all' } }
	},
	{ id: 'weightCheck' }
];

function sub(taskId: string, jobInstanceId: string): Submission {
	return {
		jobId: 'intake',
		taskId,
		jobInstanceId,
		timestamp: 't',
		userId: 'u',
		facts: {},
		idsTouched: []
	};
}

describe('isTaskUnlocked', () => {
	test('first Task unlocked without instance', () => {
		expect(
			isTaskUnlocked({
				orderedIds: order,
				jobId: 'intake',
				taskId: 'register',
				jobInstanceId: '',
				submissions: [],
				tasks
			})
		).toBe(true);
	});

	test('codiceFir locked until register Submission', () => {
		expect(
			isTaskUnlocked({
				orderedIds: order,
				jobId: 'intake',
				taskId: 'codiceFir',
				jobInstanceId: 'NIR-1',
				submissions: [],
				tasks
			})
		).toBe(false);
		expect(
			isTaskUnlocked({
				orderedIds: order,
				jobId: 'intake',
				taskId: 'codiceFir',
				jobInstanceId: 'NIR-1',
				submissions: [sub('register', 'NIR-1')],
				tasks
			})
		).toBe(true);
	});

	test('completeness all predecessor unlocks next after Submission', () => {
		const afterPw = [sub('register', 'NIR-1'), sub('codiceFir', 'NIR-1'), sub('palletWeight', 'NIR-1')];
		expect(
			isTaskUnlocked({
				orderedIds: order,
				jobId: 'intake',
				taskId: 'weightCheck',
				jobInstanceId: 'NIR-1',
				submissions: afterPw.slice(0, 2),
				tasks
			})
		).toBe(false);
		expect(
			isTaskUnlocked({
				orderedIds: order,
				jobId: 'intake',
				taskId: 'weightCheck',
				jobInstanceId: 'NIR-1',
				submissions: afterPw,
				tasks
			})
		).toBe(true);
	});

	test('completeness some predecessor is not an unlock gate', () => {
		expect(
			taskMayBeUnlockGate({
				dynamicProperties: { progress: { completeness: 'some' } }
			})
		).toBe(false);
	});

	test('completeness some predecessor does not unlock next without unlockFrom instance', () => {
		const someTasks: TaskConfig[] = [
			{
				id: 'lineLoad',
				dynamicProperties: {
					progress: { read: 'leftovers', completeness: 'some' }
				}
			},
			{ id: 'next' }
		];
		expect(
			isTaskUnlocked({
				orderedIds: ['lineLoad', 'next'],
				jobId: 'line',
				taskId: 'next',
				jobInstanceId: '1st',
				submissions: [
					{
						jobId: 'line',
						taskId: 'lineLoad',
						jobInstanceId: '1st',
						timestamp: 't',
						userId: 'u',
						facts: {},
						idsTouched: []
					}
				],
				tasks: someTasks
			})
		).toBe(false);
	});

	test('completeness some predecessor unlocks next only with unlockFrom instance', () => {
		const someTasks: TaskConfig[] = [
			{
				id: 'lineLoad',
				dynamicProperties: {
					progress: { read: 'leftovers', completeness: 'some' }
				}
			},
			{ id: 'containerReg', unlockFrom: 'instance' }
		];
		expect(
			isTaskUnlocked({
				orderedIds: ['lineLoad', 'containerReg'],
				jobId: 'line',
				taskId: 'containerReg',
				jobInstanceId: '1st',
				submissions: [],
				tasks: someTasks
			})
		).toBe(true);
		expect(
			isTaskUnlocked({
				orderedIds: ['lineLoad', 'containerReg'],
				jobId: 'line',
				taskId: 'containerReg',
				jobInstanceId: '',
				submissions: [],
				tasks: someTasks
			})
		).toBe(false);
	});

	test('orderedTaskIds preserves array order', () => {
		expect(orderedTaskIds(tasks)).toEqual(order);
		expect(orderedTaskIds(undefined)).toEqual([]);
	});
});
