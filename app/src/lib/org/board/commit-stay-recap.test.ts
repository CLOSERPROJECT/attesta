import { describe, expect, test } from 'vitest';

import { stayAftermath, stayConfirmSurface } from './commit-stay-recap.js';

describe('stayConfirmSurface', () => {
	test('submitMany stay lands on Recap tab', () => {
		expect(stayConfirmSurface(true)).toBe('many-recap-tab');
	});

	test('once stay lands on done/recap surface', () => {
		expect(stayConfirmSurface(false)).toBe('once-recap');
	});
});

describe('stayAftermath', () => {
	test('submitMany stay requires a fresh empty Form draft', () => {
		expect(stayAftermath(true)).toEqual({
			confirmSurface: 'many-recap-tab',
			freshDraft: true
		});
	});

	test('once stay keeps Form hidden behind once-recap (no fresh-draft advance)', () => {
		expect(stayAftermath(false)).toEqual({
			confirmSurface: 'once-recap',
			freshDraft: false
		});
	});
});
