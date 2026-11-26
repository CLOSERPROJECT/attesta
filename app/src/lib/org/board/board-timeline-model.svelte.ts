import { createContext } from 'svelte';

import { taskAddressKey, type TaskAddress } from '../task-address.js';
import { outlineLetter } from '../timeline/outline-label.js';
import type {
	TimelineRow,
	TimelineStep,
	TimelineSubstep,
	TimelineView
} from '../timeline/types.js';
import {
	addressFromBoardTask,
	boardTaskFromAddress,
	focusMatchesAddress,
	parseTaskHref,
	type BoardTaskFocus
} from './board-url.js';
import {
	BoardEmbedOpen,
	boardEmbedOpen,
	embedBodyKey,
	type EmbedSeed
} from './board-embed-open.svelte.js';
import {
	BoardJustCompleted,
	boardJustCompleted
} from './board-just-completed.svelte.js';
import { boardChromePolicy, type BoardChromePolicy } from './board-chrome-policy.js';

export type { BoardChromePolicy };

/** Actionable timeline card with optional `?seed.*` from its href. */
export type SeededTaskRef = {
	address: TaskAddress;
	seed: EmbedSeed;
};

function seedEquals(a: EmbedSeed, b: EmbedSeed): boolean {
	const ak = Object.keys(a).filter(Boolean).sort();
	const bk = Object.keys(b).filter(Boolean).sort();
	if (ak.length !== bk.length) return false;
	return ak.every((k, i) => k === bk[i] && a[k] === b[k]);
}

export type VisibleSub = {
	sub: TimelineSubstep;
	mark: string;
	address: TaskAddress | null;
};

export type VisibleRow = {
	row: TimelineRow;
	mark: string;
	substeps: VisibleSub[];
};

export type VisibleStep = {
	step: TimelineStep;
	mark: string;
	rows?: VisibleRow[];
	substeps?: VisibleSub[];
};

export type TimelineShowFilter = {
	showAll: boolean;
	focus?: BoardTaskFocus | null;
	stream?: string;
	instance?: string;
	/** Session-sticky just-completed matcher (pin store or test double). */
	justCompleted?: { has(addr: TaskAddress): boolean } | null;
};

function walkSubsteps(view: TimelineView): TimelineSubstep[] {
	const out: TimelineSubstep[] = [];
	for (const step of view.steps) {
		for (const sub of step.substeps ?? []) {
			out.push(sub);
		}
		for (const row of step.rows ?? []) {
			for (const sub of row.substeps) {
				out.push(sub);
			}
		}
	}
	return out;
}

function addressKey(addr: TaskAddress): string {
	return `${addr.jobId}|${addr.jobInstanceId}|${addr.taskId}|${addr.provisionalJob}`;
}

function focusEquals(a: BoardTaskFocus | null, b: BoardTaskFocus | null): boolean {
	if (a == null || b == null) return a === b;
	return (
		a.jobId === b.jobId &&
		a.jobInstanceId === b.jobInstanceId &&
		a.taskId === b.taskId
	);
}

/**
 * Org-actionable cards with seed (provisional Start rows stay distinct).
 * Form preload still uses {@link actionableAddressesFromTimeline} (address dedupe).
 */
export function actionableSeededRefsFromTimeline(
	view: TimelineView,
	_opts: { showAll?: boolean } = {}
): SeededTaskRef[] {
	const out: SeededTaskRef[] = [];
	const seen = new Set<string>();
	for (const sub of walkSubsteps(view)) {
		if (!sub.orgActionable || !sub.href) continue;
		const parsed = parseTaskHref(sub.href, view.stream, view.instance);
		if (!parsed) continue;
		const key = embedBodyKey(parsed.address, parsed.seed);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push({ address: parsed.address, seed: parsed.seed });
	}
	return out;
}

/** Org-actionable Task addresses from the timeline projection (optional Show-all). */
export function actionableAddressesFromTimeline(
	view: TimelineView,
	_opts: { showAll?: boolean } = {}
): TaskAddress[] {
	const out: TaskAddress[] = [];
	const seen = new Set<string>();
	for (const sub of walkSubsteps(view)) {
		if (!sub.orgActionable || !sub.href) continue;
		const parsed = parseTaskHref(sub.href, view.stream, view.instance);
		if (!parsed) continue;
		const key = addressKey(parsed.address);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(parsed.address);
	}
	return out;
}

/** Addresses that need progressive task-recap preload (many and/or known submissions). */
export function recapAddressesFromTimeline(view: TimelineView): TaskAddress[] {
	const out: TaskAddress[] = [];
	const seen = new Set<string>();
	for (const sub of walkSubsteps(view)) {
		if (!sub.href) continue;
		const hasMany = Boolean(sub.submitMany);
		const hasSubs = (sub.submissions?.length ?? 0) > 0;
		if (!hasMany && !hasSubs) continue;
		const parsed = parseTaskHref(sub.href, view.stream, view.instance);
		if (!parsed) continue;
		const key = taskAddressKey(parsed.address);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(parsed.address);
	}
	return out;
}

