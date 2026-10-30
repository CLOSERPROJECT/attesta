/**
 * Load-scoped FeragameFinalResults figures.
 * Pure read of DomainSnapshot — no FormPlan / Svelte.
 */

import {
	CONTAINER_REG_MATERIALS,
	type ContainerRegMaterial
} from './container-reg-materials.js';
import type { DomainSnapshot, EntityFacts, Submission } from './domain-snapshot.js';
import type { ChromeToneBand } from './chrome-tone.js';

/** Fixture `palletWeight.tareKinds` defaults (euro / industrial). */
export const DEFAULT_PALLET_TARE_KINDS: Readonly<Record<string, number>> = {
	euro: 25,
	industrial: 30
};

/** Same band spirit as C1/C7 Match % recap rows (98 / 90 / 0). */
export const FINAL_RESULTS_MATCH_BANDS: readonly ChromeToneBand[] = [
	{ min: 98, tone: 'green' },
	{ min: 90, tone: 'warning' },
	{ min: 0, tone: 'danger' }
];

export type ProjectFinalResultsOptions = {
	/** Kind → kg; defaults to euro:25 industrial:30. */
	readonly tareKinds?: Readonly<Record<string, number>>;
	readonly materials?: readonly ContainerRegMaterial[];
};

export type FinalResultsMaterialKg = {
	readonly material: string;
	readonly label: string;
	readonly weightKg: number;
};

export type FinalResultsSection = {
	readonly loadId: string;
	/** Distinct dual secondaries under this primary — muted chrome only. */
	readonly secondaryLoadIds: readonly string[];
	readonly grossTruckKg: number | null;
	readonly truckTareKg: number | null;
	readonly effectiveLoadKg: number | null;
	readonly grossPalletSumKg: number;
	readonly actualPalletSumKg: number;
	readonly totalPalletTareKg: number;
	readonly netPanelMassKg: number;
	readonly extractedByMaterial: readonly FinalResultsMaterialKg[];
	readonly totalExtractedKg: number;
	/** Sign: netPanelMassKg − totalExtractedKg. */
	readonly panelsVsExtractedDeltaKg: number;
	/** Match spirit: 100 * min/max; 0 when both sides are 0. */
	readonly panelsVsExtractedPct: number;
};

export type FinalResultsProjection = {
	readonly sections: readonly FinalResultsSection[];
};

