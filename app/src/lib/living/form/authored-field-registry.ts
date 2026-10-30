/**
 * Authored `ui:field` → sjsf registered field (ADR-0009 component resolution).
 * One row per marker: paint (`ui:components`) and theme overrides share this map.
 * Field `.svelte` modules only implement UI; types live here.
 */

import type { FieldCommonProps } from '@sjsf/form';
import type { SchemaObjectValue } from '@sjsf/form/core';

export type AuthoredFieldSlot = 'objectField' | 'numberField' | 'booleanField';

export type AuthoredFieldRow = {
	/** YAML / fixture `ui:field` marker. */
	authored: string;
	/** Theme + `ui:components` key (FoundationalComponents). */
	componentKey: string;
	/** Which stock field slot the marker replaces. */
	slot: AuthoredFieldSlot;
};

/** Sole authored → resolved name map. New `ui:field` = one row. */
export const AUTHORED_FIELD_REGISTRY = [
	{ authored: 'leftoverTable', componentKey: 'leftoverTableField', slot: 'objectField' },
	{ authored: 'mapTable', componentKey: 'mapTableField', slot: 'objectField' },
	{ authored: 'derivedLeaf', componentKey: 'derivedLeafField', slot: 'numberField' },
	{ authored: 'checkbox-only', componentKey: 'checkboxOnlyField', slot: 'booleanField' }
] as const satisfies readonly AuthoredFieldRow[];

export type AuthoredFieldComponentKey =
	(typeof AUTHORED_FIELD_REGISTRY)[number]['componentKey'];

const BY_AUTHORED: ReadonlyMap<string, AuthoredFieldRow> = new Map(
	AUTHORED_FIELD_REGISTRY.map((row) => [row.authored, row])
);

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function asRecord(value: unknown): Record<string, unknown> | null {
	return isPlainObject(value) ? value : null;
}

/**
 * Map authored `ui:field` markers to `ui:components` via the registry; recurse
 * `additionalProperties`. Also forces `expandable: false` on each painted node.
 */
export function paintAuthoredFieldMarkers(uiNode: Record<string, unknown>): Record<string, unknown> {
	const next = { ...uiNode };
	const field = next['ui:field'];
	if (typeof field === 'string') {
		const row = BY_AUTHORED.get(field);
		if (row) {
			delete next['ui:field'];
			const existing = asRecord(next['ui:components']) ?? {};
			next['ui:components'] = {
				...existing,
				[row.slot]: row.componentKey
			};
		}
	}

	const additional = asRecord(next.additionalProperties);
	if (additional) {
		const paintedAdditional: Record<string, unknown> = {};
		for (const [key, child] of Object.entries(additional)) {
			paintedAdditional[key] = isPlainObject(child) ? paintAuthoredFieldMarkers(child) : child;
		}
		next.additionalProperties = paintedAdditional;
	}

	const options = asRecord(next['ui:options']) ?? {};
	next['ui:options'] = {
		...options,
		expandable: false
	};
	return next;
}

/** Theme override record: every registry `componentKey` must supply a component. */
export function authoredFieldThemeOverrides<
	T extends { [K in AuthoredFieldComponentKey]: unknown }
>(components: T): Pick<T, AuthoredFieldComponentKey> {
	const out = {} as Pick<T, AuthoredFieldComponentKey>;
	for (const row of AUTHORED_FIELD_REGISTRY) {
		const key = row.componentKey;
		out[key] = components[key];
	}
	return out;
}

declare module '@sjsf/form' {
	interface FoundationalComponents {
		leftoverTableField: {};
		mapTableField: {};
		derivedLeafField: {};
		checkboxOnlyField: {};
	}
	interface ComponentProps {
		leftoverTableField: FieldCommonProps<SchemaObjectValue>;
		mapTableField: FieldCommonProps<SchemaObjectValue>;
		derivedLeafField: FieldCommonProps<number | string | undefined>;
		checkboxOnlyField: FieldCommonProps<boolean | undefined>;
	}
	interface ComponentBindings {
		leftoverTableField: 'value';
		mapTableField: 'value';
		derivedLeafField: 'value';
		checkboxOnlyField: 'value';
	}
	interface ActionFields {
		leftoverTableField: {};
		mapTableField: {};
		derivedLeafField: {};
		checkboxOnlyField: {};
	}
}
