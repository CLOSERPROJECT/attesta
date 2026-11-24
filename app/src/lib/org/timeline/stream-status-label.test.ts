import { describe, expect, test } from 'vitest';

import { streamStatusChipVisible } from './stream-status-label.js';

describe('streamStatusChipVisible', () => {
	test('hides available when showAll is off', () => {
		expect(streamStatusChipVisible('available', false)).toBe(false);
	});

	test('shows available when showAll is on', () => {
		expect(streamStatusChipVisible('available', true)).toBe(true);
	});

	test('always shows other statuses', () => {
		for (const status of ['pending', 'locked', 'done', 'terminated'] as const) {
			expect(streamStatusChipVisible(status, false)).toBe(true);
			expect(streamStatusChipVisible(status, true)).toBe(true);
		}
	});
});
