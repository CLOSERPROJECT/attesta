import type { Schema, UiSchemaRoot } from '@sjsf/form';
import type { ChromeTone, ChromeToneBand } from './chrome-tone.js';
import type { DerivedKeyEntry } from './derived-keys.js';

export type ChromeValuePrint = {
	id: string;
	kind: 'valuePrint';
	expr: string;
	after?: string;
	before?: string;
};

export type ChromeReadonly = {
	id: string;
	kind: 'readonly';
	expr: string;
	after?: string;
	before?: string;
};

export type ChromeRecapRow = {
	label: string;
	expr: string;
	tone?: ChromeTone;
	bands?: ChromeToneBand[];
};

export type ChromeRecap = {
	id: string;
	kind: 'recap';
	title?: string;
	before?: string;
	after?: string;
	rows: ChromeRecapRow[];
};

/** Presentation-only C7 chrome; payload from projector via chrome context. */
export type ChromeFinalResults = {
	id: string;
	kind: 'finalResults';
	title?: string;
	before?: string;
	after?: string;
};

export type ChromeEntry = ChromeValuePrint | ChromeReadonly | ChromeRecap | ChromeFinalResults;

/** Frozen FormPlan — produced from a server TaskFormBundle, never client compile. */
export type TaskFormPlan = {
	surface: 'live' | 'leftover-table';
	schemas: { schema: Schema; uiSchema: UiSchemaRoot };
	derived?: Record<string, DerivedKeyEntry>;
	chrome: readonly ChromeEntry[];
	chromeFieldIds: readonly string[];
	projectionKey: string;
	sealSubmission: (value: Record<string, unknown>) => Record<string, unknown>;
};
