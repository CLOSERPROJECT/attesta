import { getContext, hasContext, setContext } from 'svelte';
import type { ChromeTone } from './chrome-tone.js';

export type ChromeRecapRowDisplay = {
	label: string;
	value: string | null;
	error: string | null;
	tone: ChromeTone | null;
};

/** One primary-load section for `kind: dataReport` (server chromeDisplay fills). */
export type ChromeDataReportSectionDisplay = {
	loadId: string;
	/** Dual secondary load ids — muted chrome text only, never their own section. */
	secondaryLoadIds?: readonly string[];
	rows?: readonly ChromeRecapRowDisplay[];
};

export type ChromeDisplay = {
	value: string | null;
	error: string | null;
	/** Present for `kind: dataReport` chrome. */
	title?: string | null;
	/**
	 * Flat row list for flat `kind: dataReport` (e.g. weightCheck).
	 * When non-empty and `sections` empty/absent, dataReportField paints title+rows.
	 */
	rows?: readonly ChromeRecapRowDisplay[];
	/**
	 * Multi-load sections for `kind: dataReport` (e.g. loadedExtracted).
	 * Sections mode when non-empty; flat `rows` paint when sections empty/absent.
	 */
	sections?: readonly ChromeDataReportSectionDisplay[];
};

export type ChromeContext = {
	get: (id: string) => ChromeDisplay;
};

const KEY = Symbol('attesta-chrome-context');

export function setChromeContext(ctx: ChromeContext): void {
	setContext(KEY, ctx);
}

export function getChromeContext(): ChromeContext {
	if (!hasContext(KEY)) {
		throw new Error('chrome context is missing — mount ConfigForm with LiveTaskForm first');
	}
	return getContext(KEY);
}