export function firstActionableAddress(
	view: TimelineView
): TaskAddress | null {
	return actionableAddressesFromTimeline(view)[0] ?? null;
}

/** Auto-open candidate: exactly one actionable card (seed-distinct). */
export function soleActionableSeededRef(view: TimelineView): SeededTaskRef | null {
	const refs = actionableSeededRefsFromTimeline(view);
	return refs.length === 1 ? refs[0]! : null;
}

/** Auto-open candidate: only when the board has exactly one actionable Task. */
export function soleActionableAddress(
	view: TimelineView
): TaskAddress | null {
	return soleActionableSeededRef(view)?.address ?? null;
}

/** Whether a substep card belongs on the board under Show all / actionable-only. */
export function timelineSubstepShown(
	sub: TimelineSubstep,
	opts: TimelineShowFilter
): boolean {
	if (opts.showAll) return true;
	if (sub.orgActionable) return true;
	if (!sub.href || opts.stream == null) return false;
	// Instance-aware only — substepId is jobId.taskId and collides across lot rows.
	const parsed = parseTaskHref(sub.href, opts.stream, opts.instance ?? '');
	if (!parsed) return false;
	if (opts.focus && focusMatchesAddress(opts.focus, parsed.address)) {
		return true;
	}
	return opts.justCompleted?.has(parsed.address) === true;
}

/** True when the board already has a card for this Task (instance-scoped). */
export function addressVisibleOnTimeline(
	view: TimelineView,
	addr: TaskAddress
): boolean {
	for (const sub of walkSubsteps(view)) {
		if (!sub.href) continue;
		const parsed = parseTaskHref(sub.href, view.stream, view.instance);
		if (
			parsed &&
			parsed.address.jobId === addr.jobId &&
			parsed.address.taskId === addr.taskId &&
			parsed.address.jobInstanceId === addr.jobInstanceId
		) {
			return true;
		}
	}
	return false;
}

function parseSubAddress(
	sub: TimelineSubstep,
	stream: string,
	instance: string
): TaskAddress | null {
	if (!sub.href) return null;
	return parseTaskHref(sub.href, stream, instance)?.address ?? null;
}

/** Project the filtered outline with marks keyed to original indices (stable). */
export function projectVisibleSteps(
	view: TimelineView,
	opts: TimelineShowFilter
): VisibleStep[] {
	const stream = opts.stream ?? view.stream;
	const instance = opts.instance ?? view.instance;
	const out: VisibleStep[] = [];

	for (const step of view.steps) {
		const stepMark = step.stepId;
		const rows = step.rows ?? [];
		if (rows.length > 0) {
			const visibleRows: VisibleRow[] = [];
			for (let i = 0; i < rows.length; i++) {
				const row = rows[i]!;
				const rowMark = `${stepMark}${outlineLetter(i)}`;
				const substeps: VisibleSub[] = [];
				for (let j = 0; j < row.substeps.length; j++) {
					const sub = row.substeps[j]!;
					if (!timelineSubstepShown(sub, opts)) continue;
					substeps.push({
						sub,
						mark: `${rowMark}.${j + 1}`,
						address: parseSubAddress(sub, stream, instance)
					});
				}
				if (substeps.length === 0) continue;
				visibleRows.push({ row, mark: rowMark, substeps });
			}
			if (visibleRows.length === 0) continue;
			out.push({ step, mark: stepMark, rows: visibleRows });
			continue;
		}

		const subs = step.substeps ?? [];
		const substeps: VisibleSub[] = [];
		for (let i = 0; i < subs.length; i++) {
			const sub = subs[i]!;
			if (!timelineSubstepShown(sub, opts)) continue;
			substeps.push({
				sub,
				mark: `${stepMark}.${i + 1}`,
				address: parseSubAddress(sub, stream, instance)
			});
		}
		if (substeps.length === 0) continue;
		out.push({ step, mark: stepMark, substeps });
	}

	return out;
}

export type BoardTimelineModelOptions = {
	syncUrl: (focus: BoardTaskFocus | null) => void;
	embedOpen?: BoardEmbedOpen;
	justCompleted?: BoardJustCompleted;
};

/**
 * Board timeline visibility + selection + embed-open policy.
 * Identity is TaskAddress / BoardTaskFocus only — never substepId alone.
 * Nest collapse stays outside this module.
 */
export class BoardTimelineModel {
	#syncUrl: (focus: BoardTaskFocus | null) => void;
	#embed: BoardEmbedOpen;
	#justCompleted: BoardJustCompleted;

	view = $state.raw<TimelineView | null>(null);
	showAll = $state(false);
	stream = $state('');
	instance = $state('');
	selection = $state.raw<BoardTaskFocus | null>(null);
	/** Seed bag for the current selection (provisional accordion isolation). */
	selectionSeed = $state.raw<EmbedSeed>({});

	constructor(opts: BoardTimelineModelOptions) {
		this.#syncUrl = opts.syncUrl;
		this.#embed = opts.embedOpen ?? boardEmbedOpen;
		this.#justCompleted = opts.justCompleted ?? boardJustCompleted;
	}

