import {
	groupTaskPath,
	normalizeGroupInstanceId,
	PROVISIONAL_GROUP_INSTANCE_ID,
	taskPath
} from './types.js';

/** Client-only Draft bag — never part of Domain until Commit succeeds. */
export type DraftKey = string;

/** LivingSession bag entry: value plus dirty intent for marks/persist. */
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
 * Nested: `jobInstanceId::groupId::groupInstanceId` (provisional group → '').
 * Flat: jobInstanceId as-is (including '' for Start).
 */
export function draftBagInstanceKey(parts: {
	jobInstanceId: string;
	groupId?: string;
	groupInstanceId?: string;
}): string {
	const groupId = parts.groupId ?? '';
	if (!groupId) return parts.jobInstanceId;
	return `${parts.jobInstanceId}::${groupId}::${parts.groupInstanceId ?? ''}`;
}

/** Flat or nested Task address → Draft bag coords (incl. provisional `new` → ''). */
export function draftAddressForTask(parts: {
	jobId: string;
	taskId: string;
	jobInstanceId: string;
	groupId?: string;
	groupInstanceId?: string;
}): DraftAddress {
	const groupId = parts.groupId ?? '';
	if (!groupId) {
		return {
			taskPath: taskPath(parts.jobId, parts.taskId),
			draftBagInstanceKey: draftBagInstanceKey({
				jobInstanceId: parts.jobInstanceId
			})
		};
	}
	const wireGroupInstanceId =
		parts.groupInstanceId && parts.groupInstanceId.length > 0
			? parts.groupInstanceId
			: PROVISIONAL_GROUP_INSTANCE_ID;
	return {
		taskPath: groupTaskPath(
			parts.jobId,
			groupId,
			wireGroupInstanceId,
			parts.taskId
		),
		draftBagInstanceKey: draftBagInstanceKey({
			jobInstanceId: parts.jobInstanceId,
			groupId,
			groupInstanceId: normalizeGroupInstanceId(wireGroupInstanceId)
		})
	};
}
