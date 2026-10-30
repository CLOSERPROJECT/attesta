/**
 * Plant fact on pallet: unset = available; `partial`; `done` (finished-for-processing).
 * Form Draft is separate — this module encodes the domain enum and path-A seed.
 */

import type { DomainSnapshot } from './domain-snapshot.js';

export type PalletProgress = 'partial' | 'done';

export function isPalletProgress(value: unknown): value is PalletProgress {
	return value === 'partial' || value === 'done';
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
