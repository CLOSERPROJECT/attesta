import { readFileSync } from 'node:fs';
import { join } from 'node:path';

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
			refreshForms: true,
			clearUnderFormLinks: true
		});
	});

	test('once stay keeps Form behind once-recap (timeline refresh is session.afterDomainCommit)', () => {
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
