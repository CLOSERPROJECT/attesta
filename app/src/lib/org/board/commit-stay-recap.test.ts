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
	test('submitMany stay refreshes form bundles and lands on Recap tab', () => {
		expect(stayAftermath(true)).toEqual({
			confirmSurface: 'many-recap-tab',
			refreshForms: true
		});
	});

	test('once stay keeps Form hidden behind once-recap (no form refresh)', () => {
		expect(stayAftermath(false)).toEqual({
			confirmSurface: 'once-recap',
			refreshForms: false
		});
	});
});
