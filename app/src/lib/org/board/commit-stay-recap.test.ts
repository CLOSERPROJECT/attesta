import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, test } from 'vitest';

import {
	initialManyLand,
	reduceManyLand,
	showComposer,
	showDoneBand,
	stayAftermath,
	stayConfirmSurface,
	type ManyLandSurface
} from './commit-stay-recap.js';

describe('stayConfirmSurface', () => {
	test('submitMany stay lands on many-recap', () => {
		expect(stayConfirmSurface(true)).toBe('many-recap');
	});

	test('once stay lands on done/recap surface', () => {
		expect(stayConfirmSurface(false)).toBe('once-recap');
	});
});

describe('ManyLandSurface (Option C)', () => {
	test('empty land → empty-compose: form open, Done-band hidden', () => {
		const surface = initialManyLand({ submissionCount: 0, confirmRecap: false });
		expect(surface).toBe('empty-compose');
		expect(showComposer(surface)).toBe(true);
		expect(showDoneBand(surface)).toBe(false);
	});

	test('land with submissions or confirmRecap → done-band', () => {
		expect(initialManyLand({ submissionCount: 1, confirmRecap: false })).toBe('done-band');
		expect(initialManyLand({ submissionCount: 0, confirmRecap: true })).toBe('done-band');
		expect(showComposer('done-band')).toBe(false);
		expect(showDoneBand('done-band')).toBe(true);
	});

	test('commitSuccess closes composer before confirmRecap/submissions catch-up', () => {
		// Register remountForms race: progressive chrome refetch must not reopen
		// the form — mode is done-band, not inferred from manyEmpty.
		const afterCommit = reduceManyLand('composing', { type: 'commitSuccess' });
		expect(afterCommit).toBe('done-band');
		expect(showComposer(afterCommit)).toBe(false);
		expect(showDoneBand(afterCommit)).toBe(true);

		const fromEmpty = reduceManyLand('empty-compose', { type: 'commitSuccess' });
		expect(fromEmpty).toBe('done-band');
		expect(showComposer(fromEmpty)).toBe(false);
	});

	test('openComposer → composing: form open, Done-band stays', () => {
		const surface = reduceManyLand('done-band', { type: 'openComposer' });
		expect(surface).toBe('composing');
		expect(showComposer(surface)).toBe(true);
		expect(showDoneBand(surface)).toBe(true);
	});

	test('second stay: Commit from composing → done-band (Add then Commit)', () => {
		let surface: ManyLandSurface = 'done-band';
		surface = reduceManyLand(surface, { type: 'openComposer' });
		surface = reduceManyLand(surface, { type: 'commitSuccess' });
		expect(surface).toBe('done-band');
		expect(showComposer(surface)).toBe(false);
	});

	test('discard → empty-compose when N=0, else done-band', () => {
		expect(
			reduceManyLand('composing', { type: 'discard', submissionCount: 0 })
		).toBe('empty-compose');
		expect(
			reduceManyLand('composing', { type: 'discard', submissionCount: 2 })
		).toBe('done-band');
	});

	test('visibility is mode-only (no submissionCount / confirmRecap race)', () => {
		expect(showComposer('empty-compose')).toBe(true);
		expect(showComposer('composing')).toBe(true);
		expect(showComposer('done-band')).toBe(false);
		expect(showDoneBand('empty-compose')).toBe(false);
		expect(showDoneBand('composing')).toBe(true);
		expect(showDoneBand('done-band')).toBe(true);
	});
});

describe('stayAftermath', () => {
	test('submitMany stay refreshes form bundles and lands on many-recap', () => {
		expect(stayAftermath(true)).toEqual({
			confirmSurface: 'many-recap',
			refreshForms: true,
			clearUnderFormLinks: true
		});
	});

	test('once stay keeps Form behind once-recap (timeline refresh is session.domainChanged)', () => {
		expect(stayAftermath(false)).toEqual({
			confirmSurface: 'once-recap',
			refreshForms: false,
			clearUnderFormLinks: true
		});
	});

	test('stay always clears under-form HttpLinkStrip (links live on recap tables)', () => {
		expect(stayAftermath(true).clearUnderFormLinks).toBe(true);
		expect(stayAftermath(false).clearUnderFormLinks).toBe(true);
	});
});

describe('org-board stay wires clearUnderFormLinks', () => {
	test('onCommitted stay path clears session leave links before recap', () => {
		const board = readFileSync(join(import.meta.dirname, 'org-board.svelte'), 'utf8');
		expect(board).toContain('aftermath.clearUnderFormLinks');
		expect(board).toContain('orgSession.clearLinksStay()');
	});
});

describe('embedded-task-panel many land surface', () => {
	test('dispatches reduceManyLand on Commit / Add / Discard; lands via initialManyLand', () => {
		const panel = readFileSync(join(import.meta.dirname, 'embedded-task-panel.svelte'), 'utf8');
		expect(panel).toContain('manySurface');
		expect(panel).toContain('initialManyLand');
		expect(panel).toContain('reduceManyLand');
		expect(panel).toContain('showComposer');
		expect(panel).toContain('showDoneBand');
		expect(panel).toContain('ManyRecapDoneBand');
		expect(panel).toContain('orgActionable');
		expect(panel).not.toContain('holdComposerClosed');
		expect(panel).not.toContain('composerOpen');
		expect(panel).toMatch(
			/function openComposer\(\)[\s\S]*reduceManyLand\(manySurface, \{ type: 'openComposer' \}\)/
		);
		expect(panel).toMatch(
			/function onFormCommitted\([\s\S]*reduceManyLand\(manySurface, \{ type: 'commitSuccess' \}\)/
		);
		expect(panel).toMatch(
			/function discardComposer\(\)[\s\S]*type: 'discard'/
		);
		// Locked stay land: Done-band only (no Add / composer region).
		expect(panel).toMatch(/\{#if orgActionable\}[\s\S]*embed-composer-region/);
	});
});
