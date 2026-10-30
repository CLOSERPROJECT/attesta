import type { Schema, UiSchemaRoot } from '@sjsf/form';
import { isChromeTone, type ChromeTone, type ChromeToneBand } from './chrome-tone.js';
import { paintContainerRegCompactArrayUi } from './compact-array-item-ui.js';
import { injectDefaultPlaceholders } from './default-field-placeholders.js';
import { expandMapDerived, type DerivedKeyEntry } from './derived-keys.js';
import { isContainerRegTask } from './flatten-container-reg.js';
import {
	hasEnumReads,
	isLeftoverTableTask,
	isMapTableTask
} from './leftover-table-task.js';
import { leftoversIdentity } from './project-form-schema.js';

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

export type TaskFormPlan = {
	surface: 'live' | 'leftover-table';
	schemas: { schema: Schema; uiSchema: UiSchemaRoot };
	derived?: Record<string, DerivedKeyEntry>;
	chrome: readonly ChromeEntry[];
	chromeFieldIds: readonly string[];
	projectionKey: string;
	sealSubmission: (value: Record<string, unknown>) => Record<string, unknown>;
};

export type CompileTaskFormPlanContext = {
	leftoverIds?: readonly string[];
	/** Projected paint (mapTable leftovers and/or enumReads) replaces authored schema/ui. */
	projectedSchemas?: { schema: Schema; uiSchema: UiSchemaRoot };
};

export type LiveProjectedFormSlice = {
	leftoverIds: readonly string[];
	schema: Schema;
	uiSchema: UiSchemaRoot;
};

/**
 * Session seam: whether live compile can proceed and which projection ctx to pass.
 * Leftover-table stays on TableFormShell; mapTable/enumReads wait for projection.
 */
export function buildLiveTaskPlanContext(
	authored: unknown,
	projected: LiveProjectedFormSlice | null | undefined
):
	| { kind: 'skip' }
	| { kind: 'awaiting-projection' }
	| { kind: 'ready'; ctx?: CompileTaskFormPlanContext } {
	if (isLeftoverTableTask(authored)) return { kind: 'skip' };
	const needsProjected =
		isMapTableTask(authored) || hasEnumReads(authored);
	if (needsProjected && !projected) return { kind: 'awaiting-projection' };
	if (!needsProjected || !projected) return { kind: 'ready' };
	return {
		kind: 'ready',
		ctx: {
			leftoverIds: projected.leftoverIds,
			projectedSchemas: {
				schema: projected.schema,
				uiSchema: projected.uiSchema
			}
		}
	};
}

const MAP_STAR = /^(.+)\.\*\.(.+)$/;

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function asRecord(value: unknown): Record<string, unknown> | null {
	return isPlainObject(value) ? value : null;
}

function readBands(raw: unknown): ChromeToneBand[] | undefined {
	if (!Array.isArray(raw) || raw.length === 0) return undefined;
	const bands: ChromeToneBand[] = [];
	for (const entry of raw) {
		const rec = asRecord(entry);
		if (!rec || typeof rec.min !== 'number' || !Number.isFinite(rec.min)) continue;
		if (!isChromeTone(rec.tone)) continue;
		bands.push({ min: rec.min, tone: rec.tone });
	}
	return bands.length > 0 ? bands : undefined;
}

function readRecapRows(raw: unknown): ChromeRecapRow[] {
	if (!Array.isArray(raw)) return [];
	const rows: ChromeRecapRow[] = [];
	for (const entry of raw) {
		const rec = asRecord(entry);
		if (!rec || typeof rec.label !== 'string' || typeof rec.expr !== 'string') continue;
		const tone = isChromeTone(rec.tone) ? rec.tone : undefined;
		const bands = readBands(rec.bands);
		rows.push({
			label: rec.label,
			expr: rec.expr,
			...(tone !== undefined ? { tone } : {}),
			...(bands !== undefined ? { bands } : {})
		});
	}
	return rows;
}

function readChrome(substep: Record<string, unknown>): ChromeEntry[] {
	const raw = asRecord(substep.chrome);
	if (!raw) return [];
	const out: ChromeEntry[] = [];
	for (const [id, entry] of Object.entries(raw)) {
		const rec = asRecord(entry);
		if (!rec) continue;
		const after = typeof rec.after === 'string' ? rec.after : undefined;
		const before = typeof rec.before === 'string' ? rec.before : undefined;
		if (rec.kind === 'valuePrint' && typeof rec.expr === 'string') {
			out.push({
				id,
				kind: 'valuePrint',
				expr: rec.expr,
				...(after !== undefined ? { after } : {}),
				...(before !== undefined ? { before } : {})
			});
			continue;
		}
		if (rec.kind === 'readonly' && typeof rec.expr === 'string') {
			out.push({
				id,
				kind: 'readonly',
				expr: rec.expr,
				...(after !== undefined ? { after } : {}),
				...(before !== undefined ? { before } : {})
			});
			continue;
		}
		if (rec.kind === 'recap') {
			const rows = readRecapRows(rec.rows);
			const title = typeof rec.title === 'string' && rec.title.length > 0 ? rec.title : undefined;
			out.push({
				id,
				kind: 'recap',
				rows,
				...(title !== undefined ? { title } : {}),
				...(after !== undefined ? { after } : {}),
				...(before !== undefined ? { before } : {})
			});
			continue;
		}
		if (rec.kind === 'finalResults') {
			const title = typeof rec.title === 'string' && rec.title.length > 0 ? rec.title : undefined;
			out.push({
				id,
				kind: 'finalResults',
				...(title !== undefined ? { title } : {}),
				...(after !== undefined ? { after } : {}),
				...(before !== undefined ? { before } : {})
			});
		}
	}
	return out;
}

