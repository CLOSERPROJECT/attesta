import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

describe('radioTilesWidget empty selection contract', () => {
	test('undefined value stays unselected: controlled checked, no bind:group', () => {
		const src = readFileSync(join(import.meta.dirname, 'radio-tiles-widget.svelte'), 'utf8');
		expect(src).not.toContain('bind:group');
		expect(src).toContain('checked={selected}');
		expect(src).toContain('mapped.current = optionValue');
		expect(src).toContain('role="radiogroup"');
		expect(src).toContain("inputAttributes(ctx, config, 'radioTiles', handlers");
		expect(src).not.toMatch(/feragame|lotId|containerReg|Polvere/);
	});
});
