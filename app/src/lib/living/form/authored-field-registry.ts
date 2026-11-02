/**
 * Authored `ui:field` → sjsf registered field (ADR-0009 component resolution).
 * One row per marker: theme overrides share this map.
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
	{ authored: 'mapTable', componentKey: 'mapTableField', slot: 'objectField' },
	{ authored: 'derivedLeaf', componentKey: 'derivedLeafField', slot: 'numberField' },
	{ authored: 'checkbox-only', componentKey: 'checkboxOnlyField', slot: 'booleanField' }
] as const satisfies readonly AuthoredFieldRow[];

export type AuthoredFieldComponentKey =
	(typeof AUTHORED_FIELD_REGISTRY)[number]['componentKey'];

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
		mapTableField: {};
		derivedLeafField: {};
		checkboxOnlyField: {};
	}
	interface ComponentProps {
		mapTableField: FieldCommonProps<SchemaObjectValue>;
		derivedLeafField: FieldCommonProps<number | string | undefined>;
		checkboxOnlyField: FieldCommonProps<boolean | undefined>;
	}
	interface ComponentBindings {
		mapTableField: 'value';
		derivedLeafField: 'value';
		checkboxOnlyField: 'value';
	}
	interface ActionFields {
		mapTableField: {};
		derivedLeafField: {};
		checkboxOnlyField: {};
	}
}
