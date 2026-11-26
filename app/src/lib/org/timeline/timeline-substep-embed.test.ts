import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { timelineSubstepShowsTaskPanel } from './timeline-substep-embed.js';

describe('timelineSubstepShowsTaskPanel', () => {
	test('no address → never panel', () => {
		expect(
			timelineSubstepShowsTaskPanel({
				hasAddress: false,
				orgActionable: true,
				confirmRecap: true,
				justCompleted: true,
				submissionCount: 3
			})
		).toBe(false);
	});

	test('orgActionable → panel', () => {
		expect(
			timelineSubstepShowsTaskPanel({
				hasAddress: true,
				orgActionable: true,
				confirmRecap: false,
				justCompleted: false,
				submissionCount: 0
			})
		).toBe(true);
	});

	test('locked after stay: confirmRecap / justCompleted / submissions keep panel', () => {
		const locked = {
			hasAddress: true,
			orgActionable: false,
			confirmRecap: false,
			justCompleted: false,
			submissionCount: 0
		};
		expect(timelineSubstepShowsTaskPanel(locked)).toBe(false);
		expect(timelineSubstepShowsTaskPanel({ ...locked, confirmRecap: true })).toBe(true);
		expect(timelineSubstepShowsTaskPanel({ ...locked, justCompleted: true })).toBe(true);
		expect(timelineSubstepShowsTaskPanel({ ...locked, submissionCount: 1 })).toBe(true);
	});
});

describe('timeline-substep shell wiring', () => {
	test('one address host; body gate is showTaskPanel, not orgActionable-only mount', () => {
		const src = readFileSync(join(import.meta.dirname, 'timeline-substep.svelte'), 'utf8');
		expect(src).toContain('timelineSubstepShowsTaskPanel');
		expect(src).not.toContain('timelineSubstepUsesEmbed');
		expect(src).toMatch(/\{#if address\}/);
		expect(src).toMatch(/\{#if showTaskPanel\}/);
		expect(src).not.toMatch(
			/const embed = \$derived\(\!\!\(substep\.orgActionable && address\)\)/
		);
		// Recap chrome lives in EmbeddedTaskPanel, not a second accordion body.
		expect(src).not.toMatch(/import ManyRecapDoneBand/);
		expect(src).not.toMatch(/import BoardOnceRecap/);
	});
});