function chromeTitle(entry: Record<string, unknown>, id: string, parsed: ChromeEntry): string {
	if ((parsed.kind === 'recap' || parsed.kind === 'finalResults') && parsed.title) {
		return parsed.title;
	}
	if (typeof entry.title === 'string' && entry.title.length > 0) return entry.title;
	return id;
}

function insertAfterOrder(order: string[], after: string | undefined, id: string): string[] {
	if (order.includes(id)) return order;
	if (after && order.includes(after)) {
		const next = [...order];
		next.splice(next.indexOf(after) + 1, 0, id);
		return next;
	}
	return [...order, id];
}

/** Mirror of insertAfterOrder: place id immediately before the named field. */
export function insertBeforeOrder(order: string[], before: string | undefined, id: string): string[] {
	if (order.includes(id)) return order;
	if (before && order.includes(before)) {
		const next = [...order];
		next.splice(next.indexOf(before), 0, id);
		return next;
	}
	return [...order, id];
}

function placeChromeInOrder(
	order: string[],
	entry: { id: string; before?: string; after?: string }
): string[] {
	if (entry.before !== undefined) {
		return insertBeforeOrder(order, entry.before, entry.id);
	}
	return insertAfterOrder(order, entry.after, entry.id);
}

function authoredPropertyKeys(schema: Record<string, unknown>): Set<string> {
	const props = asRecord(schema.properties);
	return new Set(props ? Object.keys(props) : []);
}

function concreteDerivedOnly(
	derived: Record<string, DerivedKeyEntry>
): Record<string, DerivedKeyEntry> {
	const out: Record<string, DerivedKeyEntry> = {};
	for (const [key, entry] of Object.entries(derived)) {
		if (!MAP_STAR.test(key)) out[key] = entry;
	}
	return out;
}

function stringFieldFor(
	kind: ChromeEntry['kind']
): 'valuePrintField' | 'recapField' | 'readonlyField' | 'feragameFinalResultsField' {
	if (kind === 'recap') return 'recapField';
	if (kind === 'readonly') return 'readonlyField';
	if (kind === 'finalResults') return 'feragameFinalResultsField';
	return 'valuePrintField';
}

function mergeChromeIntoSchemas(
	authoredSchema: Schema,
	authoredUi: Record<string, unknown>,
	chrome: ChromeEntry[],
	chromeRaw: Record<string, unknown>
): { schema: Schema; uiSchema: UiSchemaRoot } {
	const sealProps = (asRecord(authoredSchema.properties) ?? {}) as Record<string, Schema>;
	const sealRequired = Array.isArray(authoredSchema.required)
		? authoredSchema.required.filter((k): k is string => typeof k === 'string')
		: [];

	const chromeProps: Record<string, Schema> = {};
	for (const c of chrome) {
		const entry = asRecord(chromeRaw[c.id]) ?? {};
		chromeProps[c.id] = {
			type: 'string',
			title: chromeTitle(entry, c.id, c),
			readOnly: true
		};
	}

	const mergedProps: Record<string, Schema> = { ...sealProps, ...chromeProps };

	const uiOptions = asRecord(authoredUi['ui:options']) ?? {};
	let order = Array.isArray(uiOptions.order)
		? uiOptions.order.filter((k): k is string => typeof k === 'string')
		: Object.keys(mergedProps);

	for (const c of chrome) {
		order = placeChromeInOrder(order, c);
	}

	const uiSchema: Record<string, unknown> = {
		...authoredUi,
		'ui:options': {
			...uiOptions,
			order
		}
	};

	for (const c of chrome) {
		uiSchema[c.id] = {
			'ui:components': {
				stringField: stringFieldFor(c.kind)
			},
			'ui:options': {
				chromeId: c.id
			}
		};
	}

	return {
		schema: {
			...authoredSchema,
			type: 'object',
			properties: mergedProps,
			required: sealRequired
		},
		uiSchema: uiSchema as UiSchemaRoot
	};
}