function asFiniteNumber(value: unknown): number | null {
	return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

function asNonEmptyString(value: unknown): string | null {
	if (typeof value !== 'string') return null;
	const trimmed = value.trim();
	return trimmed.length > 0 ? trimmed : null;
}

/** Match % spirit used by C1/C7 recap rows. */
export function matchPct(a: number, b: number): number {
	const max = Math.max(a, b);
	if (max === 0) return 0;
	return (100 * Math.min(a, b)) / max;
}

/**
 * Resolve weigh-Submission tare kg. Q9: missing / non-numeric → null (caller skips).
 */
export function resolveWeighTareKg(
	facts: Record<string, unknown>,
	tareKinds: Readonly<Record<string, number>>
): number | null {
	const kind = asNonEmptyString(facts.tareKind);
	if (kind === null) return null;
	if (kind === 'custom') return asFiniteNumber(facts.customTareWeight);
	const lookedUp = tareKinds[kind];
	return asFiniteNumber(lookedUp);
}

function weighPalletCount(facts: Record<string, unknown>): number {
	const weights = facts.weights;
	if (weights === null || typeof weights !== 'object' || Array.isArray(weights)) return 0;
	return Object.keys(weights as Record<string, unknown>).length;
}

function isWeighSubmissionForLoad(submission: Submission, loadId: string): boolean {
	const taskId = submission.taskId ?? submission.substepId;
	const instanceId =
		submission.jobInstanceId ??
		(submission as { stepInstanceId?: string }).stepInstanceId;
	return taskId === 'palletWeight' && instanceId === loadId;
}

function sumPalletWeights(
	pallets: DomainSnapshot['pallet'],
	loadId: string,
	predicate: (facts: EntityFacts) => boolean
): number {
	let sum = 0;
	for (const facts of Object.values(pallets)) {
		if (facts.loadId !== loadId) continue;
		if (!predicate(facts)) continue;
		const weight = asFiniteNumber(facts.weight);
		if (weight === null) continue;
		sum += weight;
	}
	return sum;
}

function totalPalletTareForLoad(
	submissions: readonly Submission[],
	loadId: string,
	tareKinds: Readonly<Record<string, number>>
): number {
	let total = 0;
	for (const submission of submissions) {
		if (!isWeighSubmissionForLoad(submission, loadId)) continue;
		const tareKg = resolveWeighTareKg(submission.facts, tareKinds);
		if (tareKg === null) continue;
		total += tareKg * weighPalletCount(submission.facts);
	}
	return total;
}

function collectSecondaryLoadIds(
	containers: DomainSnapshot['container'],
	primaryLoadId: string
): string[] {
	const seen = new Set<string>();
	const ordered: string[] = [];
	for (const facts of Object.values(containers)) {
		if (facts.sourceLoadId !== primaryLoadId) continue;
		const secondary = asNonEmptyString(facts.secondarySourceLoadId);
		if (secondary === null || secondary === primaryLoadId) continue;
		if (seen.has(secondary)) continue;
		seen.add(secondary);
		ordered.push(secondary);
	}
	return ordered;
}

function extractedByMaterialForLoad(
	containers: DomainSnapshot['container'],
	primaryLoadId: string,
	materials: readonly ContainerRegMaterial[]
): { rows: FinalResultsMaterialKg[]; totalKg: number } {
	const byKey = new Map<string, number>();
	for (const facts of Object.values(containers)) {
		if (facts.sourceLoadId !== primaryLoadId) continue;
		const weight = asFiniteNumber(facts.weight);
		if (weight === null) continue;
		const material = asNonEmptyString(facts.material) ?? '';
		byKey.set(material, (byKey.get(material) ?? 0) + weight);
	}

	const labelByKey = new Map(materials.map((m) => [m.key, m.label]));
	const rows: FinalResultsMaterialKg[] = [];
	for (const material of materials) {
		const weightKg = byKey.get(material.key);
		if (weightKg === undefined) continue;
		rows.push({ material: material.key, label: material.label, weightKg });
		byKey.delete(material.key);
	}
	const extras = [...byKey.keys()].sort();
	for (const material of extras) {
		rows.push({
			material,
			label: labelByKey.get(material) ?? material,
			weightKg: byKey.get(material)!
		});
	}

	let totalKg = 0;
	for (const row of rows) totalKg += row.weightKg;
	return { rows, totalKg };
}

function projectSection(
	snapshot: DomainSnapshot,
	loadId: string,
	tareKinds: Readonly<Record<string, number>>,
	materials: readonly ContainerRegMaterial[]
): FinalResultsSection {
	const load = snapshot.load[loadId];
	const grossTruckKg = asFiniteNumber(load?.truckWeight);
	const truckTareKg = asFiniteNumber(load?.truckTare);
	const effectiveLoadKg =
		grossTruckKg !== null && truckTareKg !== null ? grossTruckKg - truckTareKg : null;

	const grossPalletSumKg = sumPalletWeights(snapshot.pallet, loadId, () => true);
	const actualPalletSumKg = sumPalletWeights(
		snapshot.pallet,
		loadId,
		(facts) => facts.progress === 'done'
	);
	const totalPalletTareKg = totalPalletTareForLoad(snapshot.submissions, loadId, tareKinds);
	const netPanelMassKg = grossPalletSumKg - totalPalletTareKg;

	const { rows: extractedByMaterial, totalKg: totalExtractedKg } = extractedByMaterialForLoad(
		snapshot.container,
		loadId,
		materials
	);

	return {
		loadId,
		secondaryLoadIds: collectSecondaryLoadIds(snapshot.container, loadId),
		grossTruckKg,
		truckTareKg,
		effectiveLoadKg,
		grossPalletSumKg,
		actualPalletSumKg,
		totalPalletTareKg,
		netPanelMassKg,
		extractedByMaterial,
		totalExtractedKg,
		panelsVsExtractedDeltaKg: netPanelMassKg - totalExtractedKg,
		panelsVsExtractedPct: matchPct(netPanelMassKg, totalExtractedKg)
	};
}

/**
 * One section per distinct primary `container.sourceLoadId`.
 * Dual weight stays on the primary; secondary ids are muted list only.
 */
export function projectFinalResults(
	snapshot: DomainSnapshot,
	options?: ProjectFinalResultsOptions
): FinalResultsProjection {
	const tareKinds = options?.tareKinds ?? DEFAULT_PALLET_TARE_KINDS;
	const materials = options?.materials ?? CONTAINER_REG_MATERIALS;

	const primaryIds = new Set<string>();
	for (const facts of Object.values(snapshot.container)) {
		const sourceLoadId = asNonEmptyString(facts.sourceLoadId);
		if (sourceLoadId !== null) primaryIds.add(sourceLoadId);
	}

	const sections = [...primaryIds]
		.sort()
		.map((loadId) => projectSection(snapshot, loadId, tareKinds, materials));

	return { sections };
}
