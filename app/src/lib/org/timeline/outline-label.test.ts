import { describe, expect, test } from 'vitest';

import { displayStepTitle, outlineLetter } from './outline-label.js';

describe('outlineLetter', () => {
	test('A–Z then AA', () => {
		expect(outlineLetter(0)).toBe('A');
		expect(outlineLetter(25)).toBe('Z');
		expect(outlineLetter(26)).toBe('AA');
		expect(outlineLetter(27)).toBe('AB');
	});
});

describe('displayStepTitle', () => {
	test('strips ×N suffix', () => {
		expect(displayStepTitle('Lavorazione ×2')).toBe('Lavorazione');
		expect(displayStepTitle('Lavorazione')).toBe('Lavorazione');
	});
});
