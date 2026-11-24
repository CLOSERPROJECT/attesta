import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

describe('readonlyField hideIfNull contract', () => {
	test('theme gates paint on hideIfNull + null display; no plant Task ids', () => {
		const src = readFileSync(join(import.meta.dirname, 'readonly-field.svelte'), 'utf8');
		expect(src).toContain("uiOption('hideIfNull')");
		expect(src).toContain('display.value == null');
		expect(src).toContain('hideIfNull?: boolean');
		expect(src).not.toMatch(/feragame|lotId|containerReg/);
	});
});
