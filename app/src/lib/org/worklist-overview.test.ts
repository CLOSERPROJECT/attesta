import { describe, expect, test } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import {
	formatFactValue,
	formatSubmissionCount,
	manySubmitHistoryVisible,
	taskStatusLabel
} from './worklist-overview.js';

describe('worklist-overview helpers (Task chrome)', () => {
	test('taskStatusLabel maps four-value status via Paraglide', () => {
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

	test('manySubmitHistoryVisible requires many + count > 0', () => {
		expect(manySubmitHistoryVisible('many', 2)).toBe(true);
		expect(manySubmitHistoryVisible('many', 0)).toBe(false);
		expect(manySubmitHistoryVisible(undefined, 3)).toBe(false);
	});

	test('formatSubmissionCount uses Paraglide one/many', () => {
		expect(formatSubmissionCount(1)).toBe(m.submission_count_one({ count: 1 }));
		expect(formatSubmissionCount(3)).toBe(m.submission_count_many({ count: 3 }));
	});
});
