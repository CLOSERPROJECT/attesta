import type { FailureValidationResult } from '@sjsf/form';
import { untrack } from 'svelte';
import type {
	ChromeDisplay,
	ChromeFinalResultsSectionDisplay,
	ChromeRecapRowDisplay
} from './chrome-context.js';
import { resolveChromeTone } from './chrome-tone.js';
import type { DomainSnapshot } from './domain-snapshot.js';
import { evaluateSafe, formatJsonataError } from './jsonata-safe.js';
import type { OperatorIdentity } from './operator-identity.js';
import {
	FINAL_RESULTS_MATCH_BANDS,
	projectFinalResults,
	type FinalResultsSection
} from './project-final-results.js';
import type { ChromeFinalResults, ChromeRecap, TaskFormPlan } from './task-form-plan.js';

export type LiveTaskFormPorts = {
	getValue: () => Record<string, unknown>;
	setValue: (next: Record<string, unknown>) => void;
	/** JSONata `$domain` binding (recap / valuePrint / derived) — may be a thin projection. */
	getDomain: () => unknown;
	/**
	 * Full DomainSnapshot for `kind: finalResults` projector.
	 * Separate from getDomain so host JSONata bindings cannot starve the projector.
	 */
	getSnapshot?: () => unknown;
	getInstance?: () => unknown;
	/** Chrome eval `$operator` — not readyWhen. */
	getOperator?: () => OperatorIdentity;
	onSubmit?: (value: Record<string, unknown>) => void | Promise<void>;
	onSubmitError?: (result: FailureValidationResult) => void;
};

const EMPTY_DISPLAY: ChromeDisplay = { value: null, error: null };

/** Stringify chrome expr results for valuePrint / readonly (display-only). */
export function formatChromeEvalValue(value: unknown): ChromeDisplay {
	if (value === undefined || (typeof value === 'number' && !Number.isFinite(value))) {
		return { value: null, error: 'chrome-failed' };
	}
	return { value: value == null ? null : String(value), error: null };
}

function chromeBindings(
	domain: unknown,
	instance: unknown | undefined,
	operator: OperatorIdentity | undefined
): Record<string, unknown> {
	const bindings: Record<string, unknown> = { domain };
	if (instance !== undefined) {
		bindings.instance = instance;
	}
	if (operator !== undefined) {
		bindings.operator = operator;
	}
	return bindings;
}

export async function evaluateChromeEntry(
	expr: string,
	sealableDraft: Record<string, unknown>,
	domain: unknown,
	instance: unknown | undefined,
	operator?: OperatorIdentity
): Promise<ChromeDisplay> {
	const result = await evaluateSafe(
		expr,
		sealableDraft,
		chromeBindings(domain, instance, operator)
	);
	if (!result.ok) {
		return { value: null, error: formatJsonataError(result.error) };
	}
	return formatChromeEvalValue(result.value);
}

export async function evaluateRecapEntry(
	entry: ChromeRecap,
	sealableDraft: Record<string, unknown>,
	domain: unknown,
	instance: unknown | undefined,
	operator?: OperatorIdentity
): Promise<ChromeDisplay> {
	const bindings = chromeBindings(domain, instance, operator);

	const rows: ChromeRecapRowDisplay[] = [];
	for (const row of entry.rows) {
		const result = await evaluateSafe(row.expr, sealableDraft, bindings);
		if (!result.ok) {
			rows.push({
				label: row.label,
				value: null,
				error: formatJsonataError(result.error),
				tone: null
			});
			continue;
		}
		const formatted = formatChromeEvalValue(result.value);
		rows.push({
			label: row.label,
			value: formatted.value,
			error: formatted.error,
			tone: resolveChromeTone(result.value, { tone: row.tone, bands: row.bands })
		});
	}

	return {
		value: null,
		error: null,
		title: entry.title ?? null,
		rows
	};
}

function isDomainSnapshot(value: unknown): value is DomainSnapshot {
	if (value === null || typeof value !== 'object' || Array.isArray(value)) return false;
	const rec = value as Record<string, unknown>;
	return (
		rec.load !== null &&
		typeof rec.load === 'object' &&
		!Array.isArray(rec.load) &&
		rec.pallet !== null &&
		typeof rec.pallet === 'object' &&
		!Array.isArray(rec.pallet) &&
		rec.container !== null &&
		typeof rec.container === 'object' &&
		!Array.isArray(rec.container) &&
		Array.isArray(rec.submissions)
	);
}

function formatKg(value: number | null): string | null {
	return value === null ? null : String(value);
}

function kgRow(label: string, value: number | null): ChromeRecapRowDisplay {
	return { label, value: formatKg(value), error: null, tone: null };
}

