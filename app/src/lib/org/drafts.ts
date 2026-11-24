import { taskPath } from './types.js';

/** Client-only Draft bag — never part of Domain until Commit succeeds. */
export type DraftKey = string;

/** OrgSession bag entry: value plus dirty intent for marks/persist. */
export type DraftEntry = {
	value: Record<string, unknown>;
	/** Operator/QR write; sticks until clearDraft. Seed leaves this false. */
	dirty: boolean;
	/** Field names last written by URL/QR seed for this bag (operator-locked). */
	seededFields?: readonly string[];
};

/** Address for hasDirtyDraft / getDraft mark binding (projections stay pure). */
export type DraftAddress = {
	taskPath: string;
	draftBagInstanceKey: string;
};

export function draftKey(taskPath: string, jobInstanceId: string): DraftKey {
	return `${taskPath}::${jobInstanceId || '_new'}`;
}

/**
 * Stable provisional bag key from URL/QR seed so accordion L1 cannot share
 * the empty-instance bag with unseeded start or another row's seed.
 */
export function provisionalSeedDraftKey(seed: Record<string, string>): string {
	const keys = Object.keys(seed).filter(Boolean).sort();
	if (keys.length === 0) return '';
	return `seed:${keys.map((k) => `${encodeURIComponent(k)}=${encodeURIComponent(seed[k]!)}`).join('&')}`;
}

/**
 * Instance id passed to getDraft/setDraft/seedDraft/clearDraft/hasDirtyDraft.
 * Flat leaf: jobInstanceId as-is. Provisional + seed → seed-derived key; else ''.
 */
export function draftBagInstanceKey(parts: {
	jobInstanceId: string;
	seed?: Record<string, string>;
}): string {
	if (parts.jobInstanceId) return parts.jobInstanceId;
	if (parts.seed && Object.keys(parts.seed).length > 0) {
		return provisionalSeedDraftKey(parts.seed);
	}
	return '';
}

/** Flat Task address → Draft bag coords (incl. provisional Start → ''). */
export function draftAddressForTask(parts: {
	jobId: string;
	taskId: string;
	jobInstanceId: string;
	seed?: Record<string, string>;
}): DraftAddress {
	return {
		taskPath: taskPath(parts.jobId, parts.taskId),
		draftBagInstanceKey: draftBagInstanceKey({
			jobInstanceId: parts.jobInstanceId,
			seed: parts.seed
		})
	};
}
