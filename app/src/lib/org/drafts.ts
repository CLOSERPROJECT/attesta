import { taskPath } from './types.js';

/** Client-only Draft bag — never part of Domain until Commit succeeds. */
export type DraftKey = string;

/** OrgSession bag entry: value plus dirty intent for marks/persist. */
export type DraftEntry = {
	value: Record<string, unknown>;
	/** Operator/QR write; sticks until clearDraft. Seed leaves this false. */
	dirty: boolean;
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
 * Instance id passed to getDraft/setDraft/seedDraft/clearDraft/hasDirtyDraft.
 * Flat leaf: jobInstanceId as-is (including '' for Start).
 */
export function draftBagInstanceKey(parts: { jobInstanceId: string }): string {
	return parts.jobInstanceId;
}

/** Flat Task address → Draft bag coords (incl. provisional Start → ''). */
export function draftAddressForTask(parts: {
	jobId: string;
	taskId: string;
	jobInstanceId: string;
}): DraftAddress {
	return {
		taskPath: taskPath(parts.jobId, parts.taskId),
		draftBagInstanceKey: draftBagInstanceKey({
			jobInstanceId: parts.jobInstanceId
		})
	};
}
