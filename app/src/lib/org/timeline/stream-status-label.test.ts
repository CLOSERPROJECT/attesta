import { describe, expect, test } from 'vitest';

import { nestStatusChipVisible, streamStatusChipVisible } from './stream-status-label.js';

describe('streamStatusChipVisible', () => {
	test('hides available when projectAll is off', () => {
		expect(streamStatusChipVisible('available', false)).toBe(false);
	});

	test('shows available when projectAll is on', () => {
		expect(streamStatusChipVisible('available', true)).toBe(true);
	});

	test('always shows other statuses', () => {
		for (const status of ['pending', 'locked', 'done', 'terminated'] as const) {
			expect(streamStatusChipVisible(status, false)).toBe(true);
			expect(streamStatusChipVisible(status, true)).toBe(true);
		}
	});
});

describe('nestStatusChipVisible', () => {
	test('suppresses all chips when nestStatusChips is off', () => {
		for (const status of ['available', 'locked', 'done'] as const) {
			expect(
				nestStatusChipVisible(status, { nestStatusChips: false, projectAll: true })
			).toBe(false);
		}
	});

	test('respects available-only when nest chips are on', () => {
		expect(
			nestStatusChipVisible('available', { nestStatusChips: true, projectAll: false })
		).toBe(false);
		expect(
			nestStatusChipVisible('locked', { nestStatusChips: true, projectAll: false })
		).toBe(true);
		expect(
			nestStatusChipVisible('available', { nestStatusChips: true, projectAll: true })
		).toBe(true);
	});

	test('empty status is hidden', () => {
		expect(
			nestStatusChipVisible(null, { nestStatusChips: true, projectAll: true })
		).toBe(false);
		expect(
			nestStatusChipVisible('', { nestStatusChips: true, projectAll: true })
		).toBe(false);
	});
});
