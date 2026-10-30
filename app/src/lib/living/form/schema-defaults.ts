import type { Schema } from '@sjsf/form';
import { createFormMerger } from '@sjsf/form/mergers/modern';
import { createAjvFormValidator } from './ajv-form-validator.js';

/** Authored sentinel — not a valid calendar date until resolved at form open. */
export const TODAY_DEFAULT_SENTINEL = '$today';

/** Local calendar date as `YYYY-MM-DD` (plant UI / operator timezone). */
export function localCalendarDateISO(now: Date = new Date()): string {
	const y = now.getFullYear();
	const m = String(now.getMonth() + 1).padStart(2, '0');
	const d = String(now.getDate()).padStart(2, '0');
	return `${y}-${m}-${d}`;
}

/**
 * Replace authored `default: $today` on `format: date` schemas with a real
 * ISO date. Config stays authority — other date fields are untouched.
 */
export function resolveSchemaDateDefaultSentinels(
	schema: Schema,
	today: string = localCalendarDateISO()
): Schema {
	return resolveNode(schema, today) as Schema;
}

function resolveNode(node: unknown, today: string): unknown {
	if (node === null || typeof node !== 'object' || Array.isArray(node)) {
		return node;
	}
	const schema = node as Schema & Record<string, unknown>;
	const out: Record<string, unknown> = { ...schema };

	if (schema.format === 'date' && schema.default === TODAY_DEFAULT_SENTINEL) {
		out.default = today;
	}

	if (schema.properties && typeof schema.properties === 'object') {
		const props: Record<string, unknown> = {};
		for (const [key, prop] of Object.entries(schema.properties)) {
			props[key] = resolveNode(prop, today);
		}
		out.properties = props;
	}

	if (schema.items !== undefined) {
		out.items = Array.isArray(schema.items)
			? schema.items.map((item) => resolveNode(item, today))
			: resolveNode(schema.items, today);
	}

	if (schema.$defs && typeof schema.$defs === 'object') {
		const defs: Record<string, unknown> = {};
		for (const [key, def] of Object.entries(schema.$defs as Record<string, unknown>)) {
			defs[key] = resolveNode(def, today);
		}
		out.$defs = defs;
	}

	return out;
}

/**
 * Merge JSON Schema defaults into formData (sjsf merger), after resolving
 * authored `$today` date sentinels. Preserves operator / partial drafts.
 */
export function mergeSchemaDefaultsIntoFormData<T extends Record<string, unknown>>(
	schema: Schema,
	formData: T,
	today: string = localCalendarDateISO()
): T {
	const resolvedSchema = resolveSchemaDateDefaultSentinels(schema, today);
	const validator = createAjvFormValidator<T>();
	const merger = createFormMerger({ validator, schema: resolvedSchema });
	return merger.mergeFormDataAndSchemaDefaults({
		schema: resolvedSchema,
		formData
	}) as T;
}