function stripChromeFields(
	value: Record<string, unknown>,
	chromeFieldIds: readonly string[]
): Record<string, unknown> {
	if (chromeFieldIds.length === 0) return value;
	const out = { ...value };
	for (const id of chromeFieldIds) {
		delete out[id];
	}
	return out;
}

function compileDerived(
	authored: Record<string, unknown>,
	ctx: CompileTaskFormPlanContext | undefined
): Record<string, DerivedKeyEntry> | undefined {
	const raw = authored.derived;
	if (!isPlainObject(raw)) return undefined;
	const derived: Record<string, DerivedKeyEntry> = {};
	for (const [key, entry] of Object.entries(raw)) {
		if (!isPlainObject(entry) || typeof entry.expr !== 'string') continue;
		derived[key] = {
			expr: entry.expr,
			...(entry.overridable !== undefined ? { overridable: entry.overridable === true } : {})
		};
	}
	if (Object.keys(derived).length === 0) return undefined;

	const leftoverIds = ctx?.leftoverIds;
	if (leftoverIds !== undefined && leftoverIds.length > 0) {
		return expandMapDerived(derived, leftoverIds);
	}
	const concrete = concreteDerivedOnly(derived);
	return Object.keys(concrete).length > 0 ? concrete : undefined;
}

function projectionKeyFor(
	authored: Record<string, unknown>,
	ctx: CompileTaskFormPlanContext | undefined
): string {
	if (!isMapTableTask(authored)) return '';
	const ids = ctx?.leftoverIds;
	if (ids === undefined || ids.length === 0) return '';
	return leftoversIdentity([...ids]);
}

/**
 * Pure compile: authored Substep YAML (+ optional projection ctx) → frozen FormPlan.
 */
export function compileTaskFormPlan(
	authored: unknown,
	ctx?: CompileTaskFormPlanContext
): TaskFormPlan | { error: string } {
	if (!isPlainObject(authored)) {
		return { error: 'authored substep must be an object' };
	}

	const authoredSchema = (asRecord(authored.schema) ?? {
		type: 'object',
		properties: {}
	}) as Schema;
	const authoredUi = asRecord(authored.uiSchema) ?? {};
	const displaySchema = ctx?.projectedSchemas?.schema ?? authoredSchema;
	const displayUi = ctx?.projectedSchemas?.uiSchema ?? (authoredUi as UiSchemaRoot);

	const chrome = readChrome(authored);
	const chromeRaw = asRecord(authored.chrome) ?? {};

	for (const c of chrome) {
		if (authoredPropertyKeys(displaySchema as Record<string, unknown>).has(c.id)) {
			return { error: `chrome id "${c.id}" collides with authored property` };
		}
	}

	const baseSchemas =
		chrome.length === 0
			? {
					schema: displaySchema,
					uiSchema: displayUi
				}
			: mergeChromeIntoSchemas(
					displaySchema,
					asRecord(displayUi) ?? {},
					chrome,
					chromeRaw
				);

	const chromeFieldIds = chrome.map((c) => c.id);
	const schemas = withDefaultPlaceholders(baseSchemas, chromeFieldIds, authored);
	const derived = compileDerived(authored, ctx);

	return {
		surface: isLeftoverTableTask(authored) ? 'leftover-table' : 'live',
		schemas,
		...(derived !== undefined ? { derived } : {}),
		chrome,
		chromeFieldIds,
		projectionKey: projectionKeyFor(authored, ctx),
		sealSubmission: (value) => stripChromeFields(value, chromeFieldIds)
	};
}

/**
 * Attesta runner: never show root schema title as form FieldSet legend / title row.
 * Accordion labels still read authored `schema.title` via substepTitleFromConfig.
 */
export function paintHideRootFormTitle(uiSchema: Record<string, unknown>): void {
	const existingOpts = asRecord(uiSchema['ui:options']) ?? {};
	uiSchema['ui:options'] = {
		...existingOpts,
		hideTitle: true
	};
}

/** Clone uiSchema then inject defaults so authored / projected objects stay pristine. */
function withDefaultPlaceholders(
	schemas: { schema: Schema; uiSchema: UiSchemaRoot },
	chromeFieldIds: readonly string[],
	authored: Record<string, unknown>
): { schema: Schema; uiSchema: UiSchemaRoot } {
	const properties = (asRecord((schemas.schema as Record<string, unknown>).properties) ??
		{}) as Record<string, Schema>;
	const uiSchema: Record<string, unknown> = { ...(asRecord(schemas.uiSchema) ?? {}) };
	paintHideRootFormTitle(uiSchema);
	// Compact paint before defaults so sourceLoadId / nested weight placeholders win;
	// injectDefaultPlaceholders then skips fields that already have a placeholder.
	if (isContainerRegTask(authored)) {
		paintContainerRegCompactArrayUi(uiSchema);
	}
	injectDefaultPlaceholders(uiSchema, properties, new Set(chromeFieldIds));
	return { schema: schemas.schema, uiSchema: uiSchema as UiSchemaRoot };
}
