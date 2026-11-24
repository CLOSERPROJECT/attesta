import type { UiSchemaRoot } from '@sjsf/form';

import type { ChromeEntry } from './task-form-plan.js';

type UiNode = Record<string, unknown>;

function asObject(value: unknown): UiNode {
	if (value != null && typeof value === 'object' && !Array.isArray(value)) {
		return { ...(value as UiNode) };
	}
	return {};
}

/**
 * Force seeded schema properties off editable enums: prefer readonlyField
 * (chrome paints `$.field` via chromeForSeededFields). Default is readonly;
 * pass hiddenField to omit paint.
 *
 * Compat resolver maps schema.enum → enumField (`@sjsf/form/resolvers/compat`),
 * so ui:components must override **enumField** (not only stringField). Authored
 * `stringField: enumField` alone never remaps that resolved type.
 */
export function lockSeededUiSchema(
	uiSchema: UiSchemaRoot,
	seededFields: readonly string[],
	component: 'readonlyField' | 'hiddenField' = 'readonlyField'
): UiSchemaRoot {
	if (seededFields.length === 0) return uiSchema;
	const next: UiNode = { ...(uiSchema as UiNode) };
	for (const field of seededFields) {
		if (!field) continue;
		const prior = asObject(next[field]);
		const components = asObject(prior['ui:components']);
		const options = asObject(prior['ui:options']);
		next[field] = {
			...prior,
			'ui:components': {
				...components,
				stringField: component,
				enumField: component
			},
			...(component === 'readonlyField'
				? {
						'ui:options': {
							...options,
							chromeId: field
						}
					}
				: { 'ui:options': options })
		};
	}
	return next as UiSchemaRoot;
}

/**
 * Chrome eval so readonlyField can paint the seeded draft value.
 * Do not add these ids to chromeFieldIds — they are real seal fields.
 */
export function chromeForSeededFields(
	seededFields: readonly string[],
	existing: readonly ChromeEntry[] = []
): ChromeEntry[] {
	if (seededFields.length === 0) return [...existing];
	const ids = new Set(existing.map((e) => e.id));
	const added: ChromeEntry[] = [];
	for (const field of seededFields) {
		if (!field || ids.has(field)) continue;
		added.push({ id: field, kind: 'readonly', expr: `$.${field}` });
		ids.add(field);
	}
	return added.length === 0 ? [...existing] : [...existing, ...added];
}
