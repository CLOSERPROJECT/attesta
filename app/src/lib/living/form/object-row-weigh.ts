/**
 * Object-row weigh Draft shape helpers for map-table UI.
 * Commit-prep twins (filterFilled / flatten) live in Go; this module only renders.
 */

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
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
