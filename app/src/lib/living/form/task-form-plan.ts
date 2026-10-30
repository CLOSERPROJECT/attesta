import type { Schema, UiSchemaRoot } from '@sjsf/form';
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

/** Server-eval display chrome; SPA paints `bindings.chromeDisplay[id]` only. */
export type ChromeDataReport = {
	id: string;
	kind: 'dataReport';
	title?: string;
	before?: string;
	after?: string;
	/** Authored on server; optional on wire if Go strips it after eval. */
	expr?: string;
};

export type ChromeEntry = ChromeValuePrint | ChromeReadonly | ChromeDataReport;

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
