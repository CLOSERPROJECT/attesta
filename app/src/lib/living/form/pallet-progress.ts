/**
 * Plant fact on pallet: unset = available; `partial`; `done` (finished-for-processing).
 * Form Draft seed comes from the Task form bundle `initialData` — not a client Domain re-seed.
 */

export type PalletProgress = 'partial' | 'done';

export function isPalletProgress(value: unknown): value is PalletProgress {
	return value === 'partial' || value === 'done';
}

/**
 * Untouched wire for Ajv / seal: omit `""`, own-property `undefined`, and
 * non-progress values (e.g. weigh-row objects from a mis-seed).
 * Keep `null` (explicit clear) and `partial` | `done`. Do not widen schema.
 */
export function omitUntouchedProgressCells(
	progress: Readonly<Record<string, unknown>>
): Record<string, unknown> {
	const out: Record<string, unknown> = {};
	for (const [id, cell] of Object.entries(progress)) {
		if (cell === undefined) continue;
		if (typeof cell === 'string' && cell.trim() === '') continue;
		if (cell === null || isPalletProgress(cell)) out[id] = cell;
	}
	return out;
}

function progressCellTypeLabel(cell: unknown): string {
	if (cell === null) return 'null';
	if (cell === undefined) return 'undefined';
	if (isPalletProgress(cell)) return cell;
	if (typeof cell === 'string' && cell.trim() === '') return 'empty-string';
	if (typeof cell === 'object' && !Array.isArray(cell)) return 'object';
	return typeof cell;
}

/** Normalize form `progress` map before Ajv submit / sealSubmission. */
export function normalizeFormProgress<T extends Record<string, unknown>>(value: T): T {
	if (!Object.hasOwn(value, 'progress')) return value;
	const progress = value.progress;
	if (progress === null || typeof progress !== 'object' || Array.isArray(progress)) {
		return value;
	}
	const record = progress as Record<string, unknown>;
	if (import.meta.env.DEV) {
		const types: Record<string, string> = {};
		for (const [id, cell] of Object.entries(record)) {
			types[id] = progressCellTypeLabel(cell);
		}
		console.debug('[normalizeFormProgress] cell types', types);
	}
	return {
		...value,
		progress: omitUntouchedProgressCells(record)
	};
}
