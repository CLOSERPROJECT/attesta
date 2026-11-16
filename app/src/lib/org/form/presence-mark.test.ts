import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { shouldPaintPresenceMark } from './presence-mark.js';

describe('shouldPaintPresenceMark', () => {
	test('paints when presenceMark and value set', () => {
		expect(shouldPaintPresenceMark(true, 'C-1')).toBe(true);
		expect(shouldPaintPresenceMark(true, '  prior  ')).toBe(true);
	});

	test('omits when value empty or option off', () => {
		expect(shouldPaintPresenceMark(true, '')).toBe(false);
		expect(shouldPaintPresenceMark(true, '   ')).toBe(false);
		expect(shouldPaintPresenceMark(true, undefined)).toBe(false);
		expect(shouldPaintPresenceMark(true, null)).toBe(false);
		expect(shouldPaintPresenceMark(false, 'C-1')).toBe(false);
		expect(shouldPaintPresenceMark(undefined, 'C-1')).toBe(false);
	});
});

describe('hiddenField presenceMark theme contract', () => {
	test('theme paints mark from option + value; no plant Task ids', () => {
		const dir = import.meta.dirname;
		const hidden = readFileSync(join(dir, 'hidden-field.svelte'), 'utf8');
		const compact = readFileSync(join(dir, 'compact-array-item-field.svelte'), 'utf8');
		expect(hidden).toContain("uiOption('presenceMark')");
		expect(hidden).toContain('shouldPaintPresenceMark');
		expect(hidden).toContain('data-presence-mark');
		expect(hidden).toContain('data-testid="presence-mark"');
		expect(hidden).toContain('aria-label={label || undefined}');
		expect(hidden).not.toMatch(/containerReg|continuesContainerId/);
		expect(compact).toContain('data-presence-mark');
		expect(compact).toContain(':has([data-hidden-field])');
	});
});
