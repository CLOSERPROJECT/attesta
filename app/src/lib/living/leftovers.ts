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

export type WeighRowDraft = {
	weight: number | '';
	tareKind: string;
	idoneita: string;
	panelCount: number | '';
};

export const TARE_KG: Record<string, number> = {
	euro: 25,
	industrial: 30
};

/** Floor((weight − tare) / averagePanelWeight); empty inputs → ''. */
export function derivedPanelCount(args: {
	weight: number | '';
	tareKind: string;
	customTareWeight: number | '';
	averagePanelWeight: number | '';
}): number | '' {
	const { weight, tareKind, customTareWeight, averagePanelWeight } = args;
	if (typeof weight !== 'number' || typeof averagePanelWeight !== 'number') return '';
	if (!(averagePanelWeight > 0)) return '';
	const tare =
		tareKind === 'custom'
			? typeof customTareWeight === 'number'
				? customTareWeight
				: NaN
			: (TARE_KG[tareKind] ?? NaN);
	if (!Number.isFinite(tare)) return '';
	return Math.floor((weight - tare) / averagePanelWeight);
}

export function emptyWeighRow(defaults?: {
	tareKind?: string;
	idoneita?: string;
}): WeighRowDraft {
	return {
		weight: '',
		tareKind: defaults?.tareKind ?? 'euro',
		idoneita: defaults?.idoneita ?? 'NIR',
		panelCount: ''
	};
}
