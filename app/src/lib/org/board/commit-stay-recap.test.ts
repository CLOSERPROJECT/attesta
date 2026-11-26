import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, test } from 'vitest';

import {
	manyLandEmpty,
	manyShowComposer,
	stayAftermath,
	stayConfirmSurface
} from './commit-stay-recap.js';

describe('stayConfirmSurface', () => {
	test('submitMany stay lands on many-recap', () => {
		expect(stayConfirmSurface(true)).toBe('many-recap');
	});

	test('once stay lands on done/recap surface', () => {
		expect(stayConfirmSurface(false)).toBe('once-recap');
	});
});

describe('manyShowComposer / manyLandEmpty (Option C)', () => {
	test('empty land auto-opens the form', () => {
		expect(
			manyShowComposer({
				composerOpen: false,
				holdClosed: false,
				submissionCount: 0,
				confirmRecap: false
			})
		).toBe(true);
		expect(
			manyLandEmpty({ holdClosed: false, submissionCount: 0, confirmRecap: false })
		).toBe(true);
	});

	test('holdClosed after Commit closes even before confirmRecap/submissions', () => {
		// Register remountForms race: progressive chrome refetch while manyEmpty
		// would still be true if we only keyed off confirmRecap.
		expect(
			manyShowComposer({
				composerOpen: false,
				holdClosed: true,
				submissionCount: 0,
				confirmRecap: false
			})
		).toBe(false);
		expect(
			manyLandEmpty({ holdClosed: true, submissionCount: 0, confirmRecap: false })
		).toBe(false);
	});

	test('after Commit with submissions, composer stays closed until Add', () => {
		expect(
			manyShowComposer({
				composerOpen: false,
				holdClosed: true,
				submissionCount: 1,
				confirmRecap: true
			})
		).toBe(false);
		expect(
			manyShowComposer({
				composerOpen: true,
				holdClosed: false,
				submissionCount: 1,
				confirmRecap: true
			})
		).toBe(true);
	});

	test('second stay: holdClosed wins over composerOpen (Add then Commit)', () => {
		expect(
			manyShowComposer({
				composerOpen: true,
				holdClosed: true,
				submissionCount: 2,
				confirmRecap: true
			})
		).toBe(false);
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

describe('embedded-task-panel many stay close', () => {
	test('latches holdClosed on Commit and clears hold only via openComposer', () => {
		const panel = readFileSync(join(import.meta.dirname, 'embedded-task-panel.svelte'), 'utf8');
		expect(panel).toContain('holdComposerClosed');
		expect(panel).toContain('onFormCommitted');
		expect(panel).toContain('manyShowComposer');
		expect(panel).toMatch(/function openComposer\(\)[\s\S]*holdComposerClosed = false/);
		expect(panel).toMatch(
			/function onFormCommitted\([\s\S]*holdComposerClosed = true[\s\S]*composerOpen = false/
		);
	});
});
