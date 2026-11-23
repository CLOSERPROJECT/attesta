import type { TaskAddress } from '../task-address.js';
import type { TimelineSubstep, TimelineView } from '../timeline/types.js';
import { parseTaskHref } from './board-url.js';

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

/** Org-actionable Task addresses from the timeline projection (optional Show-all). */
export function actionableAddressesFromTimeline(
	view: TimelineView,
	opts: { showAll?: boolean } = {}
): TaskAddress[] {
	const showAll = opts.showAll ?? false;
	const out: TaskAddress[] = [];
	const seen = new Set<string>();
	for (const sub of walkSubsteps(view)) {
		if (!sub.orgActionable || !sub.href) continue;
		if (!showAll && !sub.orgActionable) continue;
		const parsed = parseTaskHref(sub.href, view.stream, view.instance);
		if (!parsed) continue;
		const key = `${parsed.address.jobId}|${parsed.address.jobInstanceId}|${parsed.address.taskId}|${parsed.address.provisionalJob}`;
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

/** Auto-open candidate: only when the board has exactly one actionable Task. */
export function soleActionableAddress(
	view: TimelineView
): TaskAddress | null {
	const addrs = actionableAddressesFromTimeline(view);
	return addrs.length === 1 ? addrs[0] : null;
}

/** True when the board already has a card for this Task (any status). */
export function addressVisibleOnTimeline(
	view: TimelineView,
	addr: TaskAddress
): boolean {
	const needle = `${addr.jobId}.${addr.taskId}`;
	for (const sub of walkSubsteps(view)) {
		if (sub.substepId === needle) return true;
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
