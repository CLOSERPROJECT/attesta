/**
 * Flatten keyed material arrays → mintMany rows.
 * Material fact = human label; empty / incomplete weight rows are skipped.
 * Dual-load UX: per-row dualSources uses form-level secondarySourceLoadId ≠ primary;
 * dual off strips secondary on that row. Never secondaryWeight.
 */

import { CONTAINER_REG_MATERIALS } from './container-reg-materials.js';

export type ContainerRegRow = {
	weight: number;
	material: string;
	secondarySourceLoadId?: string;
};

export type FlattenContainerRegOk = {
	sourceLoadId: string;
	rows: ContainerRegRow[];
};

export type FlattenContainerRegResult =
	| { ok: true; facts: FlattenContainerRegOk }
	| { ok: false; error: string };

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function isEmptyWeight(cell: unknown): boolean {
	if (cell === undefined || cell === null) return true;
	if (typeof cell === 'string' && cell.trim() === '') return true;
	return false;
}

function isDualSourcesOn(item: Record<string, unknown>): boolean {
	return item.dualSources === true;
}

export function flattenContainerRegFacts(
	value: Record<string, unknown>
): FlattenContainerRegResult {
	const sourceRaw = value.sourceLoadId;
	if (typeof sourceRaw !== 'string' || sourceRaw.trim() === '') {
		return { ok: false, error: 'sourceLoadId is required' };
	}
	const sourceLoadId = sourceRaw.trim();

	const secondaryRaw = value.secondarySourceLoadId;
	const formSecondary =
		typeof secondaryRaw === 'string' && secondaryRaw.trim() !== ''
			? secondaryRaw.trim()
			: undefined;

	const rows: ContainerRegRow[] = [];
	for (const { key, label } of CONTAINER_REG_MATERIALS) {
		const arr = value[key];
		if (arr === undefined || arr === null) continue;
		if (!Array.isArray(arr)) {
			return { ok: false, error: `invalid ${key} — expected array` };
		}
		for (const item of arr) {
			if (!isPlainObject(item)) {
				return { ok: false, error: `invalid row in ${label}` };
			}
			const weight = item.weight;
			if (isEmptyWeight(weight)) continue;
			if (typeof weight !== 'number' || !Number.isFinite(weight) || !(weight > 0)) {
				return { ok: false, error: `weight must be > 0 in ${label}` };
			}

			if (!isDualSourcesOn(item)) {
				rows.push({ weight, material: label });
				continue;
			}

			if (!formSecondary) {
				return {
					ok: false,
					error: 'secondarySourceLoadId is required when any row has dualSources'
				};
			}
			if (formSecondary === sourceLoadId) {
				return {
					ok: false,
					error: 'secondarySourceLoadId must differ from sourceLoadId'
				};
			}
			rows.push({ weight, material: label, secondarySourceLoadId: formSecondary });
		}
	}

	return { ok: true, facts: { sourceLoadId, rows } };
}

/** True when Substep schema pins shared sourceLoadId + fixed material array keys. */
export function isContainerRegTask(substep: unknown): boolean {
	if (substep === null || typeof substep !== 'object') return false;
	const schema = (substep as { schema?: { properties?: Record<string, unknown> } }).schema;
	const props = schema?.properties;
	if (!props || typeof props !== 'object') return false;
	if (!('sourceLoadId' in props)) return false;
	return CONTAINER_REG_MATERIALS.every(({ key }) => key in props);
}