/** Map projector section → chrome display rows (Destination C7 recipe). */
export function finalResultsSectionDisplay(
	section: FinalResultsSection
): ChromeFinalResultsSectionDisplay {
	const rows: ChromeRecapRowDisplay[] = [
		kgRow('Camion lordo', section.grossTruckKg),
		kgRow('Tara camion', section.truckTareKg),
		kgRow('Carico netto', section.effectiveLoadKg),
		kgRow('Σ bancali lordi', section.grossPalletSumKg),
		kgRow('Σ bancali effettivi (fatti)', section.actualPalletSumKg),
		kgRow('Tara totale bancali', section.totalPalletTareKg),
		kgRow('Massa netta pannelli', section.netPanelMassKg)
	];
	for (const material of section.extractedByMaterial) {
		rows.push({
			label: material.label,
			value: String(material.weightKg),
			error: null,
			tone: 'info'
		});
	}
	rows.push(kgRow('Totale estratto', section.totalExtractedKg));
	rows.push({
		label: 'Pannelli vs estratto',
		value: `${section.panelsVsExtractedDeltaKg} kg (${section.panelsVsExtractedPct}%)`,
		error: null,
		tone: resolveChromeTone(section.panelsVsExtractedPct, {
			bands: FINAL_RESULTS_MATCH_BANDS
		})
	});
	return {
		loadId: section.loadId,
		secondaryLoadIds: section.secondaryLoadIds,
		rows
	};
}

/** Project DomainSnapshot into finalResults chrome display. */
export async function evaluateFinalResultsEntry(
	entry: ChromeFinalResults,
	_sealableDraft: Record<string, unknown>,
	snapshot: unknown,
	_instance: unknown | undefined,
	_operator?: OperatorIdentity
): Promise<ChromeDisplay> {
	if (!isDomainSnapshot(snapshot)) {
		return {
			value: null,
			error: 'final-results-needs-snapshot',
			title: entry.title ?? null,
			sections: []
		};
	}
	const { sections } = projectFinalResults(snapshot);
	return {
		value: null,
		error: null,
		title: entry.title ?? null,
		sections: sections.map(finalResultsSectionDisplay)
	};
}

export type LiveTaskChromeHandle = {
	chromeDisplay: (id: string) => ChromeDisplay;
	whenSettled: () => Promise<void>;
	dispose: () => void;
};

export function startLiveTaskChrome(
	plan: TaskFormPlan,
	ports: LiveTaskFormPorts
): LiveTaskChromeHandle {
	let displays = $state<Record<string, ChromeDisplay>>({});
	let generation = 0;
	let pending: Promise<void> = Promise.resolve();
	const disposers: (() => void)[] = [];

	const schedule = (): void => {
		const gen = ++generation;
		pending = applyChrome(plan, ports, () => gen === generation, (next) => {
			displays = next;
		});
	};

	const stop = $effect.root(() => {
		$effect(() => {
			void JSON.stringify(ports.getValue());
			void ports.getDomain();
			void ports.getSnapshot?.();
			void ports.getInstance?.();
			void ports.getOperator?.();
			schedule();
		});
	});
	disposers.push(stop);
	schedule();

	return {
		chromeDisplay: (id: string) => displays[id] ?? EMPTY_DISPLAY,
		whenSettled: () => pending,
		dispose: () => {
			for (const d of disposers) d();
			disposers.length = 0;
			generation += 1;
		}
	};
}

async function applyChrome(
	plan: TaskFormPlan,
	ports: LiveTaskFormPorts,
	isCurrent: () => boolean,
	commit: (next: Record<string, ChromeDisplay>) => void
): Promise<void> {
	if (plan.chrome.length === 0) {
		commit({});
		return;
	}

	const sealableDraft = untrack(() => plan.sealSubmission(ports.getValue()));
	const domain = untrack(() => ports.getDomain());
	const snapshot = untrack(() => ports.getSnapshot?.());
	const instance = untrack(() => ports.getInstance?.());
	const operator = untrack(() => ports.getOperator?.());

	const next: Record<string, ChromeDisplay> = {};
	for (const entry of plan.chrome) {
		if (!isCurrent()) return;
		if (entry.kind === 'recap') {
			next[entry.id] = await evaluateRecapEntry(
				entry,
				sealableDraft,
				domain,
				instance,
				operator
			);
		} else if (entry.kind === 'finalResults') {
			next[entry.id] = await evaluateFinalResultsEntry(
				entry,
				sealableDraft,
				snapshot,
				instance,
				operator
			);
		} else {
			next[entry.id] = await evaluateChromeEntry(
				entry.expr,
				sealableDraft,
				domain,
				instance,
				operator
			);
		}
	}
	if (!isCurrent()) return;
	commit(next);
}
