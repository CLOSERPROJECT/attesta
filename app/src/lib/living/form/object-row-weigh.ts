/**
 * Object-row weigh Draft shape: `Record<id, { weight, tareKind, idoneita, panelCount }>`
 * under the primary dynamicProperties key. Nest / peel / seed-empty / filter-filled —
 * Session seed and Submission prep share this interface; UI only renders.
 */

import { primaryDynamicProperty } from './dynamic-properties.js';

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

/** Column keys on each object-row weigh cell (schema additionalProperties.properties). */
export const OBJECT_ROW_WEIGH_COLUMNS = [
	'weight',
	'tareKind',
	'idoneita',
	'panelCount'
] as const;

export type ObjectRowWeighSeedDefaults = {
	tareKind?: string;
	idoneita?: string;
};

/** Empty Draft cell — '' weight for untouched sjsf number inputs; panelCount 0. */
export type ObjectRowWeighEmptyCell = {
	weight: '';
	tareKind: string;
	idoneita: string;
	panelCount: 0;
};

/** Object-row weigh fact kept for patchEach (`$value.*`). */
export type WeighRowFact = {
	weight: number;
	tareKind: string;
	idoneita: string;
	panelCount: number;
};

export type FilterFilledObjectWeightsResult =
	| { ok: true; filled: Record<string, WeighRowFact> }
	| { ok: false; error: string };

const IDONEITA_VALUES = new Set(['NIR', 'IR']);

export function emptyObjectRowWeighCell(
	defaults?: ObjectRowWeighSeedDefaults
): ObjectRowWeighEmptyCell {
	return {
		weight: '',
		tareKind: defaults?.tareKind ?? 'euro',
		idoneita: defaults?.idoneita ?? 'NIR',
		panelCount: 0
	};
}

/** Filled row fact with euro/NIR defaults (tests + harness). */
export function weighRowFact(
	weight: number,
	panelCount: number,
	opts?: Partial<Pick<WeighRowFact, 'tareKind' | 'idoneita'>>
): WeighRowFact {
	return {
		weight,
		panelCount,
		tareKind: opts?.tareKind ?? 'euro',
		idoneita: opts?.idoneita ?? 'NIR'
	};
}
/** Peel the id→row map nested under the primary dynamic property. */
export function peelObjectRowWeighMap(
	value: Record<string, unknown>,
	primaryKey: string
): Record<string, unknown> {
	const nested = value[primaryKey];
	return isPlainObject(nested) ? nested : {};
}

/** Nest an id→row map under the primary key; preserve sibling form fields. */
export function nestObjectRowWeighMap(
	value: Record<string, unknown>,
	primaryKey: string,
	map: Record<string, unknown>
): Record<string, unknown> {
	return { ...value, [primaryKey]: map };
}

/**
 * Seed missing leftover ids with empty cells; keep existing plain-object rows
 * (filled weight/panelCount or partial edits). New rows inherit form-level tare
 * default (and NIR idoneità) when provided.
 */
export function seedEmptyObjectRowWeighRows(
	map: Record<string, unknown>,
	leftoverIds: readonly string[],
	defaults?: ObjectRowWeighSeedDefaults
): Record<string, unknown> {
	const nested = { ...map };
	for (const id of leftoverIds) {
		if (isPlainObject(nested[id])) continue;
		nested[id] = emptyObjectRowWeighCell(defaults);
	}
	return nested;
}

/** Object-row weigh map (`additionalProperties.type: object`) vs flat number cells. */
export function isObjectRowWeightMap(substep: unknown): boolean {
	if (!isPlainObject(substep) || !isPlainObject(substep.schema)) return false;
	const primaryKey = primaryDynamicProperty(substep);
	if (primaryKey === null) return false;
	const props = isPlainObject(substep.schema.properties) ? substep.schema.properties : null;
	const propSchema = props && isPlainObject(props[primaryKey]) ? props[primaryKey] : null;
	const additional = propSchema ? propSchema.additionalProperties : null;
	return isPlainObject(additional) && additional.type === 'object';
}

function isEmptyWeightCell(cell: unknown): boolean {
	if (cell === undefined || cell === null) return true;
	if (typeof cell === 'string' && cell.trim() === '') return true;
	return false;
}

/**
 * Object-row weigh map → filled row facts.
 * Empty/missing weight omits the row; filled weight requires finite panelCount,
 * tareKind, and idoneita (IR|NIR).
 */
export function filterFilledObjectWeights(
	value: Record<string, unknown>
): FilterFilledObjectWeightsResult {
	const filled: Record<string, WeighRowFact> = {};
	for (const [id, cell] of Object.entries(value)) {
		if (!isPlainObject(cell)) {
			if (isEmptyWeightCell(cell)) continue;
			return { ok: false, error: `invalid weight for ${id}` };
		}
		const weight = cell.weight;
		if (isEmptyWeightCell(weight)) continue;
		if (typeof weight !== 'number' || !Number.isFinite(weight)) {
			return { ok: false, error: `invalid weight for ${id}` };
		}
		const panelCount = cell.panelCount;
		if (typeof panelCount !== 'number' || !Number.isFinite(panelCount)) {
			return { ok: false, error: `invalid panelCount for ${id}` };
		}
		const tareKind = cell.tareKind;
		if (typeof tareKind !== 'string' || tareKind.trim() === '') {
			return { ok: false, error: `invalid tareKind for ${id}` };
		}
		const idoneita = cell.idoneita;
		if (typeof idoneita !== 'string' || !IDONEITA_VALUES.has(idoneita)) {
			return { ok: false, error: `invalid idoneita for ${id}` };
		}
		filled[id] = { weight, tareKind, idoneita, panelCount };
	}
	return { ok: true, filled };
}

/** Column headers from a row schema's `properties` (map-table; empty → no columns). */
export function objectRowMapColumnKeys(rowProperties: unknown): string[] {
	if (!isPlainObject(rowProperties)) return [];
	return Object.keys(rowProperties);
}

/**
 * Merge one column into an object-row cell map; empty clears the column.
 * Returns undefined when the row has no remaining keys (drop from parent map).
 */
export function writeObjectRowMapCell(
	row: Record<string, unknown>,
	column: string,
	cell: unknown
): Record<string, unknown> | undefined {
	const next = { ...row };
	if (cell === undefined || cell === null || cell === '') {
		delete next[column];
	} else {
		next[column] = cell;
	}
	return Object.keys(next).length === 0 ? undefined : next;
}
