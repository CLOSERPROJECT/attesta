import { describe, expect, test } from 'vitest';

import type { TaskAddress } from '../task-address.js';
import { recapRowMatchesSubmissionId } from '../submission-table.js';
import type { JustCompletedPin } from './board-just-completed.svelte.js';
import { pinSubmissionIdForAddress } from './just-completed-recap-row.js';

const addr = (taskId: string, jobInstanceId = 'T1'): TaskAddress => ({
	stream: 'recycling',
	instance: '261102-1',
	jobId: 'intake',
	jobInstanceId,
	taskId,
	provisionalJob: false
});

describe('pinSubmissionIdForAddress', () => {
	test('returns submissionId only when pin address matches', () => {
		const pin: JustCompletedPin = {
			address: addr('register'),
			submissionId: '2026-11-13T12:00:01Z'
		};
		expect(pinSubmissionIdForAddress(pin, addr('register'))).toBe(
			'2026-11-13T12:00:01Z'
		);
		expect(pinSubmissionIdForAddress(pin, addr('truckArrive'))).toBeUndefined();
		expect(pinSubmissionIdForAddress(null, addr('register'))).toBeUndefined();
		expect(
			pinSubmissionIdForAddress(
				{ address: addr('register') },
				addr('register')
			)
		).toBeUndefined();
	});
});

describe('recapRowMatchesSubmissionId', () => {
	test('matches TaskRecapRow.submissionId to Commit submissionId', () => {
		const id = '2026-11-13T12:00:01Z';
		expect(recapRowMatchesSubmissionId({ submissionId: id }, id)).toBe(true);
		expect(
			recapRowMatchesSubmissionId({ submissionId: '2026-11-13T12:00:00Z' }, id)
		).toBe(false);
		expect(recapRowMatchesSubmissionId({ submissionId: id }, undefined)).toBe(
			false
		);
		expect(recapRowMatchesSubmissionId({ submissionId: id }, '')).toBe(false);
	});
});
