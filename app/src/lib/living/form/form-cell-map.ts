/**
 * Cell-map Draft helpers: omit untouched scalar cells before Ajv / seal.
 * Enum values live on the cell schema — not table-owned paint.
 */

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

/**
 * Untouched wire for Ajv / seal: omit `""`, own-property `undefined`, and
 * non-string cells (e.g. weigh-row objects from a mis-seed).
 * Keep `null` (explicit clear) and non-empty strings (schema validates enum).
 */
export function omitUntouchedCells(
	cells: Readonly<Record<string, unknown>>
): Record<string, unknown> {
	const out: Record<string, unknown> = {};
	for (const [id, cell] of Object.entries(cells)) {
		if (cell === undefined) continue;
		if (typeof cell === 'string' && cell.trim() === '') continue;
		if (cell === null) {
			out[id] = null;
			continue;
		}
		if (typeof cell === 'string') out[id] = cell;
	}
	return out;
}

function looksLikeCellMap(value: unknown): value is Record<string, unknown> {
	if (!isPlainObject(value)) return false;
	for (const cell of Object.values(value)) {
		if (cell === null || cell === undefined) continue;
		if (typeof cell === 'string') continue;
		return false;
	}
	return true;
}

function cellTypeLabel(cell: unknown): string {
	if (cell === null) return 'null';
	if (cell === undefined) return 'undefined';
	if (typeof cell === 'string' && cell.trim() === '') return 'empty-string';
	if (typeof cell === 'string') return cell;
	if (typeof cell === 'object' && !Array.isArray(cell)) return 'object';
	return typeof cell;
}

/** Normalize cell-map-shaped maps before Ajv submit / sealSubmission. */
export function normalizeFormCellMaps<T extends Record<string, unknown>>(value: T): T {
	let changed = false;
	const next: Record<string, unknown> = { ...value };
	for (const [key, raw] of Object.entries(value)) {
		if (!looksLikeCellMap(raw)) continue;
		if (import.meta.env.DEV) {
			const types: Record<string, string> = {};
			for (const [id, cell] of Object.entries(raw)) {
				types[id] = cellTypeLabel(cell);
			}
			console.debug('[normalizeFormCellMaps] cell types', key, types);
		}
		const cleaned = omitUntouchedCells(raw);
		next[key] = cleaned;
		changed = true;
	}
	return (changed ? next : value) as T;
}
