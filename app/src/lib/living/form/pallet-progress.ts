/**
 * Plant fact on pallet: unset = available; `partial`; `done` (finished-for-processing).
 * Form Draft is separate — this module encodes the domain enum and path-A seed.
 */

import type { DomainSnapshot } from './domain-snapshot.js';

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

/** Domain progress for leftover ids — input to path-A Draft seed (not a live overlay). */
export function domainProgressForIds(
	snapshot: DomainSnapshot,
	ids: readonly string[]
): Record<string, PalletProgress> {
	const out: Record<string, PalletProgress> = {};
	for (const id of ids) {
		const progress = snapshot.pallet[id]?.progress;
		if (isPalletProgress(progress)) out[id] = progress;
	}
	return out;
}

/**
 * Path A: merge plant progress into Draft for leftover ids.
 * Keeps existing Draft cells (incl null clear); fills missing ids from domain.
 * Host seeds at Mark-progress open and after clearKeys — select paints from sjsf only.
 */
export function seedProgressDraftFromDomain(
	draftProgress: unknown,
	domain: Readonly<Record<string, PalletProgress>>,
	ids: readonly string[]
): Record<string, PalletProgress | null> {
	const existing =
		draftProgress !== null &&
		typeof draftProgress === 'object' &&
		!Array.isArray(draftProgress)
			? (draftProgress as Record<string, unknown>)
			: {};
	const out: Record<string, PalletProgress | null> = {};
	for (const id of ids) {
		if (Object.hasOwn(existing, id)) {
			const cell = existing[id];
			if (cell === null) out[id] = null;
			else if (isPalletProgress(cell)) out[id] = cell;
			continue;
		}
		if (isPalletProgress(domain[id])) out[id] = domain[id]!;
	}
	return out;
}

/**
 * After Commit clears Draft: remount seed from domain only (no stale null clears).
 * Partial stays in leftovers pool — selects must still show Parziale/Completo.
 */
export function progressDraftFromDomain(
	domain: Readonly<Record<string, PalletProgress>>,
	ids: readonly string[]
): Record<string, PalletProgress> {
	const out: Record<string, PalletProgress> = {};
	for (const id of ids) {
		if (isPalletProgress(domain[id])) out[id] = domain[id]!;
	}
	return out;
}
