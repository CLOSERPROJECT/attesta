import type { FailureValidationResult } from '@sjsf/form';
import { untrack } from 'svelte';
import type {
	ChromeDataReportSectionDisplay,
	ChromeDisplay,
	ChromeRecapRowDisplay
} from './chrome-context.js';
import { isChromeTone, resolveChromeTone } from './chrome-tone.js';
import { evaluateSafe, formatJsonataError } from './jsonata-safe.js';
import type { OperatorIdentity } from './operator-identity.js';
import type { ChromeDataReport, ChromeRecap, TaskFormPlan } from './task-form-plan.js';

export type LiveTaskFormPorts = {
	getValue: () => Record<string, unknown>;
	setValue: (next: Record<string, unknown>) => void;
	/** JSONata `$domain` binding (recap / valuePrint / derived) — may be a thin projection. */
	getDomain: () => unknown;
	/** Server `bindings.chromeDisplay` bag for `kind: dataReport` (Load/Commit only). */
	getChromeDisplayBag?: () => Record<string, unknown> | undefined;
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

function normalizeRow(raw: unknown): ChromeRecapRowDisplay | null {
	if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) return null;
	const rec = raw as Record<string, unknown>;
	if (typeof rec.label !== 'string') return null;
	return {
		label: rec.label,
		value: rec.value == null ? null : String(rec.value),
		error: rec.error == null ? null : String(rec.error),
		tone: isChromeTone(rec.tone) ? rec.tone : null
	};
}

function normalizeSection(raw: unknown): ChromeDataReportSectionDisplay | null {
	if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) return null;
	const rec = raw as Record<string, unknown>;
	if (typeof rec.loadId !== 'string') return null;
	const secondary =
		Array.isArray(rec.secondaryLoadIds) &&
		rec.secondaryLoadIds.every((id) => typeof id === 'string')
			? (rec.secondaryLoadIds as string[])
			: undefined;
	const rows = Array.isArray(rec.rows)
		? rec.rows.map(normalizeRow).filter((row): row is ChromeRecapRowDisplay => row != null)
		: undefined;
	return {
		loadId: rec.loadId,
		...(secondary !== undefined ? { secondaryLoadIds: secondary } : {}),
		...(rows !== undefined ? { rows } : {})
	};
}

/** Paint-only: map server `chromeDisplay[fieldId]` DTO → ChromeDisplay. */
export function displayFromChromeDisplayBag(
	bag: Record<string, unknown> | undefined,
	entry: ChromeDataReport
): ChromeDisplay {
	const fallbackTitle = entry.title ?? null;
	const raw = bag?.[entry.id];
	if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) {
		return {
			value: null,
			error: 'chrome-display-missing',
			title: fallbackTitle,
			sections: []
		};
	}
	const rec = raw as Record<string, unknown>;
	const sections = Array.isArray(rec.sections)
		? rec.sections
				.map(normalizeSection)
				.filter((s): s is ChromeDataReportSectionDisplay => s != null)
		: [];
	const title =
		typeof rec.title === 'string'
			? rec.title
			: rec.title === null
				? null
				: fallbackTitle;
	return {
		value: rec.value == null ? null : String(rec.value),
		error: rec.error == null ? null : String(rec.error),
		title,
		sections
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
			void JSON.stringify(ports.getChromeDisplayBag?.() ?? null);
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
	const chromeDisplayBag = untrack(() => ports.getChromeDisplayBag?.());
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
		} else if (entry.kind === 'dataReport') {
			next[entry.id] = displayFromChromeDisplayBag(chromeDisplayBag, entry);
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
