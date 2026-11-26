import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { timelineSubstepUsesEmbed } from './timeline-substep-embed.js';

describe('timelineSubstepUsesEmbed', () => {
	test('no address → never embed', () => {
		expect(
			timelineSubstepUsesEmbed({
				hasAddress: false,
				orgActionable: true,
				focused: true,
				confirmRecap: true,
				justCompleted: true
			})
		).toBe(false);
	});

	test('orgActionable → embed', () => {
		expect(
			timelineSubstepUsesEmbed({
				hasAddress: true,
				orgActionable: true,
				focused: false,
				confirmRecap: false,
				justCompleted: false
			})
		).toBe(true);
	});

	test('locked after stay: focus / confirmRecap / justCompleted keep embed', () => {
		const locked = {
			hasAddress: true,
			orgActionable: false,
			focused: false,
			confirmRecap: false,
			justCompleted: false
		};
		expect(timelineSubstepUsesEmbed(locked)).toBe(false);
		expect(timelineSubstepUsesEmbed({ ...locked, focused: true })).toBe(true);
		expect(timelineSubstepUsesEmbed({ ...locked, confirmRecap: true })).toBe(true);
		expect(timelineSubstepUsesEmbed({ ...locked, justCompleted: true })).toBe(true);
	});
});

describe('timeline-substep embed gate wiring', () => {
	test('uses timelineSubstepUsesEmbed; not orgActionable alone', () => {
		const src = readFileSync(join(import.meta.dirname, 'timeline-substep.svelte'), 'utf8');
		expect(src).toContain('timelineSubstepUsesEmbed');
		expect(src).not.toMatch(
			/const embed = \$derived\(\!\!\(substep\.orgActionable && address\)\)/
		);
	});
});