	get chrome(): BoardChromePolicy {
		return boardChromePolicy(this.view?.status, this.showAll);
	}

	/** Projection: Show-all toggle or done-stream archive. */
	get effectiveShowAll(): boolean {
		return this.chrome.projectAll;
	}

	get showAllFilterVisible(): boolean {
		return !!this.view && this.chrome.showAllFilterVisible;
	}

	get visibleSteps(): VisibleStep[] {
		const view = this.view;
		if (!view) return [];
		void this.#justCompleted.get();
		return projectVisibleSteps(view, {
			showAll: this.chrome.projectAll,
			focus: this.selection,
			stream: this.stream || view.stream,
			instance: this.instance || view.instance,
			justCompleted: this.#justCompleted
		});
	}

	get actionableAddresses(): TaskAddress[] {
		const view = this.view;
		if (!view) return [];
		return actionableAddressesFromTimeline(view, { showAll: this.chrome.projectAll });
	}

	get soleActionable(): TaskAddress | null {
		const view = this.view;
		if (!view) return null;
		return soleActionableAddress(view);
	}

	/** Available-only empty: no projected cards (actionable, focus, or pin). */
	get isEmpty(): boolean {
		if (this.chrome.projectAll) return false;
		if (!this.view) return false;
		return this.visibleSteps.length === 0;
	}

	pinJustCompleted(addr: TaskAddress, submissionId?: string): void {
		this.#justCompleted.pin(addr, submissionId);
	}

	clearJustCompleted(): void {
		this.#justCompleted.clear();
	}

	isSelected(addr: TaskAddress, seed?: EmbedSeed): boolean {
		if (!focusMatchesAddress(this.selection, addr)) return false;
		if (!seed || Object.keys(seed).filter(Boolean).length === 0) return true;
		if (Object.keys(this.selectionSeed).filter(Boolean).length === 0) return true;
		return seedEquals(this.selectionSeed, seed);
	}

	isBodyOpen(addr: TaskAddress, seed?: EmbedSeed): boolean {
		void this.#embed.opened;
		return this.#embed.isOpen(addr, seed);
	}

	addressVisible(addr: TaskAddress): boolean {
		const view = this.view;
		if (!view) return false;
		return addressVisibleOnTimeline(view, addr);
	}

	/** Focused Task with no timeline card (e.g. provisional Can start). */
	get orphanFocus(): TaskAddress | null {
		const focus = this.selection;
		if (!focus) return null;
		const addr = addressFromBoardTask(this.stream, this.instance, focus);
		if (this.addressVisible(addr)) return null;
		return addr;
	}

	setView(view: TimelineView | null): void {
		this.view = view;
	}

	setShowAll(v: boolean): void {
		this.showAll = v;
	}

	setScope(stream: string, instance: string): void {
		this.stream = stream;
		this.instance = instance;
	}

	select(addr: TaskAddress, seed: EmbedSeed = {}): void {
		const next = boardTaskFromAddress(addr);
		this.#justCompleted.onFocusChange(next, this.stream, this.instance);
		this.selection = next;
		this.selectionSeed = { ...seed };
		this.#syncUrl(next);
		this.#embed.setOpen(addr, true, seed);
	}

	clearSelection(): void {
		this.#justCompleted.onFocusChange(null, this.stream, this.instance);
		this.selection = null;
		this.selectionSeed = {};
		this.#syncUrl(null);
	}

	/**
	 * URL deep-link: adopt selection from `?task=`.
	 * Opens the body only when the focus *changes* — re-running the URL sync
	 * must not undo a user collapse while the same `?task=` stays in the bar.
	 * URL has no seed, so provisional multi-lot rows stay closed until clicked.
	 */
	setFocusFromUrl(focus: BoardTaskFocus): void {
		const unchanged = focusEquals(this.selection, focus);
		this.#justCompleted.onFocusChange(focus, this.stream, this.instance);
		this.selection = focus;
		this.selectionSeed = {};
		this.#syncUrl(focus);
		if (unchanged) return;
		const addr = addressFromBoardTask(this.stream, this.instance, focus);
		this.#embed.setOpen(addr, true);
	}

	setBodyOpen(addr: TaskAddress, open: boolean, seed?: EmbedSeed): void {
		this.#embed.setOpen(addr, open, seed);
	}

	toggleBody(addr: TaskAddress, seed?: EmbedSeed): boolean {
		return this.#embed.toggle(addr, seed);
	}

	collapseAllBodies(): void {
		const view = this.view;
		if (view) {
			this.#embed.closeAll(
				actionableAddressesFromTimeline(view, { showAll: true })
			);
		}
		const orphan = this.orphanFocus;
		if (orphan) this.#embed.setOpen(orphan, false);
	}

	/**
	 * Bare load policy: sole actionable → select+open; many → no selection.
	 * Caller must skip when URL already carries focus.
	 */
	applyLoadPolicy(): void {
		const view = this.view;
		if (!view) return;
		const sole = soleActionableSeededRef(view);
		if (sole) this.select(sole.address, sole.seed);
	}
}

export const [getBoardTimelineModel, setBoardTimelineModel] =
	createContext<BoardTimelineModel>();
