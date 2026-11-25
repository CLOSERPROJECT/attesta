import { describe, expect, test } from 'vitest';

import { timelineSubstepTrailVisible } from './timeline-substep-trail.js';

describe('timelineSubstepTrailVisible', () => {
	test('draft alone shows trail (independent of showAll filters)', () => {
		expect(
			timelineSubstepTrailVisible({
				showStatusChip: false,
				roleBadgeCount: 0,
				showDraft: true
			})
		).toBe(true);
	});

	test('empty when no chip, roles, or draft', () => {
		expect(
			timelineSubstepTrailVisible({
				showStatusChip: false,
				roleBadgeCount: 0,
				showDraft: false
			})
		).toBe(false);
	});

	test('status chip or role badges also show trail', () => {
		expect(
			timelineSubstepTrailVisible({
				showStatusChip: true,
				roleBadgeCount: 0,
				showDraft: false
			})
		).toBe(true);
		expect(
			timelineSubstepTrailVisible({
				showStatusChip: false,
				roleBadgeCount: 1,
				showDraft: false
			})
		).toBe(true);
	});
});
