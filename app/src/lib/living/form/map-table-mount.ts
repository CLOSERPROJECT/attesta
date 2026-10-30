/**
 * mapTable Substep mount (pure half): projected leftovers + authored Substep →
 * seed Draft rows and createConfigForm knobs (expanded Derived, `$domain`).
 * `map-table-mount.svelte.ts` turns knobs into live FormState (+ ConfigForm).
 */

import type { Schema, UiSchemaRoot } from '@sjsf/form';
import type { ConfigFormDerivedEntry } from './config-form.svelte.js';
import type {
	ConfigFormSchemas,
	CreateConfigFormOptions
} from './create-config-form.svelte.js';
import { expandMapDerived } from './derived-keys.js';
import { primaryDynamicProperty } from './dynamic-properties.js';
import { isMapTableTask } from './leftover-table-task.js';
import {
	nestObjectRowWeighMap,
	peelObjectRowWeighMap,
	seedEmptyObjectRowWeighRows
} from './object-row-weigh.js';

export type MapTableProjectedForm = {
	schema: Schema;
	uiSchema: UiSchemaRoot;
	leftoverIds: readonly string[];
};

export type MapTableAuthoredSubstep = {
	derived?: Record<string, ConfigFormDerivedEntry>;
	tareKinds?: Record<string, number>;
};

/**
 * Seed object-row keys from leftover ids; keep filled weight/panelCount.
 * New rows inherit form-level `tareKind` (default euro) + NIR idoneità.
 * mapTable / object-row weigh maps only — never leftoverTable progress cells.
 * Returns null when there is nothing to seed (empty leftovers, not mapTable, or no primary).
 */
export function seedMapTableDraft(
	draft: Record<string, unknown>,
	substep: unknown,
	leftoverIds: readonly string[]
): Record<string, unknown> | null {
	if (leftoverIds.length === 0) return null;
	if (!isMapTableTask(substep)) return null;
	const primary = primaryDynamicProperty(substep);
	if (primary === null) return null;
	const formTare =
		typeof draft.tareKind === 'string' && draft.tareKind.trim() !== ''
			? draft.tareKind
			: 'euro';
	const seeded = seedEmptyObjectRowWeighRows(
		peelObjectRowWeighMap(draft, primary),
		leftoverIds,
		{ tareKind: formTare, idoneita: 'NIR' }
	);
	const next = nestObjectRowWeighMap(draft, primary, seeded);
	if (next.tareKind === undefined) next.tareKind = formTare;
	return next;
}
/** Schemas + createConfigForm options (expand Derived, tareKinds `$domain`). */
export function prepareMapTableForm(
	projected: MapTableProjectedForm,
	substep: MapTableAuthoredSubstep
): {
	schemas: ConfigFormSchemas;
	options: Pick<CreateConfigFormOptions, 'derived' | 'getDomain'>;
} {
	const derived =
		substep.derived !== undefined
			? expandMapDerived(substep.derived, projected.leftoverIds)
			: undefined;
	const tareKinds = substep.tareKinds;
	return {
		schemas: { schema: projected.schema, uiSchema: projected.uiSchema },
		options: {
			...(derived !== undefined ? { derived } : {}),
			...(tareKinds !== undefined
				? { getDomain: () => ({ tareKinds }) }
				: {})
		}
	};
}
