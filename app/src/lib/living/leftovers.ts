import type { Domain } from './types.js';

/** Pallets for this intake Job instance that still lack weight (completeness all). */
export function palletWeightLeftoverIds(
	domain: Domain | null | undefined,
	jobInstanceId: string
): string[] {
	if (!domain?.pallet) return [];
	const ids: string[] = [];
	for (const [id, facts] of Object.entries(domain.pallet)) {
		if (facts?.loadId !== jobInstanceId) continue;
		if (facts.weight !== undefined && facts.weight !== null) continue;
		ids.push(id);
	}
	ids.sort();
	return ids;
}

function palletWeightExists(facts: Record<string, unknown> | undefined): boolean {
	if (!facts) return false;
	const w = facts.weight;
	return typeof w === 'number' && Number.isFinite(w);
}

/**
 * Weighted pallets not yet progress=done (lineLoad leftovers completeness some).
 * Shared pool across line Job instances — not scoped to a shift id.
 */
export function lineLoadLeftoverIds(domain: Domain | null | undefined): string[] {
	if (!domain?.pallet) return [];
	const ids: string[] = [];
	for (const [id, facts] of Object.entries(domain.pallet)) {
		if (!palletWeightExists(facts)) continue;
		if (facts.progress === 'done') continue;
		ids.push(id);
	}
	ids.sort();
	return ids;
}
