import { describe, expect, test } from 'vitest';
import { isChromeTone } from './chrome-tone.js';

describe('isChromeTone', () => {
	test('accepts catalog tones only', () => {
		expect(isChromeTone('info')).toBe(true);
		expect(isChromeTone('green')).toBe(true);
		expect(isChromeTone('red')).toBe(false);
		expect(isChromeTone(1)).toBe(false);
	});
});
