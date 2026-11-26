import { describe, expect, test } from 'vitest';

import { boardChromePolicy } from './board-chrome-policy.js';

describe('boardChromePolicy', () => {
	test('active + actionable-only', () => {
		expect(boardChromePolicy('active', false)).toEqual({
			projectAll: false,
			showAllFilterVisible: true,
			nestStatusChips: true,
			roleBadges: false,
			lockedFade: true,
			instanceStatusChip: false
		});
	});

	test('active + show-all toggle', () => {
		expect(boardChromePolicy('active', true)).toEqual({
			projectAll: true,
			showAllFilterVisible: true,
			nestStatusChips: true,
			roleBadges: true,
			lockedFade: true,
			instanceStatusChip: false
		});
	});

	test('done archive ignores toggle', () => {
		const expected = {
			projectAll: true,
			showAllFilterVisible: false,
			nestStatusChips: false,
			roleBadges: false,
			lockedFade: false,
			instanceStatusChip: true
		};
		expect(boardChromePolicy('done', false)).toEqual(expected);
		expect(boardChromePolicy('done', true)).toEqual(expected);
	});

	test('missing status behaves as active', () => {
		expect(boardChromePolicy(undefined, false).projectAll).toBe(false);
		expect(boardChromePolicy(null, true).projectAll).toBe(true);
		expect(boardChromePolicy('', false).instanceStatusChip).toBe(false);
	});
});
