import { describe, expect, test } from 'vitest';

import { taskBlockedReason, taskIsUnavailable } from './task-blocked.js';
import type { Worklist } from './types.js';

const worklist: Worklist = {
	jobs: [
		{
			jobId: 'lot_processing',
			canStart: false,
			repeatable: true,
			taskIds: ['startLot', 'weigh'],
			tasks: [
				{ id: 'startLot', title: 'Apri lotto', roles: ['logistic'] },
				{ id: 'weigh', title: 'Pesa', roles: ['logistic'] }
			],
			instances: [
				{
					id: 'L1',
					label: 'L1',
					tasks: { startLot: 'done', weigh: 'unavailable' },
					blockedReasons: {
						weigh: "Linea occupata: lotto L2 (camion T-A)"
					}
				}
			]
		}
	]
};

describe('taskBlockedReason', () => {
	test('returns authored reason for unavailable leaf', () => {
		expect(taskBlockedReason(worklist, 'lot_processing', 'L1', 'weigh')).toBe(
			'Linea occupata: lotto L2 (camion T-A)'
		);
	});

	test('null when leaf is not unavailable or reason missing', () => {
		expect(taskBlockedReason(worklist, 'lot_processing', 'L1', 'startLot')).toBeNull();
		const bare: Worklist = {
			jobs: [
				{
					...worklist.jobs[0]!,
					instances: [
						{
							id: 'L1',
							label: 'L1',
							tasks: { startLot: 'done', weigh: 'unavailable' }
						}
					]
				}
			]
		};
		expect(taskBlockedReason(bare, 'lot_processing', 'L1', 'weigh')).toBeNull();
	});
});

describe('taskIsUnavailable', () => {
	test('detects unavailable leaf', () => {
		expect(taskIsUnavailable(worklist, 'lot_processing', 'L1', 'weigh')).toBe(true);
		expect(taskIsUnavailable(worklist, 'lot_processing', 'L1', 'startLot')).toBe(
			false
		);
	});
});
